package slackbot

import (
	"flag"
)

// Config holds all the data loaded from the CLI flags
type Config struct {
	// Slack token for the bot
	SlackAPIToken string

	// Origin is the api root we are seting up a websocket for
	Origin string

	// To debug or not to debug
	Debug bool

	// DatabaseConnectionString is the full connection string that is passed in
	DatabaseName string

	DatabaseUser string

	DatabaseHost string

	DatabaseSSL string
}

var (
	token        = flag.String("token", "", "Slack authentication token")
	database     = flag.String("database", "", "Name of the database we are connecting to")
	databaseUser = flag.String("database_user", "", "User we are connecting to the database with")
	databaseHost = flag.String("database_host", "", "Hostwe are connecting to the database with")
	databaseSSL  = flag.String("database_ssl", "", "SSL mode we are using for the database")
	origin       = flag.String("origin", "https://api.slack.com", "Slack origin url")
	debug        = flag.Bool("debug", false, "Enable debug mode")
)

// LoadFromFlags loads all global config from CLI flags
func LoadFromFlags() (*Config, error) {
	flag.Parse()
	return &Config{
		SlackAPIToken: *token,
		DatabaseName:  *database,
		DatabaseUser:  *databaseUser,
		DatabaseHost:  *databaseHost,
		DatabaseSSL:   *databaseSSL,
		Origin:        *origin,
		Debug:         *debug,
	}, nil
}
