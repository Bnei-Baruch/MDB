package api

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/volatiletech/null.v6"

	"github.com/Bnei-Baruch/mdb/common"
	"github.com/Bnei-Baruch/mdb/models"
	"github.com/Bnei-Baruch/mdb/utils"
)

type AutonameSuite struct {
	suite.Suite
	utils.TestDBManager
	tx *sql.Tx
}

func (suite *AutonameSuite) SetupSuite() {
	suite.Require().Nil(suite.InitTestDB())
	suite.Require().Nil(common.InitTypeRegistries(suite.DB))
}

func (suite *AutonameSuite) TearDownSuite() {
	suite.Require().Nil(suite.DestroyTestDB())
}

func (suite *AutonameSuite) SetupTest() {
	var err error
	suite.tx, err = suite.DB.Begin()
	suite.Require().Nil(err)
}

func (suite *AutonameSuite) TearDownTest() {
	err := suite.tx.Rollback()
	suite.Require().Nil(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestAutoname(t *testing.T) {
	suite.Run(t, new(AutonameSuite))
}

func (suite *AutonameSuite) TestGenericDescriberCollection() {
	describer := new(GenericDescriber)
	c := new(models.Collection)

	for _, x := range common.ALL_CONTENT_TYPES {
		c.TypeID = common.CONTENT_TYPE_REGISTRY.ByName[x].ID
		names, err := GetI18ns(fmt.Sprintf("content_type.%s", x))
		suite.Require().Nil(err)
		i18ns, err := describer.DescribeCollection(c)
		suite.Require().Nil(err)
		suite.Len(i18ns, len(names), "len(i18ns)")
		for _, i18n := range i18ns {
			suite.Equal(names[i18n.Language], i18n.Name.String, "%s name", i18n.Language)
		}
	}
}

func (suite *AutonameSuite) TestGenericDescriberContentUnit() {
	describer := new(GenericDescriber)
	cu := new(models.ContentUnit)

	for _, x := range common.ALL_CONTENT_TYPES {
		metadata := CITMetadata{
			FinalName: "final_name",
		}
		cu.TypeID = common.CONTENT_TYPE_REGISTRY.ByName[x].ID
		i18ns, err := describer.DescribeContentUnit(suite.tx, cu, metadata)
		suite.Require().Nil(err)
		for _, i18n := range i18ns {
			if x == common.CT_KITEI_MAKOR ||
				x == common.CT_LELO_MIKUD ||
				x == common.CT_FULL_LESSON ||
				x == common.CT_PUBLICATION ||
				x == common.CT_RESEARCH_MATERIAL {
				suite.Equal(metadata.FinalName, i18n.Name.String, "%s technical name", i18n.Language)
				suite.Len(i18ns, 3, "len(i18ns)")
			} else {
				suite.Len(i18ns, 9, "len(i18ns)")
				suite.NotEmpty(i18n.Name.String, "%s name empty", i18n.Language)
				suite.NotEqual(metadata.FinalName, i18n.Name.String, "%s name", i18n.Language)
			}
		}
	}
}

func (suite *AutonameSuite) TestLessonPartDescriber() {
	cu, err := CreateContentUnit(suite.tx, common.CT_LESSON_PART, nil)
	suite.Require().Nil(err)

	metadata := CITMetadata{
		ContentType:    common.CT_LESSON_PART,
		AutoName:       "auto_name",
		FinalName:      "final_name",
		CaptureDate:    Date{time.Now()},
		Language:       common.LANG_HEBREW,
		HasTranslation: true,
		Lecturer:       "rav",
		Number:         null.IntFrom(1),
		Part:           null.IntFrom(0),
	}

	describer := new(LessonPartDescriber)
	i18ns, err := describer.DescribeContentUnit(suite.tx, cu, metadata)
	suite.Require().Nil(err)
	suite.NotEmpty(i18ns, "i18ns.empty")
	for _, i18n := range i18ns {
		switch i18n.Language {
		case common.LANG_HEBREW:
			suite.Equal("הכנה לשיעור", i18n.Name.String, "Hebrew name")
			break
		case common.LANG_ENGLISH:
			suite.Equal("Preparation to the Lesson", i18n.Name.String, "English name")
			break
		case common.LANG_RUSSIAN:
			suite.Equal("Подготовка к уроку", i18n.Name.String, "Russian name")
			break
		}
	}

	// test in event
	metadata.Number = null.IntFrom(2)
	metadata.CollectionUID = null.StringFrom("12345678")
	i18ns, err = describer.DescribeContentUnit(suite.tx, cu, metadata)
	suite.Require().Nil(err)
	suite.NotEmpty(i18ns, "i18ns.empty")
	for _, i18n := range i18ns {
		switch i18n.Language {
		case common.LANG_HEBREW:
			suite.Equal("הכנה לשיעור 2", i18n.Name.String, "Hebrew name")
		case common.LANG_ENGLISH:
			suite.Equal("Preparation to the Lesson 2", i18n.Name.String, "English name")
		}
	}
}

func (suite *AutonameSuite) TestEventPartDescriber() {
	c, err := CreateCollection(suite.tx, common.CT_CONGRESS, nil)
	suite.Require().Nil(err)
	err = c.AddCollectionI18ns(suite.tx, true,
		&models.CollectionI18n{
			Language: common.LANG_HEBREW,
			Name:     null.StringFrom("כנס"),
		},
		&models.CollectionI18n{
			Language: common.LANG_ENGLISH,
			Name:     null.StringFrom("Convention"),
		})
	suite.Require().Nil(err)

	cu, err := CreateContentUnit(suite.tx, common.CT_LESSON_PART, nil)
	suite.Require().Nil(err)

	metadata := CITMetadata{
		CollectionUID:  null.StringFrom(c.UID),
		ContentType:    common.CT_LESSON_PART,
		AutoName:       "auto_name",
		FinalName:      "final_name",
		CaptureDate:    Date{time.Now()},
		Language:       common.LANG_HEBREW,
		HasTranslation: true,
		Lecturer:       "rav",
		Number:         null.IntFrom(3),
		Part:           null.IntFrom(0),
	}

	describer := new(EventPartDescriber)
	i18ns, err := describer.DescribeContentUnit(suite.tx, cu, metadata)
	suite.Require().Nil(err)
	suite.NotEmpty(i18ns, "i18ns.empty")
	for _, i18n := range i18ns {
		switch i18n.Language {
		case common.LANG_HEBREW:
			suite.Equal("כנס. הכנה לשיעור 3", i18n.Name.String, "Hebrew name")
		case common.LANG_ENGLISH:
			suite.Equal("Convention. Preparation to the Lesson 3", i18n.Name.String, "English name")
		}
	}
}

func (suite *AutonameSuite) TestBlogPostDescriber() {
	cu, err := CreateContentUnit(suite.tx, common.CT_BLOG_POST, map[string]interface{}{"original_language": "ru"})
	suite.Require().Nil(err)

	metadata := CITMetadata{}
	describer := new(BlogPostDescriber)
	i18ns, err := describer.DescribeContentUnit(suite.tx, cu, metadata)
	suite.Require().Nil(err)
	suite.NotEmpty(i18ns, "i18ns.empty")
	for _, i18n := range i18ns {
		switch i18n.Language {
		case common.LANG_HEBREW:
			suite.Equal("הבלוג של רב ד\"ר מיכאל לייטמן – גרסת אודיו (רוסית)", i18n.Name.String, "Hebrew name")
		case common.LANG_ENGLISH:
			suite.Equal("Dr. Michael Laitman Blog - Audio Version (Russian)", i18n.Name.String, "English name")
		case common.LANG_RUSSIAN:
			suite.Equal("Радио-версия блога д-ра Михаэля Лайтмана (Русский)", i18n.Name.String, "Russian name")
		case common.LANG_SPANISH:
			suite.Equal("Blog de Rav M. Laitman - Versión Audio (Ruso)", i18n.Name.String, "Spanish name")
		}
	}
}

func (suite *AutonameSuite) TestDescribeCollection() {
	c, err := CreateCollection(suite.tx, common.CT_UNKNOWN, nil)
	suite.Require().Nil(err)

	err = DescribeCollection(suite.tx, c)
	suite.Require().Nil(err)
	err = c.L.LoadCollectionI18ns(suite.tx, true, c)
	suite.Require().Nil(err)
	i18ns := c.R.CollectionI18ns

	names, err := GetI18ns("content_type.UNKNOWN")
	suite.Require().Nil(err)
	suite.Len(i18ns, len(names), "len(i18ns)")
	for _, i18n := range i18ns {
		suite.Equal(names[i18n.Language], i18n.Name.String, "%s name", i18n.Language)
	}
}

func (suite *AutonameSuite) TestSourceNamers() {
	author := &models.Author{ID: -1}
	author.L.LoadAuthorI18ns(suite.tx, true, author) // dummy call to initialize R
	author.R.AuthorI18ns = make(models.AuthorI18nSlice, 0)
	author.R.AuthorI18ns = append(author.R.AuthorI18ns,
		&models.AuthorI18n{
			Language: common.LANG_HEBREW,
			Name:     null.StringFrom("author"),
		},
		&models.AuthorI18n{
			Language: common.LANG_ENGLISH,
			Name:     null.StringFrom("author"),
		},
	)

	path := make([]*models.Source, 4)
	for i := 1; i < 5; i++ {
		s := &models.Source{ID: -1}
		s.L.LoadSourceI18ns(suite.tx, true, s) // dummy call to initialize R
		s.R.SourceI18ns = make(models.SourceI18nSlice, 0)
		s.R.SourceI18ns = append(s.R.SourceI18ns,
			&models.SourceI18n{
				Language: common.LANG_HEBREW,
				Name:     null.StringFrom(fmt.Sprintf("source %d", i)),
			})
		path[i-1] = s
	}

	var namer sourceNamer
	namer = new(PlainNamer)
	names, err := namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name := names[common.LANG_HEBREW]
	suite.Equal("author. source 1. source 2. source 3. source 4", name, "name")

	namer = new(PrefaceNamer)
	names, err = namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name = names[common.LANG_HEBREW]
	suite.Equal("author. source 4", name, "name")

	namer = new(LettersNamer)
	path[len(path)-1].R.SourceI18ns[0].Name = null.StringFrom("source 4 (1920)")
	names, err = namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name = names[common.LANG_HEBREW]
	suite.Equal("author. source 4", name, "name")
	path[len(path)-1].R.SourceI18ns[0].Name = null.StringFrom("source 4")

	namer = new(RBRecordsNamer)
	path[len(path)-1].Position = null.IntFrom(137)
	names, err = namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name = names[common.LANG_HEBREW]
	suite.Equal("author. רשומה 137. source 4", name, "name")

	namer = new(RBArticlesNamer)
	path[len(path)-1].Name = "(1984-01-2) Matarat Hevra 2"
	names, err = namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name = names[common.LANG_HEBREW]
	suite.Equal("author. source 4. 1-2 (1984)", name, "name")

	namer = new(ShamatiNamer)
	path[len(path)-1].Name = "015 some name"
	names, err = namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name = names[common.LANG_HEBREW]
	suite.Equal("author. source 1, טו. source 4", name, "name")

	path[1].R.SourceI18ns[0].Description = null.StringFrom("source 2 description")
	namer = new(ZoharNamer)
	names, err = namer.GetName(author, path)
	suite.Require().Nil(err)
	suite.Len(names, 1, "len(names)")
	name = names[common.LANG_HEBREW]
	suite.Equal("source 1. source 3. source 4", name, "name")
}

func (suite *AutonameSuite) TestNameByTag() {
	t := &models.Tag{ID: 88, UID: "1nyptSIo"}
	suite.Require().Nil(t.Insert(suite.tx))
	for _, lang := range [...]string{
		common.LANG_HEBREW,
		common.LANG_ENGLISH,
		common.LANG_RUSSIAN,
		common.LANG_SPANISH,
		common.LANG_ITALIAN,
		common.LANG_GERMAN,
		common.LANG_UKRAINIAN,
		common.LANG_TURKISH} {
		i18n := &models.TagI18n{
			TagID:    t.ID,
			Language: lang,
			Label:    null.StringFrom(fmt.Sprintf("tag lang %s", lang)),
		}
		suite.Require().Nil(i18n.Insert(suite.tx))
	}

	cNumber := new(int)
	*cNumber = 1
	names, err := nameByTagUID(suite.tx, t.UID, cNumber)
	suite.Require().Nil(err)
	suite.Len(names, 8, "len(names)")
}

func (suite *AutonameSuite) TestNameByLikutim() {
	likutim, err := SomeLikutim(suite.tx)
	suite.Require().Nil(err)

	names, err := nameByLikutimUID(suite.tx, likutim[0].UID)
	suite.Require().Nil(err)

	var name null.String
	for _, l := range likutim[0].R.ContentUnitI18ns {
		if l.Language == common.LANG_HEBREW {
			name = l.Name
		}
	}
	suite.NotNil(name)
	suite.Equal(name.String, names[common.LANG_HEBREW], "len(names)")
}
