package slackbot

type ResponseRTMStart struct {
	Ok    bool             `json:"ok"`
	Error string           `json:"error"`
	URL   string           `json:"url"`
	Self  *ResponseRTMSelf `json:"self"`
}

type ResponseRTMSelf struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	ID            uint64   `json:"id"`
	Type          string   `json:"type"`
	Subtype       string   `json:"subtype"`
	Channel       string   `json:"channel"`
	User          string   `json:"user"`
	Text          string   `json:"text"`
	Timestamp     string   `json:"ts"`
	Handled       bool     `json:"-"` // Did message match a handler?
	DirectMention bool     `json:"-"` // Does message contain a direct mention
	CaptureGroup  []string `json:"-"` // hold the capture group when a command is matched
}

type Attachment struct {
	Timestamp      int               `json:"ts"`
	Title          string            `json:"title"`
	TitleLink      string            `json:"title_link"`
	ImageURL       string            `json:"image_url"`
	ThumbURL       string            `json:"thumb_url"`
	Footer         string            `json:"footer"`
	FooterIcon     string            `json:"footer_icon"`
	Text           string            `json:"text"`
	Fallback       string            `json:"fallback"`
	CallbackID     string            `json:"callback_id"`
	Color          string            `json:"color"`
	AttachmentType string            `json:"attachment_type"`
	Actions        []Action          `json:"actions"`
	Author         string            `json:"author"`
	AuthorName     string            `json:"author_name"`
	AuthorLink     string            `json:"author_link"`
	AuthorIcon     string            `json:"author_icon"`
	Pretext        string            `json:"pretext"`
	Fields         []AttachmentField `json:"fields"`
}

type Action struct {
	Name    string        `json:"name"`
	Text    string        `json:"text"`
	Type    string        `json:"type"`
	Value   string        `json:"value"`
	Style   string        `json:"style"`
	Confirm ActionConfirm `json:"confirm"`
}

type ActionConfirm struct {
	Title       string `json:"title"`
	Text        string `json:"text"`
	OkText      string `json:"ok_text"`
	DismissText string `json:"ok_text"`
}

type AttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type PostResponse struct {
	Ok      bool   `json:"ok"`
	TS      string `json:"ts"`
	Channel string `json:"channel"`
	Error   string `json:"error"`
}
