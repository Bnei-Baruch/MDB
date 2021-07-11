package batch

import (
	"database/sql"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/mdb/models"
	"github.com/Bnei-Baruch/mdb/utils"
)

type PostsHttpToHttps struct {
	limit  int
	domain string
	mdb    boil.Executor
	oldStr string
	newStr string
}

func NewPostsHttpToHttps() PostsHttpToHttps {
	domain := "youtube"
	return PostsHttpToHttps{
		limit:  100,
		domain: domain,
		oldStr: "https://www." + domain,
		newStr: "http://www." + domain,
	}

}

func (c *PostsHttpToHttps) Do() {
	log.Infof("Start replace %s to %s for domain %s", c.oldStr, c.newStr, c.domain)
	mdb, err := sql.Open("postgres", viper.GetString("mdb.url"))
	utils.Must(err)
	utils.Must(mdb.Ping())
	defer mdb.Close()
	c.mdb = mdb

	total, err := models.BlogPosts(mdb, qm.Where(fmt.Sprintf("content ~ '.*%s*.'", c.oldStr))).Count()
	log.Debugf("Total blogs for replace %d", total)
	iterations := int(total) / c.limit

	log.Infof("\n\nStart transaction")
	tx, err := mdb.Begin()
	utils.Must(err)
	for i := 0; i <= iterations; i++ {
		err = c.replace(tx, i)
		if err != nil {
			log.Errorf("Exception %v on iteration %d", err, i)
			utils.Must(tx.Rollback())
			break
		}
		utils.Must(err)
	}

	utils.Must(tx.Commit())
	log.Infof("Transaction committed")
	log.Infof("End replace http to https for domain %s", c.domain)
}
func (c *PostsHttpToHttps) replace(tx boil.Transactor, iteration int) error {
	log.Infof("Start replace on iteration %d", iteration)
	posts, err := models.BlogPosts(tx,
		qm.Where(fmt.Sprintf("content ~ '.*%s*.'", c.oldStr)),
		qm.Limit(c.limit)).All()
	if err != nil {
		return err
	}
	for _, p := range posts {
		log.Infof("Replace on post id %d content %s", p.ID, p.Content)
		p.Content = strings.ReplaceAll(p.Content, c.oldStr, c.newStr)
		log.Infof("New content %s", p.Content)
		err = p.Update(tx, "content")
		if err != nil {
			return err
		}

		log.Infof("Successfully replace for post id %d", p.ID)
	}
	log.Infof("End replace on iteration %d", iteration)
	return nil
}
