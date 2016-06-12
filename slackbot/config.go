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
	DatabaseConnectionString string
}

var (
	token    = flag.String("token", "", "Slack authentication token")
	database = flag.String("database", "", "Database host and port information")
	origin   = flag.String("origin", "https://api.slack.com", "Slack origin url")
	debug    = flag.Bool("debug", false, "Enable debug mode")
)

// LoadFromFlags loads all global config from CLI flags
func LoadFromFlags() (*Config, error) {
	flag.Parse()
	return &Config{
		SlackAPIToken:            *token,
		DatabaseConnectionString: *database,
		Origin: *origin,
		Debug:  *debug,
	}, nil
}
