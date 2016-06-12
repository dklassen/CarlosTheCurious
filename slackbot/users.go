package slackbot

type User struct {
	Name string `json:"name"`

	// SlackID is the string identifier for a team member
	SlackID      string       `json:"id"`
	SlackProfile SlackProfile `json:"profile"`
}

// Slack collection of users
type UserList struct {
	Ok      bool   `json:"ok"`
	Members []User `json:"members"`
	Error   string `json:"error,omitempty"`
}

// SlackProfile contains profile information from Slack regarding the user
type SlackProfile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	RealName  string `json:"real_name"`
	Email     string `json:"email"`
}
