package main

import (
	logrus "github.com/Sirupsen/logrus"
	"github.com/dklassen/CarlosTheCurious/slackbot"
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
	slackbot.Run(conf.Origin, conf.SlackAPIToken)
}
