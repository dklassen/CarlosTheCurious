package slackbot

type GroupList struct {
	Ok     bool    `json:"ok"`
	Groups []Group `json:"groups"`
	Error  string  `json:"error,omitempty"`
}

type Group struct {
	ID         string   `json:"id"`
	Created    uint     `json:"created"`
	Creator    string   `json:"creator"`
	IsArchived bool     `json:"is_archived"`
	IsGroup    bool     `json:"is_group"`
	IsMPIM     bool     `json:"is_mpim"`
	Members    []string `json:"members"`
	Name       string   `json:"name"`
}
