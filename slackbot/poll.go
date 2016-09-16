package slackbot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
)

const (
	ResponsePoll = "response"
	FeedbackPoll = "feedback"
)

var (
	ErrExistingInactivePoll = errors.New("CarlosTheCurious: Unable to create poll due to partially created existing poll")
	ErrInvalidPollType      = errors.New("CarlosTheCurious: Invalid poll type must be of response or feedback")
)

type Poll struct {
	gorm.Model
	UUID    string `gorm:"not null;unique"`
	Channel string `gorm:"not null"`
	Creator string `gorm:"not null"`

	// The creation stage the poll is in initial -> getQuestion -> getRecipient -> Active -> Cancelled or Archived
	Stage string

	// The stage that proceeded the current stage.
	// NOTE:: This is a bit of a hack since I feel like a separate state change table or log would be a better solution
	// for tracking state changing history. For now we only need to action on the previous state transition
	// for instance moving from getAnswers -> paused and than continuing
	PreviousStage string

	// Represents the kind of poll this is. Feedback or Response. Feedback polls as for free text responses, while reponse polls
	// take a list of possible responses
	Kind string `gorm:"not null"`

	Question string

	// We track recipients at the user level. Each recipient is a user
	Recipients      []Recipient
	Responses       []PollResponse
	PossibleAnswers []PossibleAnswer
}

type PossibleAnswer struct {
	gorm.Model
	PollID uint
	Value  string
}

type PollResponse struct {
	gorm.Model
	PollID  uint
	SlackID string
	Value   string
}

type Recipient struct {
	gorm.Model
	SlackID   string
	PollID    uint
	SlackName string
}

func NewRecipient(id string) (*Recipient, error) {
	idType := TypeOfSlackID(id)
	if idType != UserID {
		return nil, errors.New(fmt.Sprintf("A recipient must be a user id not ", id))
	}

	return &Recipient{SlackID: id}, nil
}

func NewPoll(kind, creator, channel string) *Poll {
	uuid := GenerateUUID()
	return &Poll{
		UUID:            uuid,
		Kind:            kind,
		Creator:         creator,
		Channel:         channel,
		PreviousStage:   "initial",
		Stage:           "initial",
		PossibleAnswers: []PossibleAnswer{},
	}
}

func (poll *Poll) Save() error {
	return GetDB().Save(&poll).Error
}

func (poll *Poll) TransitionTo(nextStage string) error {
	poll.PreviousStage = poll.Stage
	poll.Stage = nextStage
	return poll.Save()
}

func (poll *Poll) AddRecipient(recipient Recipient) error {
	return GetDB().
		Model(poll).
		Association("Recipients").
		Append(recipient).Error
}

func (poll *Poll) SetRecipients(recipients []Recipient) error {
	poll.Recipients = recipients
	return poll.Save()
}

func (poll *Poll) GetRecipients() ([]Recipient, error) {
	recipients := []Recipient{}
	err := GetDB().Model(poll).Related(&recipients).Error
	return recipients, err
}

func FindFirstPollByStage(name, stage string) *Poll {
	poll := &Poll{}
	GetDB().Where("name = ? AND stage = ?", name, stage).First(poll)
	if poll.ID == 0 {
		return nil
	}
	return poll
}

func ValidResponse(response string) bool {
	rows, _ := GetDB().Raw("select 1 from possible_answers where value = ?", response).Rows()

	defer rows.Close()
	var count int

	for rows.Next() {
		rows.Scan(&count)
	}

	if count == 0 {
		return false
	}
	return true
}

func (p *Poll) AddResponse(userID, responseValue string) error {
	if p.Kind == ResponsePoll && !ValidResponse(responseValue) {
		return errors.New(fmt.Sprintf("Invalid response %s", responseValue))
	}

	response := PollResponse{Value: responseValue, SlackID: userID}
	return GetDB().Model(p).Association("Responses").Append(response).Error
}

func (poll *Poll) GetAnswers() ([]PossibleAnswer, error) {
	answers := []PossibleAnswer{}
	err := GetDB().Model(poll).Association("PossibleAnswers").Find(&answers).Error
	return answers, err
}

func (poll *Poll) GetResponses() ([]PollResponse, error) {
	responses := []PollResponse{}
	err := GetDB().Model(poll).Association("Responses").Find(&responses).Error
	return responses, err
}

// FindFirstPreActivePollByName finds the first preactive poll based on name
// NOTE:: OMG this is bad lets get rid of these
func FindFirstPreActivePollByName(name string) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("uuid = ? AND stage != ?", name, "active").First(poll)

	if poll.ID == 0 {
		return poll, fmt.Errorf("No inactive poll with %s found", name)
	}
	return poll, nil
}

func FindFirstActivePollByUUID(uuid string) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("uuid = ? AND stage = ?", uuid, "active").First(poll)

	if poll.ID == 0 {
		return poll, fmt.Errorf("No active poll with %s found", uuid)
	}
	return poll, nil
}

func FindFirstActivePollByMessage(msg Message) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage = ?", msg.User, msg.Channel, "active").First(&poll)

	if poll.ID != 0 {
		return poll, fmt.Errorf("No active poll found")
	}
	return poll, nil
}

func FindFirstInactivePollByMessage(msg *Message) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage != ?", msg.User, msg.Channel, "active").First(&poll)

	if poll.ID == 0 {
		return poll, fmt.Errorf("No poll found")
	}
	return poll, nil
}

func FindRecipientByID(pollID uint, slackID string) Recipient {
	recipient := Recipient{}
	GetDB().Where("poll_id = ? AND slack_id = ?", pollID, slackID).First(&recipient)
	return recipient
}

func (r *Recipient) SlackIDString() string {
	return "<@" + r.SlackID + ">"
}

func (poll *Poll) numberOfRecipients() int {
	return GetDB().Model(&poll).Association("Recipients").Count()
}

func (poll *Poll) numberOfResponses() int {
	var numOfResponses int
	GetDB().Model(&PollResponse{}).Where("poll_id = ? AND value is NOT NULL", poll.ID).Count(&numOfResponses)
	return numOfResponses
}

func (poll *Poll) slackAnswerString() string {
	answerString := ""
	possibleAnswers := []PossibleAnswer{}
	GetDB().Model(&poll).Related(&possibleAnswers)
	for k, a := range possibleAnswers {
		if k == 0 {
			answerString = a.Value
		} else {
			answerString = answerString + ", " + a.Value
		}
	}
	return answerString
}

func responseSummaryField(poll *Poll) *AttachmentField {
	total := poll.numberOfRecipients()
	responded := poll.numberOfResponses()
	responseRatio := (float64(responded) / float64(total)) * 100

	return &AttachmentField{
		Title: "Response Stats:",
		Value: fmt.Sprintf("%d%% - %d out of %d", int(responseRatio), responded, total),
		Short: false,
	}
}

func possibleAnswerField(poll *Poll) AttachmentField {
	return AttachmentField{
		Title: "Possible Answers:",
		Value: poll.slackAnswerString(),
		Short: false,
	}
}

func recipientsField(poll *Poll) AttachmentField {
	return AttachmentField{
		Title: "# of Recipients:",
		Value: fmt.Sprintf("%d", poll.numberOfRecipients()),
		Short: false,
	}
}

func pollTypeField(poll *Poll) AttachmentField {
	return AttachmentField{
		Title: "Poll Type:",
		Value: poll.Kind,
		Short: true,
	}
}

func feedbackResponseField(responses []PollResponse) *AttachmentField {
	results := ""
	for i, resp := range responses {
		results += fmt.Sprintf("%d \n", i+1) + resp.Value
	}

	return &AttachmentField{
		Title: "Responses:",
		Value: results,
		Short: false,
	}
}

func responseField(poll *Poll) *AttachmentField {
	query := `SELECT b.value as possibleAnswer,
				  count(a.id) as responses
	FROM poll_responses as a
	FULL JOIN possible_answers as b
	ON a.poll_id = b.poll_id AND a.value = b.value
	WHERE b.poll_id = ?
	GROUP BY b.value, a.value`

	rows, err := GetDB().Raw(query, poll.ID).Rows()
	if err != nil {
		logrus.Panic(err)
	}

	defer rows.Close()

	var possibleAnswer string
	var responses int

	totalNumberOfRecipients := poll.numberOfRecipients()

	summary := ""
	first := true
	for rows.Next() {
		rows.Scan(&possibleAnswer, &responses)
		responsePercent := (float64(responses) / float64(totalNumberOfRecipients)) * 100

		if first {
			first = false
			summary += fmt.Sprintf("%s - %d(%d%%)", possibleAnswer, responses, int(responsePercent))
		} else {
			summary += " | " + fmt.Sprintf("%s - %d(%d%%)", possibleAnswer, responses, int(responsePercent))
		}
	}

	return &AttachmentField{
		Title: "Responses:",
		Value: summary,
		Short: false,
	}

}

func (poll *Poll) SlackPollSummary() Attachment {
	attachments := []AttachmentField{}

	if poll.Kind == ResponsePoll {
		attachments = append(attachments, *responseField(poll))
	} else {
		responses, err := poll.GetResponses()
		if err != nil {
			logrus.Panic(err)
		}
		attachments = append(attachments, *feedbackResponseField(responses))
	}

	attachments = append(attachments, *responseSummaryField(poll))

	return Attachment{
		Color:  "#36a64f",
		Title:  "Survey Results",
		Text:   poll.Question + "\n\n",
		Fields: attachments,
	}
}

func (poll *Poll) SlackPreviewAttachment() Attachment {
	attachments := []AttachmentField{}

	attachments = append(attachments, recipientsField(poll))

	if poll.Kind == "response" {
		attachments = append(attachments, possibleAnswerField(poll))
	}

	title := fmt.Sprintf("%s Question:", strings.Title(poll.Kind))

	return Attachment{
		Title:   title,
		Pretext: "Look good to you (yes/no)?",
		Text:    poll.Question,
		Fields:  attachments,
	}
}

func (poll *Poll) SlackRecipientAttachment() Attachment {
	attachments := []AttachmentField{}

	attachments = append(attachments, pollTypeField(poll))

	if poll.Kind == ResponsePoll {
		attachments = append(attachments, possibleAnswerField(poll))
	}

	return Attachment{
		Title:   "Question",
		Pretext: fmt.Sprintf("We have a question for you. You can answer via `answer poll %s {insert response}`", poll.UUID),
		Text:    poll.Question,
		Fields:  attachments,
	}
}
