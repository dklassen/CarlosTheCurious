package slackbot

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	ResponsePoll = "response"
	FeedbackPoll = "feedback"
)

var (
	ErrExistingInactivePoll = errors.New("CarlosTheCurious: Existing poll is being created. Cancel or continue")
	ErrInvalidPollType      = errors.New("CarlosTheCurious: Invalid poll type")
)

type Poll struct {
	gorm.Model
	UUID            string `gorm:"not null;unique"`
	Channel         string `gorm:"not null"`
	Creator         string `gorm:"not null"`
	Stage           string
	Kind            string `gorm:"not null"`
	Question        string
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

func NewPoll(kind, creator, channel string) *Poll {
	uuid := GenerateUUID()
	return &Poll{
		UUID:            uuid,
		Kind:            kind,
		Creator:         creator,
		Channel:         channel,
		Stage:           "initial",
		PossibleAnswers: []PossibleAnswer{},
	}
}

func FindFirstPollByStage(name, stage string) *Poll {
	poll := &Poll{}
	GetDB().Where("name = ? AND stage = ?", name, stage).First(poll)
	if poll.ID == 0 {
		return nil
	}
	return poll
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

// FindFirstActivePollByName does what it says and finds the first poll it can that is
// in the active state
// NOTE:: OMG this is bad lets get rid of these
func FindFirstActivePollByUUID(uuid string) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("uuid = ? AND stage = ?", uuid, "active").First(poll)

	if poll.ID == 0 {
		return poll, fmt.Errorf("No active poll with %s found", uuid)
	}
	return poll, nil
}

// FindFirstActivePollByMessage finds the first active poll using the user and channel
// NOTE:: OMG this is bad lets get rid of these
func FindFirstActivePollByMessage(msg Message) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage = ?", msg.User, msg.Channel, "active").First(&poll)

	if poll.ID != 0 {
		return poll, fmt.Errorf("No active poll found")
	}
	return poll, nil
}

// FindFirstActivePollByMessage finds the first active poll using the user and channel
// NOTE:: OMG this is bad lets get rid of these
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

func (poll *Poll) slackRecipientString() string {
	recipientString := ""
	recipients := []Recipient{}
	GetDB().Model(&poll).Related(&recipients)

	for k, r := range recipients {
		if k == 0 {
			recipientString = r.SlackIDString()
		} else {
			recipientString = recipientString + ", " + r.SlackIDString()
		}
	}
	return recipientString
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

func responseSummaryField(poll *Poll) AttachmentField {
	total := GetDB().Model(&poll).Association("Recipients").Count()

	var responded int
	GetDB().Model(&PollResponse{}).Where("poll_id = ? AND value is NOT NULL", poll.ID).Count(&responded)
	responseRatio := (responded / total) * 100

	return AttachmentField{
		Title: "Response Stats:",
		Value: fmt.Sprintf("%d%% - %d out of %d", responseRatio, responded, total),
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
		Title: "Recipients:",
		Value: poll.slackRecipientString(),
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

func (poll *Poll) SlackPollSummary() Attachment {
	return Attachment{
		Title:   poll.UUID,
		Pretext: "Here are the results so far:",
		Text:    poll.Question,
		Fields: []AttachmentField{
			recipientsField(poll),
			possibleAnswerField(poll),
			responseSummaryField(poll),
		},
	}
}

func (poll *Poll) SlackPreviewAttachment() Attachment {
	attachments := []AttachmentField{}

	attachments = append(attachments, pollTypeField(poll))
	attachments = append(attachments, recipientsField(poll))

	if poll.Kind == "response" {
		attachments = append(attachments, possibleAnswerField(poll))
	}

	return Attachment{
		Title:   poll.UUID,
		Pretext: "Look good to you (yes/no)?",
		Text:    poll.Question,
		Fields:  attachments,
	}
}

func (poll *Poll) SlackRecipientAttachment() Attachment {
	attachments := []AttachmentField{}

	attachments = append(attachments, pollTypeField(poll))

	if poll.Kind == "response" {
		attachments = append(attachments, possibleAnswerField(poll))
	}

	return Attachment{
		Title:   "Question",
		Pretext: fmt.Sprintf("We have a question for you. You can answer via `answer poll %s {insert response}`", poll.UUID),
		Text:    poll.Question,
		Fields:  attachments,
	}
}
