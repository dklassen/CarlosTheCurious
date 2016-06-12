package main

import (
	logrus "github.com/Sirupsen/logrus"
	"github.com/dklassen/CarlosTheCurious/slackbot"
)

func mustLoadConfig() *slackbot.Config {
	conf, err := slackbot.LoadConfig()
	if err != nil {
		logrus.Fatal("error", err)
	}
	return conf
}

func main() {
	conf := mustLoadConfig()

	logrus.WithFields(logrus.Fields{
		"message_workers": conf.Workers,
		"origin":          conf.Origin,
	}).Info("Starting Carlos the Curious")

	slackbot.SetupDatabase(conf.DatabaseURL, conf.Debug)
	slackbot.Run(conf.Origin, conf.SlackAPIToken, conf.Workers)
}
