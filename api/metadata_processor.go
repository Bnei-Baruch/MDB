package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"gopkg.in/volatiletech/null.v6"

	"github.com/Bnei-Baruch/mdb/common"
	"github.com/Bnei-Baruch/mdb/events"
	"github.com/Bnei-Baruch/mdb/models"
	"github.com/Bnei-Baruch/mdb/utils"
)

func ProcessCITMetadata(exec boil.Executor, metadata CITMetadata, original, proxy *models.File) ([]events.Event, error) {
	return doProcess(exec, metadata, original, proxy, nil)
}

// Do all stuff for processing metadata coming from Content Identification Tool.
// 	1. Update properties for original and proxy (film_date, capture_date)
//	2. Update language of original
// 	3. Create content_unit (content_type, dates)
// 	4. Describe content unit (i18ns)
//	5. Add files to new unit
// 	6. Add ancestor files to unit
//  7. Add peer ancestor (related captures)
// 	8. Associate unit with sources, tags, and persons
// 	9. Get or create collection
// 	10. Update collection (content_type, dates, number) if full lesson or new lesson
// 	11. Associate collection and unit
// 	12. Associate unit and derived units
// 	13. Set default permissions ?!
func doProcess(exec boil.Executor, metadata CITMetadata, original, proxy *models.File, cu *models.ContentUnit) ([]events.Event, error) {
	isUpdate := cu != nil
	log.Infof("Processing CITMetadata, isUpdate: %t", isUpdate)

	// Update properties for original and proxy (film_date, capture_date)
	filmDate := metadata.CaptureDate
	//if metadata.WeekDate != nil {
	//	filmDate = *metadata.WeekDate
	//}
	if metadata.FilmDate != nil {
		filmDate = *metadata.FilmDate
	}

	evnts := make([]events.Event, 0)

	props := map[string]interface{}{
		"capture_date":      metadata.CaptureDate,
		"film_date":         filmDate,
		"original_language": common.StdLang(metadata.Language),
	}
	log.Infof("Updating files properties: %v", props)
	err := UpdateFileProperties(exec, original, props)
	if err != nil {
		return nil, err
	}
	evnts = append(evnts, events.FileUpdateEvent(original))
	if proxy != nil {
		err = UpdateFileProperties(exec, proxy, props)
		if err != nil {
			return nil, err
		}
		evnts = append(evnts, events.FileUpdateEvent(proxy))
	}

	// Update language of original.
	// TODO: What about proxy !?
	if metadata.HasTranslation {
		original.Language = null.StringFrom(common.LANG_MULTI)
	} else {
		l := common.StdLang(metadata.Language)
		if l == common.LANG_UNKNOWN {
			log.Warnf("Unknown language in metadata %s", metadata.Language)
		}
		original.Language = null.StringFrom(l)
	}
	log.Infof("Updating original.Language to %s", original.Language.String)
	err = original.Update(exec, "language")
	if err != nil {
		return nil, errors.Wrap(err, "Save original to DB")
	}

	// Create content_unit (content_type, dates)
	isDerived := metadata.ArtifactType.Valid && metadata.ArtifactType.String != "main"
	ct := metadata.ContentType
	if isDerived {
		// User input is verified below
		ct = strings.ToUpper(metadata.ArtifactType.String)
	}

	var originalProps map[string]interface{}
	err = original.Properties.Unmarshal(&originalProps)
	if err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal original properties")
	}
	if duration, ok := originalProps["duration"]; ok {
		props["duration"] = int(duration.(float64))
	} else {
		log.Warnf("Original is missing duration property [%d]", original.ID)
	}

	if metadata.LabelID.Valid {
		props["label_id"] = metadata.LabelID.Int
	}
	if metadata.Number.Valid {
		props["number"] = metadata.Number.Int
	}
	if metadata.Part.Valid {
		props["part"] = metadata.Part.Int
	}

	if isUpdate {
		// content_type
		if ctVal, ok := common.CONTENT_TYPE_REGISTRY.ByName[ct]; !ok {
			return nil, errors.Errorf("Unknown content type %s", ct)
		} else if ctVal.ID != cu.TypeID {
			// update unit's content type
			cu.TypeID = ctVal.ID
			err = cu.Update(exec, "type_id")
			if err != nil {
				return nil, errors.Wrap(err, "Update unit type in DB")
			}
		}

		// props
		propsBytes, err := json.Marshal(props)
		if err != nil {
			return nil, errors.Wrap(err, "json Marshal")
		}
		cu.Properties = null.JSONFrom(propsBytes)
		err = cu.Update(exec, "properties")
		if err != nil {
			return nil, errors.Wrap(err, "Update unit properties in DB")
		}
	} else {
		log.Infof("Creating content unit of type %s", ct)
		cu, err = CreateContentUnit(exec, ct, props)
		if err != nil {
			return nil, errors.Wrap(err, "Create content unit")
		}
		evnts = append(evnts, events.ContentUnitCreateEvent(cu))

		log.Infof("Describing content unit [%d]", cu.ID)
		err = DescribeContentUnit(exec, cu, metadata)
		if err != nil {
			log.Errorf("Error describing content unit: %s", err.Error())
		}
	}

	// we lookup Original's capture_stop operation as it holds required information below
	var captureStopProps map[string]interface{}
	captureStop, err := FindUpChainOperation(exec, original.ID, common.OP_CAPTURE_STOP)
	if err != nil {
		if ex, ok := err.(UpChainOperationNotFound); ok {
			log.Warnf("capture_stop operation not found for original: %s", ex.Error())
		}
	} else {
		if captureStop.Properties.Valid {
			err = json.Unmarshal(captureStop.Properties.JSON, &captureStopProps)
			if err != nil {
				return nil, errors.Wrap(err, "json Unmarshal")
			}
		}
	}

	// Add files to new unit
	log.Info("Adding files to unit")
	err = cu.AddFiles(exec, false, original)
	if err != nil {
		return nil, errors.Wrap(err, "Add original to unit")
	}
	if proxy != nil {
		err = cu.AddFiles(exec, false, proxy)
		if err != nil {
			return nil, errors.Wrap(err, "Add proxy to unit")
		}
	}

	// Add ancestor files to unit (not for derived units)
	if !isDerived && !isUpdate {
		log.Info("Main unit, adding ancestors...")
		ancestors, err := FindFileAncestors(exec, original.ID)
		if err != nil {
			return nil, errors.Wrap(err, "Find original's ancestors")
		}

		if proxy != nil {
			err = proxy.L.LoadParent(exec, true, proxy)
			if err != nil {
				return nil, errors.Wrap(err, "Load proxy's parent")
			}
			if proxy.R.Parent != nil {
				ancestors = append(ancestors, proxy.R.Parent)
			}
		}

		err = cu.AddFiles(exec, false, ancestors...)
		if err != nil {
			return nil, errors.Wrap(err, "Add ancestors to unit")
		}
		log.Infof("Added %d ancestors", len(ancestors))
		for i := range ancestors {
			x := ancestors[i]
			evnts = append(evnts, events.FileUpdateEvent(x))
			log.Infof("%s [%d]", x.Name, x.ID)
		}

		// Add peer ancestor (related captures)
		if workflowID, ok := captureStopProps["workflow_id"]; ok {
			// find other captures
			relatedCaptures, err := FindOperationsByWorkflowID(exec, workflowID, common.OP_CAPTURE_STOP)
			if err != nil {
				return nil, errors.Wrap(err, "find related captures")
			}

			// find other captures files and add them all to this unit
			var relatedCapturesFiles []*models.File
			for _, capture := range relatedCaptures {
				if capture.ID == captureStop.ID {
					continue
				}
				if err := capture.L.LoadFiles(exec, true, capture); err != nil {
					return nil, errors.Wrapf(err, "load related capture files %d", capture.ID)
				}
				captureFile := capture.R.Files[0]
				relatedCapturesFiles = append(relatedCapturesFiles, captureFile)
				if files, err := FindFileDescendants(exec, captureFile.ID); err != nil {
					return nil, errors.Wrapf(err, "load descendants of related capture file %d", captureFile.ID)
				} else {
					relatedCapturesFiles = append(relatedCapturesFiles, files...)
				}
			}

			err = cu.AddFiles(exec, false, relatedCapturesFiles...)
			if err != nil {
				return nil, errors.Wrap(err, "Add related capture files to unit")
			}
			log.Infof("Added %d related capture files", len(relatedCapturesFiles))
			for _, f := range relatedCapturesFiles {
				evnts = append(evnts, events.FileUpdateEvent(f))
				log.Infof("%s [%d]", f.Name, f.ID)
			}
		} else {
			log.Info("capture_stop not found or its missing workflow_id. Skipping related captures associations")
		}
	}

	// Associate unit with sources, tags, and persons
	if len(metadata.Sources) > 0 {
		log.Infof("Associating %d sources", len(metadata.Sources))
		sources, err := models.Sources(exec,
			qm.WhereIn("uid in ?", utils.ConvertArgsString(metadata.Sources)...)).
			All()
		if err != nil {
			return nil, errors.Wrap(err, "Lookup sources in DB")
		}

		// are we missing some source ?
		if len(sources) != len(metadata.Sources) {
			missing := make([]string, 0)
			for _, x := range metadata.Sources {
				found := false
				for _, y := range sources {
					if x == y.UID {
						found = true
						break
					}
				}
				if !found {
					missing = append(missing, x)
				}
			}
			log.Warnf("Unknown sources: %s", missing)
		}

		err = cu.SetSources(exec, false, sources...)
		if err != nil {
			return nil, errors.Wrap(err, "Associate sources")
		}
	}

	if len(metadata.Tags) > 0 {
		log.Infof("Associating %d tags", len(metadata.Tags))
		tags, err := models.Tags(exec,
			qm.WhereIn("uid in ?", utils.ConvertArgsString(metadata.Tags)...)).
			All()
		if err != nil {
			return nil, errors.Wrap(err, "Lookup tags  in DB")
		}

		// are we missing some tag ?
		if len(tags) != len(metadata.Tags) {
			missing := make([]string, 0)
			for _, x := range metadata.Tags {
				found := false
				for _, y := range tags {
					if x == y.UID {
						found = true
						break
					}
				}
				if !found {
					missing = append(missing, x)
				}
			}
			log.Warnf("Unknown sources: %s", missing)
		}
		err = cu.SetTags(exec, false, tags...)
		if err != nil {
			return nil, errors.Wrap(err, "Associate tags")
		}
	}
	if len(metadata.Likutim) > 0 {
		log.Infof("Associating %d likutim", len(metadata.Likutim))
		likutim, err := models.ContentUnits(exec,
			qm.Select("distinct on (\"content_units\".id) \"content_units\".*"),
			qm.WhereIn("uid in ?", utils.ConvertArgsString(metadata.Likutim)...)).
			All()
		if err != nil {
			return nil, errors.Wrap(err, "Lookup tags  in DB")
		}

		// are we missing some unit ?
		if len(likutim) != len(metadata.Likutim) {
			missing := make([]string, 0)
			for _, x := range metadata.Likutim {
				found := false
				for _, y := range likutim {
					if x == y.UID {
						found = true
						break
					}
				}
				if !found {
					missing = append(missing, x)
				}
			}
			log.Warnf("Unknown likutim: %s", missing)
		}

		derivations := make([]*models.ContentUnitDerivation, len(likutim))
		for i, l := range likutim {
			cud := &models.ContentUnitDerivation{
				SourceID:  cu.ID,
				DerivedID: l.ID,
			}
			derivations[i] = cud
		}
		err = cu.AddSourceContentUnitDerivations(exec, true, derivations...)
		if err != nil {
			return nil, errors.Wrap(err, "Associate likutim")
		}
		for _, l := range likutim {
			evnts = append(evnts, events.ContentUnitDerivativesChangeEvent(l))
		}
		evnts = append(evnts, events.ContentUnitDerivativesChangeEvent(cu))
	}

	// Handle persons ...
	if strings.ToLower(metadata.Lecturer) == common.P_RAV {
		log.Info("Associating unit to rav")
		cup := &models.ContentUnitsPerson{
			ContentUnitID: cu.ID,
			PersonID:      common.PERSON_REGISTRY.ByPattern[common.P_RAV].ID,
			RoleID:        common.CONTENT_ROLE_TYPE_REGISTRY.ByName[common.CR_LECTURER].ID,
		}

		// upsert make sure we either have such relation or insert a new one
		err = cup.Upsert(exec, false, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Associate persons")
		}
	} else if isUpdate && strings.ToLower(metadata.Lecturer) == "norav" {
		// in update mode, if norav so we remove relation to rav (if any)
		cup := &models.ContentUnitsPerson{
			ContentUnitID: cu.ID,
			PersonID:      common.PERSON_REGISTRY.ByPattern[common.P_RAV].ID,
		}
		err = cup.Delete(exec)
		if err != nil {
			return nil, errors.Wrap(err, "Delete Rav association")
		}
	} else {
		log.Infof("Unknown lecturer %s, skipping person association.", metadata.Lecturer)
	}

	// Get or create collection
	if metadata.CollectionUID.Valid {
		log.Infof("Specific collection %s", metadata.CollectionUID.String)

		// find collection
		c, err := models.Collections(exec, qm.Where("uid = ?", metadata.CollectionUID.String)).One()
		if err != nil {
			if err == sql.ErrNoRows {
				log.Warnf("No such collection %s", metadata.CollectionUID.String)
			} else {
				return nil, errors.Wrap(err, "Lookup collection in DB")
			}
		}

		// Associate unit to collection
		if c != nil &&
			(!metadata.ArtifactType.Valid || metadata.ArtifactType.String == "main") {
			err := associateUnitToCollection(exec, cu, c, metadata)
			if err != nil {
				return nil, errors.Wrap(err, "associate content_unit to collection")
			}
			evnts = append(evnts, events.CollectionContentUnitsChangeEvent(c))
		}
	}

	// Update mode ends here
	if isUpdate {
		return evnts, nil
	}

	if ct == common.CT_LESSON_PART ||
		ct == common.CT_FULL_LESSON ||
		ct == common.CT_KTAIM_NIVCHARIM {
		log.Info("Lesson reconciliation")

		// Reconcile or create new
		// Reconciliation is done by looking up the operation chain of original to capture_stop.
		// There we have a property of saying the capture_id of the full lesson capture.
		if captureID, ok := captureStopProps["collection_uid"]; ok {
			log.Infof("Reconcile by capture_id %s", captureID)
			var cct string
			if metadata.WeekDate == nil {
				cct = common.CT_DAILY_LESSON
			} else {
				cct = common.CT_SPECIAL_LESSON
			}

			// Keep this property on the collection for other parts to find it
			props["capture_id"] = captureID
			if metadata.Number.Valid {
				props["number"] = metadata.Number.Int
			}
			delete(props, "duration")
			delete(props, "part")

			// get or create collection
			c, err := FindCollectionByCaptureID(exec, captureID)
			if err != nil {
				if _, ok := err.(CollectionNotFound); !ok {
					return nil, err
				}

				// Create new collection
				log.Info("Creating new collection")
				c, err = CreateCollection(exec, cct, props)
				if err != nil {
					return nil, err
				}
				evnts = append(evnts, events.CollectionCreateEvent(c))
			} else if ct == common.CT_FULL_LESSON {
				// Update collection properties to those of full lesson
				log.Info("Full lesson, overriding collection properties")
				if c.TypeID != common.CONTENT_TYPE_REGISTRY.ByName[cct].ID {
					log.Infof("Full lesson, content_type changed to %s", cct)
					c.TypeID = common.CONTENT_TYPE_REGISTRY.ByName[cct].ID
					err = c.Update(exec, "type_id")
					if err != nil {
						return nil, errors.Wrap(err, "Update collection type in DB")
					}
				}

				err = UpdateCollectionProperties(exec, c, props)
				if err != nil {
					return nil, err
				}
				evnts = append(evnts, events.CollectionUpdateEvent(c))
			}

			// Associate unit to collection
			if c != nil &&
				(!metadata.ArtifactType.Valid ||
					metadata.ArtifactType.String == "main" ||
					metadata.ArtifactType.String == "KTAIM_NIVCHARIM") {
				err := associateUnitToCollection(exec, cu, c, metadata)
				if err != nil {
					return nil, errors.Wrap(err, "associate content_unit to collection")
				}
				evnts = append(evnts, events.CollectionContentUnitsChangeEvent(c))
			}
		} else {
			log.Warn("capture_stop not found or its missing collection_uid. Skipping lesson reconciliation")
		}
	}

	// Associate unit and derived units
	// We take into account that a derived content unit arrives before it's source content unit.
	// Such cases are possible due to the studio operator actions sequence.
	err = original.L.LoadParent(exec, true, original)
	if err != nil {
		return nil, errors.Wrap(err, "Load original's parent")
	}

	if original.R.Parent == nil {
		log.Warn("We don't have original's parent file. Skipping derived units association.")
	} else {
		log.Info("Processing derived units associations")
		if !metadata.ArtifactType.Valid ||
			metadata.ArtifactType.String == "main" {
			// main content unit
			log.Info("We're the main content unit")

			log.Info("Looking up pending derived units")
			derivedCUs, err := mainToDerived(exec, metadata, original)
			if err != nil {
				return nil, err
			}

			log.Infof("%d derived units pending our association", len(derivedCUs))
			for k, v := range derivedCUs {
				log.Infof("DerivedID: %d, Name: %s", k, v)
				cud := &models.ContentUnitDerivation{
					DerivedID: k,
					Name:      v,
				}
				err = cu.AddSourceContentUnitDerivations(exec, true, cud)
				if err != nil {
					return nil, errors.Wrap(err, "Save derived unit association in DB")
				}

				_, err = queries.Raw(exec,
					`UPDATE content_units SET properties = properties - 'artifact_type' WHERE id = $1`,
					k).Exec()
				if err != nil {
					return nil, errors.Wrap(err, "Delete derived unit artifact_type property from DB")
				}
			}

			if len(derivedCUs) > 0 {
				evnts = append(evnts, events.ContentUnitDerivativesChangeEvent(cu))
			}

		} else {
			// derived content unit
			log.Info("We're the derived content unit")

			mainCUID, err := derivedToMain(exec, metadata, cu, original)
			if err != nil {
				return nil, err
			}

			if mainCUID == 0 {
				// save artifact type for later use (when main unit appears)
				log.Info("Main content unit not found, saving artifact_type property")
				err = UpdateContentUnitProperties(exec, cu, map[string]interface{}{
					"artifact_type": metadata.ArtifactType.String,
				})
				if err != nil {
					return nil, err
				}
			} else {
				// main content unit already exists
				log.Infof("Main content unit exists %d", mainCUID)
				cud := &models.ContentUnitDerivation{
					SourceID: mainCUID,
					Name:     metadata.ArtifactType.String,
				}
				err = cu.AddDerivedContentUnitDerivations(exec, true, cud)
				if err != nil {
					return nil, errors.Wrap(err, "Save source unit in DB")
				}
				evnts = append(evnts, events.ContentUnitDerivativesChangeEvent(cu))
			}
		}
	}

	// set default permissions ?!

	return evnts, nil
}

func associateUnitToCollection(exec boil.Executor, cu *models.ContentUnit, c *models.Collection, metadata CITMetadata) error {
	log.Infof("Associating unit and collection [c-cu]=[%d-%d]", c.ID, cu.ID)

	ccu := &models.CollectionsContentUnit{
		CollectionID:  c.ID,
		ContentUnitID: cu.ID,
	}

	switch common.CONTENT_TYPE_REGISTRY.ByID[cu.TypeID].Name {
	case common.CT_FULL_LESSON:
		if c.TypeID == common.CONTENT_TYPE_REGISTRY.ByName[common.CT_DAILY_LESSON].ID ||
			c.TypeID == common.CONTENT_TYPE_REGISTRY.ByName[common.CT_SPECIAL_LESSON].ID {
			ccu.Name = "full"
		} else if metadata.Number.Valid {
			ccu.Name = strconv.Itoa(metadata.Number.Int)
		}
		break
	case common.CT_LESSON_PART:
		if metadata.Part.Valid {
			ccu.Name = strconv.Itoa(metadata.Part.Int)
		}
		break
	case common.CT_VIDEO_PROGRAM_CHAPTER:
		if metadata.Episode.Valid {
			ccu.Name = metadata.Episode.String
		}
		break
	default:
		if metadata.Number.Valid {
			ccu.Name = strconv.Itoa(metadata.Number.Int)
		}

		// first 3 event part types are lesson, YH and meal, we skip them.
		if metadata.PartType.Valid && metadata.PartType.Int > 2 {
			idx := metadata.PartType.Int - 3
			if idx < len(common.MISC_EVENT_PART_TYPES) {
				ccu.Name = common.MISC_EVENT_PART_TYPES[idx] + ccu.Name
			} else {
				log.Warnf("Unknown event part type: %d", metadata.PartType.Int)
			}
		}
		break
	}
	if metadata.ArtifactType.Valid &&
		metadata.ArtifactType.String != "main" {
		ccu.Name = fmt.Sprintf("%s_%s", metadata.ArtifactType.String, ccu.Name)
	}

	// Make this new unit the last one in this collection
	var err error
	ccu.Position, err = GetNextPositionInCollection(exec, c.ID)
	if err != nil {
		return errors.Wrap(err, "Get last position in collection")
	}

	log.Infof("Association name: %s", ccu.Name)
	err = ccu.Upsert(exec, true,
		[]string{"collection_id", "content_unit_id"},
		[]string{"name", "position"})
	if err != nil {
		return errors.Wrap(err, "Save collection and content unit association in DB")
	}

	return nil
}

func mainToDerived(exec boil.Executor, metadata CITMetadata, original *models.File) (map[int64]string, error) {
	part := -888 // something we never use for part
	if metadata.Part.Valid {
		part = metadata.Part.Int
	}

	// We lookup original's siblings for derived content units that arrived before us.
	// We then associate them with us and remove their "unprocessed" mark.
	// Meaning, the presence of "artifact_type" property
	rows, err := queries.Raw(exec,
		`SELECT
  cu.id,
  cu.properties ->> 'artifact_type'
FROM content_units cu
  INNER JOIN files f ON f.content_unit_id = cu.id AND f.parent_id = $1
WHERE cu.properties ? 'artifact_type' AND (cu.properties ->> 'part') :: INT = $2`,
		original.ParentID.Int64, part).
		Query()
	if err != nil {
		return nil, errors.Wrap(err, "Load derived content units")
	}

	// put results in a map first since we can't process them while iterating.
	// see this bug:  https://github.com/lib/pq/issues/81
	derivedCUs := make(map[int64]string)
	for rows.Next() {
		var cuid int64
		var artifactType string
		err = rows.Scan(&cuid, &artifactType)
		if err != nil {
			return nil, errors.Wrap(err, "Scan row")
		}
		derivedCUs[cuid] = artifactType
	}
	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "Iter rows")
	}
	err = rows.Close()
	if err != nil {
		return nil, errors.Wrap(err, "Close rows")
	}

	return derivedCUs, nil
}

func derivedToMain(exec boil.Executor, metadata CITMetadata, cu *models.ContentUnit, original *models.File) (int64, error) {
	part := -888 // something we never use for part
	if metadata.Part.Valid {
		part = metadata.Part.Int
	}

	mainCT, ok := common.CONTENT_TYPE_REGISTRY.ByName[metadata.ContentType]
	if !ok {
		return 0, errors.Errorf("Unknown content type %s", metadata.ContentType)
	}

	var cuID int64
	err := queries.Raw(exec, `
SELECT cu.id
FROM content_units cu
  INNER JOIN files f ON f.content_unit_id = cu.id AND f.parent_id = $1 AND cu.id != $2 AND cu.type_id = $3
WHERE (cu.properties ->> 'part') :: INT = $4`,
		original.ParentID.Int64, cu.ID, mainCT.ID, part).QueryRow().Scan(&cuID)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		} else {
			return 0, errors.Wrap(err, "Query main CU ID")
		}
	}

	return cuID, nil
}

/* send-fix

Sometimes, after a unit was created in a send operation,
we need to fix it.

Either the metadata that was given is wrong or we might need a different trim.
For such cases a new button in the trim admin "fix" is made.

The workflow simply shows the same CIT screen and unit selection.
We should:

1. re-process the metadata in "update" mode
2. figure out files for removal
3. mark those as removed
4. update the unit's published status

*/
func ProcessCITMetadataUpdate(exec boil.Executor, metadata CITMetadata, original, proxy *models.File) ([]events.Event, error) {
	unit, err := models.ContentUnits(exec, qm.Where("uid = ?", metadata.UnitToFixUID.String)).One()
	if err != nil {
		return nil, errors.Wrapf(err, "lookup unit UID %s", metadata.UnitToFixUID.String)
	}

	evnts, err := doProcess(exec, metadata, original, proxy, unit)
	if err != nil {
		return nil, errors.Wrap(err, "doProcess")
	}

	// We remove only files generated in convert (carbon)
	// and previous trimmed not in our path.
	// Other, manually inserted, files are not touched and are left to admin
	// to figure out what to do with them.

	// Figure out merged set of file IDs
	// which are either ancestor of original or proxy
	// These should be excluded from removal.
	mutualAncestors := hashset.New()

	oPath, err := FindFileAncestors(exec, original.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "lookup original ancestors %d", original.ID)
	}
	for i := range oPath {
		mutualAncestors.Add(oPath[i].ID)
	}

	if proxy != nil {
		pPath, err := FindFileAncestors(exec, proxy.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "lookup proxy ancestors %d", original.ID)
		}
		for i := range pPath {
			mutualAncestors.Add(pPath[i].ID)
		}
	}

	ancestorsIDs := mutualAncestors.Values()
	// These are the fix. Not the problem. Don't remove them
	ancestorsIDs = append(ancestorsIDs, original.ID)
	if proxy != nil {
		ancestorsIDs = append(ancestorsIDs, original.ID, proxy.ID)
	}

	log.Infof("ancestorsIDs: %v", ancestorsIDs)

	// fetch file IDs to remove
	var fIDs pq.Int64Array
	q := `SELECT array_agg(distinct f.id)
FROM files f
  INNER JOIN files_operations fo ON f.id = fo.file_id
  INNER JOIN operations o ON fo.operation_id = o.id AND o.type_id = ANY($1)
WHERE f.content_unit_id = $2 AND NOT f.id = ANY($3) 
`
	err = queries.Raw(exec, q, pq.Array([]int64{
		common.OPERATION_TYPE_REGISTRY.ByName[common.OP_TRIM].ID,
		common.OPERATION_TYPE_REGISTRY.ByName[common.OP_CONVERT].ID,
	}), unit.ID, pq.Array(ancestorsIDs)).QueryRow().Scan(&fIDs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch file IDs to remove")
	}

	log.Infof("%d files to remove: %v", len(fIDs), fIDs)
	wasPublished := false
	if len(fIDs) > 0 {
		// actual removal
		err = models.Files(exec,
			qm.WhereIn("id in ?", utils.ConvertArgsInt64(fIDs)...)).
			UpdateAll(models.M{
				"removed_at": null.TimeFrom(time.Now().UTC()),
			})
		if err != nil {
			return nil, errors.Wrap(err, "Update files to remove")
		}

		// file removed events
		removedFiles, err := models.Files(exec,
			qm.Select("id", "uid", "published"),
			qm.WhereIn("id in ?", utils.ConvertArgsInt64(fIDs)...)).
			All()
		if err != nil {
			return nil, errors.Wrap(err, "Refresh files to remove")
		}

		for i := range removedFiles {
			evnts = append(evnts, events.FileRemoveEvent(removedFiles[i]))
			wasPublished = wasPublished || removedFiles[i].Published
		}

	}

	// unit published status change
	impact, err := FileLeftUnitImpact(exec, wasPublished, unit.ID)
	if err != nil {
		return nil, errors.Wrap(err, "File left impact")
	}
	evnts = append(evnts, impact.Events()...)

	return evnts, nil
}
