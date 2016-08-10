package slackbot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
)

// Poll is the object we build up with questions we ask the user in Slack
type Poll struct {
	gorm.Model
	UUID            string `gorm:"not null;unique"`
	Channel         string `gorm:"not null"`
	Creator         string `gorm:"not null"`
	Stage           string
	Question        string
	Recipients      []Recipient
	Responses       []PollResponse
	PossibleAnswers []PossibleAnswer
}

// PossibleAnswer holds the value of a possible answer
type PossibleAnswer struct {
	gorm.Model
	PollID uint
	Value  string
}

// PollResponse holds the answer to the question
type PollResponse struct {
	gorm.Model
	PollID  uint
	SlackID string
	Value   string
}

// Recipient is the target of a particular poll
type Recipient struct {
	gorm.Model
	SlackID   string
	PollID    uint
	SlackName string
}

func NewPoll(name, creator, channel string) *Poll {
	uuid := GenerateUUID()
	return &Poll{
		UUID:            uuid,
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
func FindFirstActivePollByName(name string) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("name = ? AND stage = ?", name, "active").First(poll)

	if poll.ID == 0 {
		return poll, fmt.Errorf("No active poll with %s found", name)
	}
	return poll, nil
}

// FindFirstActivePollByMessage finds the first active poll using the user and channel
// NOTE:: OMG this is bad lets get rid of these
func FindFirstActivePollByMessage(msg Message) (*Poll, error) {
	poll := &Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage = ?", msg.User, msg.Channel, "active").First(&poll)

	if poll.ID != 0 {
		err := errors.New("No active poll found")
		return nil, err
	}
	return poll, nil
}

// FindFirstActivePollByMessage finds the first active poll using the user and channel
// NOTE:: OMG this is bad lets get rid of these
func FindFirstInactivePollByMessage(msg *Message) *Poll {
	poll := &Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage != ?", msg.User, msg.Channel, "active").First(&poll)

	if poll.ID != 0 {
		return nil
	}
	return poll
}

// FindRecipientByID finds the recipient based on the poll and slack user id
func FindRecipientByID(pollID uint, slackID string) Recipient {
	recipient := Recipient{}
	GetDB().Where("poll_id = ? AND slack_id = ?", pollID, slackID).First(&recipient)
	return recipient
}

// SlackIDString format the user id to a form that will be parsed by Slack to display the users name
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

// SlackPollSummary builds an attachment for the PostMessage api that shows a summary
// of the current statistics for a running or completed poll.
func (poll *Poll) SlackPollSummary() Attachment {
	return Attachment{
		Pretext: "Here are the results so far:",
		Text:    poll.Question,
		Fields: []AttachmentField{
			recipientsField(poll),
			possibleAnswerField(poll),
			responseSummaryField(poll),
		},
	}
}

// SlackPreviewAttachment generates an object which is translated to json
// and sent as part of the attachments field in the PostMessage api call
func (poll *Poll) SlackPreviewAttachment() Attachment {
	return Attachment{
		Pretext: "Look good to you (yes/no)?",
		Text:    poll.Question,
		Fields: []AttachmentField{
			recipientsField(poll),
			possibleAnswerField(poll),
		},
	}
}

// SlackRecipientAttachment is the attachment that gets sent to the recipient
func (poll *Poll) SlackRecipientAttachment() Attachment {
	return Attachment{
		Title:   "Question",
		Pretext: fmt.Sprintf("We have a question for you. You can answer via `answer poll %s {insert response}`", poll.UUID),
		Text:    poll.Question,
		Fields: []AttachmentField{
			AttachmentField{
				Title: "Possible Answers:",
				Value: poll.slackAnswerString(),
				Short: false,
			},
		},
	}
}
