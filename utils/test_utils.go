package utils

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Bnei-Baruch/mdb/migrations"

	// List all test dependencies here until this bug is fixed in godeps
	// which should allow us to use `godeps save -t`
	// https://github.com/tools/godep/issues/405
	_ "github.com/adams-sarah/test2doc/test"
	_ "github.com/stretchr/testify"

	// used in models tests generated by sqlboiler
	_ "github.com/kat-co/vala"
	_ "github.com/volatiletech/sqlboiler/bdb"
	_ "github.com/volatiletech/sqlboiler/bdb/drivers"
	_ "github.com/volatiletech/sqlboiler/randomize"
	_ "github.com/volatiletech/sqlboiler/strmangle"
	"time"
)

var UID_REGEX = regexp.MustCompile("[a-zA-z0-9]{8}")

type TestDBManager struct {
	DB     *sql.DB
	testDB string
}

func (m *TestDBManager) InitTestDB() error {
	m.testDB = fmt.Sprintf("test_%s", strings.ToLower(GenerateName(10)))
	fmt.Println("Initializing test DB: ", m.testDB)

	m.initConfig()

	// Open connection to RDBMS
	db, err := sql.Open("postgres", viper.GetString("mdb.url"))
	if err != nil {
		return err
	}

	// Create a new temporary test database
	if _, err := db.Exec("CREATE DATABASE " + m.testDB); err != nil {
		return err
	}

	// Close first connection and connect to temp database
	db.Close()
	fmt.Println(fmt.Sprintf(viper.GetString("test.url-template"), m.testDB))
	db, err = sql.Open("postgres", fmt.Sprintf(viper.GetString("test.url-template"), m.testDB))
	if err != nil {
		return err
	}

	// Run migrations
	m.runMigrations(db)

	time.Sleep(500 * time.Millisecond)

	// Setup SQLBoiler
	m.DB = db
	//boil.SetDB(db)
	//boil.DebugMode = viper.GetBool("test.debug-sql")

	return nil
}

func (m *TestDBManager) DestroyTestDB() error {
	fmt.Println("Destroying testDB: ", m.testDB)

	// Close temp DB
	err := m.DB.Close()
	//err := boil.GetDB().(*sql.DB).Close()
	if err != nil {
		return err
	}

	// Connect to MDB
	db, err := sql.Open("postgres", viper.GetString("mdb.url"))
	if err != nil {
		return err
	}

	// Drop test DB
	_, err = db.Exec("DROP DATABASE " + m.testDB)
	if err != nil {
		return err
	}

	return nil
}

func (m *TestDBManager) initConfig() {
	viper.SetDefault("test", map[string]interface{}{
		"url-template": "postgres://localhost/%s?sslmode=disable&?user=postgres",
		"debug-sql":    true,
	})

	viper.SetConfigName("config")
	viper.AddConfigPath("../")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Could not read config, using: ", viper.ConfigFileUsed(), err.Error())
	}

	viper.Debug()

	log.SetLevel(log.ErrorLevel)
}

func (m *TestDBManager) runMigrations(db *sql.DB) error {
	var visit = func(path string, f os.FileInfo, err error) error {
		match, _ := regexp.MatchString(".*\\.sql$", path)
		if !match {
			return nil
		}

		//fmt.Printf("Applying migration %s\n", path)
		m, err := migrations.NewMigration(path)
		if err != nil {
			fmt.Printf("Error migrating %s, %s", path, err.Error())
			return err
		}

		fmt.Printf("running migration: %s\n", path)
		for _, statement := range m.Up() {
			if strings.HasSuffix(path, "dev.sql") {
				fmt.Println(statement)
			}
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("Unable to apply migration %s: %s\nStatement: %s\n", m.Name, err, statement)
			}
		}

		return nil
	}

	return filepath.Walk("../migrations", visit)
}

func Sha1(s string) string {
	h := sha1.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func RandomSHA1() string {
	return Sha1(GenerateName(1024))
}
