package main

import (
	logrus "github.com/Sirupsen/logrus"
	"github.com/dklassen/carlos-the-curious/slackbot"
)

func mustLoadConfig() *slackbot.Config {
	conf, err := slackbot.LoadFromFlags()
	if err != nil {
		logrus.Fatal("error", err)
	}
	return conf
}

func main() {
	conf := mustLoadConfig()
	logrus.Info("Starting Carlos the Curious")
	slackbot.SetupDatabase(conf.DatabaseURL, conf.Debug)

	robot := slackbot.NewRobot(conf.Origin, conf.SlackAPIToken)
	robot.Run()
}
