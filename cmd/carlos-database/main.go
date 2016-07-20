package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dklassen/carlos-the-curious/slackbot"

	logrus "github.com/Sirupsen/logrus"
)

var (
	usage = fmt.Sprintf(`Usage: %s COMMAND
Valid commands:
	nuke - Nuke the database and migrate back to ground zero
	migrate - Migrate the database to the latest schema
`, os.Args[0])
)

func nuke() {
	startedAt := time.Now()
	logrus.Info("Going to drop the database like its hot")
	slackbot.DropDatabaseTables()
	finishedAt := time.Now()
	duration := finishedAt.Sub(startedAt)
	logrus.WithFields(logrus.Fields{
		"started_at":  startedAt,
		"finished_at": finishedAt,
		"took":        duration.Seconds()}).Info("Finished dropping the database")
	migrate()
}

func migrate() {
	startedAt := time.Now()
	logrus.Info("Starting database migration")
	slackbot.Migrate()
	finishedAt := time.Now()
	duration := finishedAt.Sub(startedAt)
	logrus.WithFields(logrus.Fields{
		"started_at":  startedAt,
		"finished_at": finishedAt,
		"took":        duration.Seconds()}).Info("Finished database migration")
}

func main() {
	conf, err := slackbot.LoadFromFlags()
	if err != nil {
		logrus.Fatal("error", err)
	}

	if len(flag.Args()) != 1 {
		logrus.Fatal(usage)
	}

	err = slackbot.SetupDatabase(*conf)
	if err != nil {
		logrus.Panic(err)
	}

	command := flag.Args()[0]
	switch command {
	case "nuke":
		nuke()
	case "migrate":
		migrate()
	default:
		logrus.Fatal(usage)
	}
}
