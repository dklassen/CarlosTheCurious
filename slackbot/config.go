package slackbot

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

// Config holds all the data loaded from the CLI flags
type Config struct {
	// Slack token for the bot
	SlackAPIToken string `json:"slack_api_key"`

	// Origin is the api root we are setting up a websocket for
	Origin string `json:"origin"`

	// To debug or not to debug
	Debug bool `json:"debug"`

	// DatabaseConnectionString is the full connection string that is passed in
	DatabaseURL string `json:"database_url"`

	// Workers is the number of goroutines to spin up for processing the message queue
	Workers int `json:"workers"`
}

var (
	token       = flag.String("token", "", "Slack authentication token")
	databaseURL = flag.String("database_url", "", "Name of the database we are connecting to")
	origin      = flag.String("origin", "https://api.slack.com", "Slack origin url")
	debug       = flag.Bool("debug", false, "Enable debug mode")
	workers     = flag.Int("workers", 4, "Configure the number of message workers")
)

// LoadFromFlags loads all global config from CLI flags
func LoadFromFlags() (*Config, error) {
	flag.Parse()
	return &Config{
		SlackAPIToken: *token,
		DatabaseURL:   *databaseURL,
		Origin:        *origin,
		Debug:         *debug,
		Workers:       *workers,
	}, nil
}

func configFilePath() string {
	root := os.Getenv("GOPATH")
	return filepath.Join(root, "src/github.com/dklassen/CarlosTheCurious/config/config.json")
}

func LoadFromFile() (*Config, error) {
	config := &Config{}

	file, err := ioutil.ReadFile(configFilePath())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, config)
	return config, err
}

func LoadConfig() (*Config, error) {
	config_file, err := LoadFromFile()
	if err != nil {
		logrus.Errorf("Error loading config file: %v", err)
	}

	config_flags, err := LoadFromFlags()
	if err != nil {
		logrus.Fatal(err)
	}

	//TODO:: Merge structs in a nicer way
	config := Config{}
	if config_flags.SlackAPIToken == "" {
		config.SlackAPIToken = config_file.SlackAPIToken
	} else {
		config.SlackAPIToken = config_flags.SlackAPIToken
	}

	if config_flags.DatabaseURL == "" {
		config.DatabaseURL = config_file.DatabaseURL
	} else {
		config.DatabaseURL = config_flags.DatabaseURL
	}

	if config_flags.Origin == "" {
		config.Origin = config_file.Origin
	} else {
		config.Origin = config_flags.Origin
	}

	if config_flags.Debug == false {
		config.Debug = config_file.Debug
	} else {
		config.Debug = config_flags.Debug
	}

	if config_flags.Workers == 0 {
		config.Workers = config_file.Workers
	} else {
		config.Workers = config_flags.Workers
	}
	return &config, nil
}
