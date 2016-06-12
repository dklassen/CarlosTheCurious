package slackbot

import (
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // we use the postgres driver
)

var db *gorm.DB

// Migrate is at the moment where we list all the Gorm Models we are using so
// we can perform a db migration. This is probably no the best solution to the problem
// oh rails I miss you
// TODO:: Find out how to register the gorm models somehwere automatically if possible
func Migrate() {
	GetDB().AutoMigrate(
		&Poll{},
		&PossibleAnswer{},
		&Recipient{},
		&PollResponse{},
	)
}

// DropDatabaseTables drops all the database tables cold turkey
// Really could just execute a drop database and recreate which would be quicker
// TODO:: Find out how to register the gorm models somehwere automatically if possible
func DropDatabaseTables() {
	GetDB().DropTableIfExists(
		&Poll{},
		&PossibleAnswer{},
		&Recipient{},
		&PollResponse{},
	)
}

// GetDB is a accessor for a shared db object
func GetDB() *gorm.DB {
	if db == nil {
		logrus.Panic("Database was not instantiated. Call SetupDatabase during application setup")
	}
	return db
}

func verifyDatabaseConnection(db *gorm.DB) {
	if err := db.DB().Ping(); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Successfully connected to database")
}

// SetupDatabase opens and verifies the connection to the database
func SetupDatabase(config Config) (err error) {
	logrus.WithField("connectionString", config.DatabaseConnectionString).Info("Attempting to connect to database")
	database, err := gorm.Open("postgres", config.DatabaseConnectionString)
	if err != nil {
		logrus.Panic(err)
	}
	db = &database
	verifyDatabaseConnection(db)
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	db.LogMode(config.Debug)

	return
}
