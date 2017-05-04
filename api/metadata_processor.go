package api

import (
	"database/sql"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/vattle/sqlboiler/boil"
	"github.com/vattle/sqlboiler/queries/qm"
	"gopkg.in/nullbio/null.v6"

	"encoding/json"
	"github.com/Bnei-Baruch/mdb/models"
	"github.com/Bnei-Baruch/mdb/utils"
	"github.com/vattle/sqlboiler/queries"
	"strings"
)

// Do all stuff for processing metadata coming from Content Identification Tool.
// 	1. Update properties for original and proxy (film_date, capture_date)
//	2. Update language of original
// 	3. Create content_unit (content_type, dates)
//	4. Add files to new unit
// 	5. Add ancestor files to unit
// 	6. Associate unit with sources, tags, and persons
// 	7. Get or create collection
// 	8. Update collection (content_type, dates, number) if full lesson or new lesson
// 	9. Associate collection and unit
// 	10. Associate unit and derived units
// 	11. Set default permissions ?!
func ProcessCITMetadata(exec boil.Executor, metadata CITMetadata, original, proxy *models.File) error {
	log.Info("Processing CITMetadata")

	// Update properties for original and proxy (film_date, capture_date)
	filmDate := metadata.CaptureDate
	if metadata.WeekDate != nil {
		filmDate = *metadata.WeekDate
	}
	props := map[string]interface{}{
		"capture_date": metadata.CaptureDate,
		"film_date":    filmDate,
	}
	log.Infof("Updaing files properties: %v", props)
	err := UpdateFileProperties(exec, original, props)
	if err != nil {
		return err
	}
	err = UpdateFileProperties(exec, proxy, props)
	if err != nil {
		return err
	}

	// Update language of original.
	// TODO: What about proxy !?
	if metadata.HasTranslation {
		original.Language = null.StringFrom(LANG_MULTI)
	} else {
		l := StdLang(metadata.Language)
		if l == LANG_UNKNOWN {
			log.Warnf("Unknown language in metadata %s", metadata.Language)
		}
		original.Language = null.StringFrom(l)
	}
	log.Infof("Updaing original.Language to %s", original.Language.String)
	err = original.Update(exec, "language")
	if err != nil {
		return errors.Wrap(err, "Save original to DB")
	}

	// Create content_unit (content_type, dates)
	log.Infof("Creating content unit of type %s", metadata.ContentType)
	cu, err := CreateContentUnit(exec, metadata.ContentType, props)
	if err != nil {
		return errors.Wrap(err, "Create content unit")
	}

	// Add files to new unit
	log.Info("Adding files to unit")
	err = cu.AddFiles(exec, false, original, proxy)
	if err != nil {
		return errors.Wrap(err, "Add files to unit")
	}

	// Add ancestor files to unit (not for derived units)
	if !metadata.ArtifactType.Valid ||
		metadata.ArtifactType.String == "main" {
		log.Info("Main unit, adding ancestors...")
		ancestors, err := FindFileAncestors(exec, original.ID)
		if err != nil {
			return errors.Wrap(err, "Find original's ancestors")
		}
		err = cu.AddFiles(exec, false, ancestors...)
		if err != nil {
			return errors.Wrap(err, "Add ancestors to unit")
		}
		log.Infof("Added %d ancestors", len(ancestors))
		for _, x := range ancestors {
			log.Infof("%s [%d]", x.Name, x.ID)
		}
	}

	// Associate unit with sources, tags, and persons
	if len(metadata.Sources) > 0 {
		log.Infof("Associating %d sources", len(metadata.Sources))
		sources, err := models.Sources(exec,
			qm.WhereIn("uid in ?", utils.ConvertArgsString(metadata.Sources)...)).
			All()
		if err != nil {
			return errors.Wrap(err, "Lookup sources in DB")
		}
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
		err = cu.AddSources(exec, false, sources...)
		if err != nil {
			return errors.Wrap(err, "Associate sources")
		}
	}

	if len(metadata.Tags) > 0 {
		log.Infof("Associating %d tags", len(metadata.Tags))
		tags, err := models.Tags(exec,
			qm.WhereIn("uid in ?", utils.ConvertArgsString(metadata.Tags)...)).
			All()
		if err != nil {
			return errors.Wrap(err, "Lookup tags  in DB")
		}
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
		err = cu.AddTags(exec, false, tags...)
		if err != nil {
			return errors.Wrap(err, "Associate tags")
		}
	}

	// Handle persons ...
	if strings.ToLower(metadata.Lecturer) == P_RAV {
		log.Info("Associating unit to rav")
		cup := &models.ContentUnitsPerson{
			PersonID: PERSONS_REGISTRY.ByPattern[P_RAV].ID,
			RoleID:   CONTENT_ROLE_TYPE_REGISTRY.ByName[CR_LECTURER].ID,
		}
		err = cu.AddContentUnitsPersons(exec, true, cup)
		if err != nil {
			return errors.Wrap(err, "Associate persons")
		}
	} else {
		log.Infof("Unknown lecturer %s, skipping person association.", metadata.Lecturer)
	}

	// Get or create collection
	var c *models.Collection
	if metadata.CollectionUID.Valid {
		log.Infof("Specific collection %s", metadata.CollectionUID .String)
		c, err = models.Collections(exec, qm.Where("uid = ?", metadata.CollectionUID.String)).One()
		if err != nil {
			if err == sql.ErrNoRows {
				log.Warnf("No such collection %s", metadata.CollectionUID.String)
			} else {
				return errors.Wrap(err, "Lookup collection in DB")
			}
		}
	} else if metadata.ContentType == CT_LESSON_PART ||
		metadata.ContentType == CT_FULL_LESSON {
		log.Info("Daily lesson reconciliation")

		// Reconcile or create new
		// Reconciliation is done by looking up the operation chain of original to capture_stop.
		// There we have a property of saying the capture_id of the full lesson capture.
		captureStop, err := FindUpChainOperation(exec, original.ID,
			OPERATION_TYPE_REGISTRY.ByName[OP_CAPTURE_STOP].ID)
		if err != nil {
			if ex, ok := err.(UpChainOperationNotFound); ok {
				log.Warnf(ex.Error())
			} else {
				return err
			}
		} else if captureStop.Properties.Valid {
			var oProps map[string]interface{}
			err = json.Unmarshal(captureStop.Properties.JSON, &oProps)
			if err != nil {
				return errors.Wrap(err, "json Unmarshal")
			}
			captureID, ok := oProps["collection_uid"]
			if ok {
				log.Infof("Reconcile by capture_id %s", captureID)
				var ct string
				if metadata.WeekDate == nil {
					ct = CT_DAILY_LESSON
				} else {
					ct = CT_SATURDAY_LESSON
				}

				// Keep this property on the collection for other parts to find it
				props["capture_id"] = captureID
				if metadata.Number.Valid {
					props["number"] = metadata.Number.Int
				}

				c, err = FindCollectionByCaptureID(exec, captureID)
				if err != nil {
					if _, ok := err.(CollectionNotFound); !ok {
						return err
					}

					// Create new collection
					log.Info("Creating new collection")
					c, err = CreateCollection(exec, ct, props)
					if err != nil {
						return err
					}
				} else if metadata.ContentType == CT_FULL_LESSON {
					// Update collection properties to those of full lesson
					log.Info("Full lesson, overriding collection properties")
					if c.TypeID != CONTENT_TYPE_REGISTRY.ByName[ct].ID {
						log.Infof("Full lesson, content_type changed to %s", ct)
						c.TypeID = CONTENT_TYPE_REGISTRY.ByName[ct].ID
						err = c.Update(exec, "type_id")
						if err != nil {
							return errors.Wrap(err, "Update collection type in DB")
						}
					}

					err = UpdateCollectionProperties(exec, c, props)
					if err != nil {
						return err
					}
				}
			} else {
				log.Warnf("No collection_uid in capture_stop [%d] properties", captureStop.ID)
			}
		} else {
			log.Warnf("Invalid properties in capture_stop [%d]", captureStop.ID)
		}
	}

	// Associate collection and unit
	if c != nil &&
		(!metadata.ArtifactType.Valid || metadata.ArtifactType.String == "main") {
		log.Info("Associating unit and collection")
		ccu := &models.CollectionsContentUnit{
			CollectionID:  c.ID,
			ContentUnitID: cu.ID,
		}
		switch metadata.ContentType {
		case CT_FULL_LESSON:
			if c.TypeID == CONTENT_TYPE_REGISTRY.ByName[CT_DAILY_LESSON].ID ||
				c.TypeID == CONTENT_TYPE_REGISTRY.ByName[CT_SATURDAY_LESSON].ID {
				ccu.Name = "full"
			} else if metadata.Number.Valid {
				ccu.Name = strconv.Itoa(metadata.Number.Int)
			}
			break
		case CT_LESSON_PART:
			if metadata.Part.Valid {
				ccu.Name = strconv.Itoa(metadata.Part.Int)
			}
			break
		case CT_VIDEO_PROGRAM_CHAPTER:
			if metadata.Episode.Valid {
				ccu.Name = metadata.Episode.String
			}
			break
		default:
			if metadata.Number.Valid {
				ccu.Name = strconv.Itoa(metadata.Number.Int)
			}
			if metadata.PartType.Valid && metadata.PartType.Int > 2 {
				idx := metadata.PartType.Int - 3
				if idx < len(MISC_EVENT_PART_TYPES) {
					ccu.Name = MISC_EVENT_PART_TYPES[idx] + ccu.Name
				} else {
					log.Warn("Unknown event part type: %d", metadata.PartType.Int)
				}
			}
			break
		}

		log.Infof("Association name: %s", ccu.Name)
		err = c.AddCollectionsContentUnits(exec, true, ccu)
		if err != nil {
			return errors.Wrap(err, "Save collection and content unit association in DB")
		}
	}

	// Associate unit and derived units
	// We take into account that a derived content unit arrives before it's source content unit.
	// Such cases are possible due to the studio operator actions sequence.
	err = original.L.LoadParent(exec, true, original)
	if err != nil {
		return errors.Wrap(err, "Load original's parent")
	}

	if original.R.Parent == nil {
		log.Warn("We don't have original's parent file. Skipping derived units association.")
	} else {
		log.Info("Processing derived units associations")
		mainCUID := original.R.Parent.ContentUnitID
		if !metadata.ArtifactType.Valid ||
			metadata.ArtifactType.String == "main" {
			// main content unit
			log.Info("We're the main content unit")

			// We lookup original's siblings for derived content units that arrived before us.
			// We then associate them with us and remove their "unprocessed" mark.
			// Meaning, the presence of "artifact_type" property
			rows, err := queries.Raw(exec,
				`SELECT
				  cu.id,
				  cu.properties ->> 'artifact_type'
				FROM files f
				  INNER JOIN content_units cu ON f.content_unit_id = cu.id
				    AND cu.id != $1
				    AND cu.properties ? 'artifact_type'
				WHERE f.parent_id = $2`,
				original.ContentUnitID.Int64, original.ParentID.Int64).
				Query()
			if err != nil {
				return errors.Wrap(err, "Load derived content units")
			}

			// put results in map due to this bug:
			// https://github.com/lib/pq/issues/81
			derivedCUs := make(map[int64]string)
			for rows.Next() {
				var cuid int64
				var artifactType string
				err = rows.Scan(&cuid, &artifactType)
				if err != nil {
					return errors.Wrap(err, "Scan row")
				}
				derivedCUs[cuid] = artifactType
			}
			err = rows.Err()
			if err != nil {
				return errors.Wrap(err, "Iter rows")
			}
			err = rows.Close()
			if err != nil {
				return errors.Wrap(err, "Close rows")
			}

			log.Infof("%d dervied units pending our association", len(derivedCUs))
			for k, v := range derivedCUs {
				log.Infof("DerivedID: %d, Name: %s", k, v)
				cud := &models.ContentUnitDerivation{
					DerivedID: k,
					Name:      v,
				}
				err = cu.AddSourceContentUnitDerivations(exec, true, cud)
				if err != nil {
					return errors.Wrap(err, "Save derived unit association in DB")
				}

				_, err = queries.Raw(exec,
					`UPDATE content_units SET properties = properties - 'artifact_type' WHERE id = $1`,
					k).Exec()
				if err != nil {
					return errors.Wrap(err, "Delete derived unit artifact_type property from DB")
				}
			}

		} else {
			// derived content unit
			log.Info("We're the derived content unit")

			if mainCUID.Valid {
				// main content unit already exists
				log.Infof("Main content unit exists %d", mainCUID.Int64)
				cud := &models.ContentUnitDerivation{
					SourceID: mainCUID.Int64,
					Name:     metadata.ArtifactType.String,
				}
				err = cu.AddDerivedContentUnitDerivations(exec, true, cud)
				if err != nil {
					return errors.Wrap(err, "Save source unit in DB")
				}
			} else {
				// save artifact type for later use (when main unit appears)
				log.Info("Main content unit not found, saving artifact_type property")
				err = UpdateContentUnitProperties(exec, cu, map[string]interface{}{
					"artifact_type": metadata.ArtifactType.String,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// set default permissions ?!

	return nil
}