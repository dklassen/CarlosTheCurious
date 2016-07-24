package slackbot

import (
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // we use the postgres driver
)

var database *gorm.DB

// Migrate is at the moment where we list all the Gorm Models we are using so
// we can perform a db migration. This is probably no the best solution to the problem
// oh rails I miss you
// TODO:: Find out how to register the gorm models somehwere automatically if possible
func Migrate() {
	err := GetDB().AutoMigrate(
		&Poll{},
		&PossibleAnswer{},
		&Recipient{},
		&PollResponse{},
	).Error

	if err != nil {
		logrus.Error(err)
	}
}

// DropDatabaseTables drops all the database tables cold turkey
// Really could just execute a drop database and recreate which would be quicker
// TODO:: Find out how to register the gorm models somehwere automatically if possible
func DropDatabaseTables() {
	err := GetDB().DropTableIfExists(
		&Poll{},
		&PossibleAnswer{},
		&Recipient{},
		&PollResponse{},
	).Error

	if err != nil {
		logrus.Error(err)
	}
}

// GetDB is a accessor for a shared db object
func GetDB() *gorm.DB {
	if database == nil {
		logrus.Panic("Database was not instantiated. Call SetupDatabase during application setup")
	}
	return database
}

func verifyDatabaseConnection(db *gorm.DB) {
	if err := db.DB().Ping(); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Successfully connected to database")
}

// SetupDatabase opens and verifies the connection to the database
func SetupDatabase(databaseURL string, debug bool) error {
	var err error
	logrus.WithField("connectionString", databaseURL).Info("Attempting to connect to database")
	database, err = gorm.Open("postgres", databaseURL)
	if err != nil {
		logrus.Panic(err)
	}
	verifyDatabaseConnection(database)
	database.DB().SetMaxIdleConns(10)
	database.DB().SetMaxOpenConns(100)
	database.LogMode(debug)

	return err
}
