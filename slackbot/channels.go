package slackbot

type ChannelList struct {
	Ok       bool      `json:"ok"`
	Channels []Channel `json:"channels"`
	Error    string    `json:"error,omitempty"`
}

type Channel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	IsChannel string   `json:"is_channel"`
	Members   []string `json:"members"`
}
