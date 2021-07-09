package likutim

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"gopkg.in/volatiletech/null.v6"

	"github.com/Bnei-Baruch/mdb/common"
	"github.com/Bnei-Baruch/mdb/models"
	"github.com/Bnei-Baruch/mdb/utils"
)

type CreateUnits struct {
	duplicates []*Double
	mdb        *sql.DB
}

func (c *CreateUnits) Run() {
	err := c.duplicatesFromJSON()
	//if no json file with data of duplicated doc files run compare
	if err != nil {
		log.Errorf("Error on read. Error: %s", err)
		compare := new(Compare)
		compare.Run()
		c.duplicates = compare.result
	}

	c.openDB()
	defer c.mdb.Close()

	utils.Must(os.MkdirAll(viper.GetString("likutim.os-dir"), os.ModePerm))
	c.updateUnits()
}

func (c *CreateUnits) updateUnits() {
	for _, d := range c.duplicates {
		log.Debugf("\n\nStart create new unit type LIKUTIM, file: %s all files: %v", d.Save, d.Doubles)

		tx, err := c.mdb.Begin()
		if err != nil {
			log.Error(err)
			continue
		}
		derived := make([]int64, len(d.Doubles))
		cu, err := c.createUnit(tx, d.Save, derived)
		if err != nil {
			log.Error(err)
			if err = tx.Rollback(); err != nil {
				log.Errorf("problem rollback transaction. Error: %s", err)
			}
			continue
		}

		// Derivation unit type LIKITIM to origins of doubles
		for i, uid := range d.Doubles {
			if uid == d.Save {
				continue
			}

			if err = moveData(tx, uid, cu, derived, i); err != nil {
				log.Error(err)
				if err = tx.Rollback(); err != nil {
					log.Errorf("problem rollback transaction. Error: %s", err)
				}
				break
			}
		}

		if err = tx.Commit(); err != nil {
			log.Errorf("problem commit transaction. Error: %s", err)
		}
		log.Debugf("End create new unit type LIKUTIM id: %d", cu.ID)
	}
}

func (c *CreateUnits) createUnit(tx *sql.Tx, uid string, derived []int64) (*models.ContentUnit, error) {
	//fetch files and get origin unit
	f, err := models.Files(tx,
		qm.Where("uid = ?", uid),
		qm.Load("ContentUnit", "ContentUnit.Files", "ContentUnit.DerivedContentUnitDerivations", "ContentUnit.DerivedContentUnitDerivations.Source"),
	).One()
	if err != nil {
		return nil, errors.Wrapf(err, "Can't find base file by uid: %s.", uid)
	}
	if f.R.ContentUnit == nil {
		return nil, errors.Wrapf(err, "Can't find base unit by file: %v.", f)
	}

	if f.R.ContentUnit.TypeID == common.CONTENT_TYPE_REGISTRY.ByName[common.CT_LIKUTIM].ID {
		return nil, errors.New(fmt.Sprintf("this file alredy have Unit type LIKUTIM : %v.", f))
	}

	if len(f.R.ContentUnit.R.DerivedContentUnitDerivations) == 0 {
		return nil, errors.Wrapf(err, "Can't find origin unit by unit: %v.", f.R.ContentUnit)
	}

	cuo, err := models.ContentUnits(tx,
		qm.Where("id = ?", f.R.ContentUnit.R.DerivedContentUnitDerivations[0].SourceID),
		qm.Load("ContentUnitI18ns", "Tags"),
	).One()
	if err != nil {
		return nil, errors.Wrapf(err, "Can't find origin unit by unit: %v.", f.R.ContentUnit)
	}
	var props map[string]interface{}
	var filmDate string
	if f.R.ContentUnit.Properties.Valid {
		if err = json.Unmarshal(f.R.ContentUnit.Properties.JSON, &props); err != nil {
			return nil, errors.Wrapf(err, "Can't unmarshal properties of unit: %d.", f.R.ContentUnit.ID)
		}
		filmDate = fmt.Sprintf("%v", props["film_date"])
	} else {
		filmDate = time.Now().Format("2006-01-02")
	}
	//create unit type LIKUTIM
	cu, err := c.createCU(tx, cuo, filmDate, f)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't create unit %v to unit: %v.", cu, cuo)
	}
	if err := c.moveFiles(tx, f.R.ContentUnit, &cu); err != nil {
		return nil, err
	}
	//derivation to origin
	derived[0] = cuo.ID
	cud := &models.ContentUnitDerivation{
		SourceID:  cuo.ID,
		DerivedID: cu.ID,
	}
	err = cuo.AddSourceContentUnitDerivations(tx, true, cud)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't derive unit %v to unit: %v.", cu, cuo)
	}
	return &cu, nil
}

func moveData(tx *sql.Tx, uid string, cu *models.ContentUnit, derived []int64, pos int) error {

	f, err := models.Files(tx, qm.Where("uid = ?", uid),
		qm.Load("ContentUnit", "ContentUnit.DerivedContentUnitDerivations", "ContentUnit.DerivedContentUnitDerivations.Source")).One()
	if err != nil {
		return errors.Wrapf(err, "Can't find file by uid: %s.", uid)
	}

	if f.R.ContentUnit == nil {
		return errors.Wrapf(err, "Can't find unit by file: %v.", f)
	}

	if len(f.R.ContentUnit.R.DerivedContentUnitDerivations) == 0 {
		return errors.Wrapf(err, "Can't find origin unit by unit: %v.", f.R.ContentUnit)
	}
	//check if new unit was already derived (origins can be same)
	for _, duid := range derived {
		if duid == f.R.ContentUnit.R.DerivedContentUnitDerivations[0].SourceID {
			return nil
		}
	}
	derived[pos] = f.R.ContentUnit.R.DerivedContentUnitDerivations[0].SourceID
	d := &models.ContentUnitDerivation{
		SourceID:  f.R.ContentUnit.R.DerivedContentUnitDerivations[0].SourceID,
		DerivedID: cu.ID,
	}
	err = cu.AddDerivedContentUnitDerivations(tx, true, d)
	if err != nil {
		return errors.Wrapf(err, "Can't derive unit %v to unit id: %d.", cu, f.R.ContentUnit.R.DerivedContentUnitDerivations[0].SourceID)
	}
	return nil
}

func (c *CreateUnits) moveFiles(tx *sql.Tx, cukm, newu *models.ContentUnit) error {
	for _, f := range cukm.R.Files {
		if !strings.Contains(f.Name, ".doc") {
			continue
		}
		f.ContentUnitID = null.Int64{Int64: newu.ID, Valid: true}
		if err := f.Update(tx, "content_unit_id"); err != nil {
			return errors.Wrapf(err, "Can't insert files from kitvei makor unit: %s  to new CU: %s", cukm.UID, newu.UID)
		}
	}
	return nil
}

func (c *CreateUnits) createCU(tx *sql.Tx, cuo *models.ContentUnit, filmDate string, f *models.File) (models.ContentUnit, error) {
	props, _ := json.Marshal(map[string]string{"film_date": filmDate, "pattern": patternByFileName(f.Name), "original_language": common.LANG_HEBREW})
	cu := models.ContentUnit{
		UID:        utils.GenerateUID(8),
		TypeID:     common.CONTENT_TYPE_REGISTRY.ByName[common.CT_LIKUTIM].ID,
		Secure:     0,
		Published:  true,
		Properties: null.JSONFrom(props),
	}
	err := cu.Insert(tx)
	if err != nil {
		log.Errorf("Can't add tags for CU id %d. Error: %s", cu.ID, err)
		return cu, err
	}
	log.Debugf("Unit was inserted CU %v", cu)

	//take data from origin for new unit
	err = cu.AddTags(tx, false, cuo.R.Tags...)
	if err != nil {
		log.Errorf("Can't add tags for CU id %d. Error: %s", cu.ID, err)
		return cu, err
	}

	var i18n models.ContentUnitI18nSlice
	for _, i := range cuo.R.ContentUnitI18ns {
		n := &models.ContentUnitI18n{
			ContentUnitID: cu.ID,
			Language:      i.Language,
			Name:          i.Name,
		}
		i18n = append(i18n, n)
	}
	err = cu.AddContentUnitI18ns(tx, true, i18n...)
	if err != nil {
		log.Errorf("Can't add i18n for CU id %d. Error: %s", cu.ID, err)
		return cu, err
	}
	return cu, nil
}
func patternByFileName(name string) string {
	spl := strings.Split(strings.ToLower(name), "kitei-makor")
	spl = strings.Split(spl[1], ".")
	spl = strings.Split(spl[0], "_")
	if spl[0] != "" || len(spl) < 2 {
		return spl[0]
	}
	return spl[1]
}

func (c *CreateUnits) duplicatesFromJSON() error {
	p := path.Join(viper.GetString("likutim.os-dir"), "kitvei-makor-duplicates.json")
	j, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}
	var r []*Double
	err = json.Unmarshal(j, &r)
	if err != nil {
		return err
	}
	c.duplicates = r
	return nil
}

func (c *CreateUnits) openDB() {
	mdb, err := sql.Open("postgres", viper.GetString("mdb.url"))
	utils.Must(err)
	utils.Must(mdb.Ping())
	boil.SetDB(mdb)
	boil.DebugMode = true
	utils.Must(common.InitTypeRegistries(mdb))
	c.mdb = mdb
}
