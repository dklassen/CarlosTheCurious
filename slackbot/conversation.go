package slackbot

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

var (
	slackIDRegex = regexp.MustCompile("<(?:@|#)([a-zA-Z0-9]+)>")
	stageLookup  = map[string]Stage{
		"initial":       getQuestion,
		"getAnswers":    getAnswers,
		"getRecipients": getRecipients,
		"sendPoll":      sendPoll,
	}

	registeredCommands = map[string]HandlerFunc{
		"^[sS]how poll (.*)$":                     showPoll,
		"^[cC]reate ([a-zA-Z]+) poll$":            createPoll,
		"^[cC]ancel poll ([a-zA-Z-0-9-_]+)$":      cancelPoll,
		"^[aA]nswer poll ([a-zA-Z-0-9-_]+) (.*$)": answerPoll,
		"^[lL]ist active polls$":                  activePolls,
		"^[hH]elp":                                usage,
	}
)

func parseRecpientsText(robot *Robot, msg Message) []Recipient {
	recipients := make(map[string]Recipient)

	for _, match := range slackIDRegex.FindAllStringSubmatch(msg.Text, -1) {
		logrus.Info(match)
		id := SlackID{Value: match[1]}

		switch id.Kind() {
		case PublicChannelID:
			channel, ok := robot.Channels[id.Value]
			if !ok {
				logrus.Error("Unable to find channel with id: ", id.Value)
			} else {
				for _, r := range channel.Members {
					recipient := Recipient{SlackID: r}
					recipients[recipient.SlackID] = recipient
				}
			}
		case UserID:
			recipient := Recipient{SlackID: id.Value}
			recipients[recipient.SlackID] = recipient
		default:
			logrus.Error("Unable to identify slackID ", id.Value)
		}
	}

	// NOTE(dana) is there a nicer way to get the values from a map?
	recipientList := []Recipient{}
	for _, v := range recipients {
		recipientList = append(recipientList, v)
	}

	return recipientList
}

func usage(robot *Robot, msg *Message, captureGroups []string) error {
	usage := `*Description*

Carlos the Curious at your service! Create and gather feedback to simple survey questions. Follow the commands below to create your poll and send it to either all members of a channel or to specific individuals. Once the poll has been created it will be sent and the responses will be collected.

*Commands*

*'create {feedback|response} poll'* - begin the process of creating a poll. Carlos will
ask you follow up questions to build the survey don't worry you can cancel at
any time. If you choose _feedback_ the answer can be freeform, if _response_ the answers show be one of the supplied responses.

*'cancel poll {poll_uuid}'* - Cancel a currently active or inprogress poll.

*'answer poll {poll_uuid} {answer}'* - When Carlos sends you a direct message you can
answer the poll with the above command. Everything after the poll_name can be free
text.

*'show poll{poll_uuid}'* - Display the results for the mentioned poll

*'list active polls'* - List your active polls

*'help'* - Display the help but you already knew that
`
	return robot.SendMessage(msg.Channel, usage)
}

func activePolls(robot *Robot, msg *Message, captures []string) (err error) {
	polls := []*Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage = ?", msg.User, msg.Channel, "active").Find(&polls)

	var result bytes.Buffer
	for k, v := range polls {
		result.WriteString(fmt.Sprintf("%d. %s - id:%s\n", k+1, v.Question, v.UUID))
	}

	attachment := Attachment{
		Text: result.String(),
	}

	if len(polls) == 0 {
		robot.SendMessage(msg.Channel, "You have no active polls")
		return nil
	}

	robot.PostMessage(msg.Channel, "Here are the list of active polls:", attachment)
	return nil
}

func createPoll(robot *Robot, msg *Message, captureGroups []string) error {
	existing, _ := FindFirstInactivePollByMessage(msg)
	if existing.ID != 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("There is already a poll being created. Cancel the poll with: 'cancel poll %s'", existing.UUID))
		return ErrExistingInactivePoll
	}

	kind := captureGroups[1]
	if strings.Compare(kind, ResponsePoll) != 0 && strings.Compare(kind, FeedbackPoll) != 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Poll must be of type response or feedback cannot be %s", kind))
		return ErrInvalidPollType
	}

	poll := NewPoll(kind, msg.User, msg.Channel)
	if err := poll.Save(); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			robot.SendMessage(msg.Channel, "Sigh at the moment we need uniquely named polls. Sorry")
		} else {
			robot.SendMessage(msg.Channel, "Something has gone wrong. We are looking into it.")
		}
		return err
	}

	return robot.SendMessage(msg.Channel, fmt.Sprintf("Creating a %s poll. You can cancel the poll any time with `cancel poll %s`\nWhat was the question you wanted to ask?", kind, poll.UUID))
}

func answerPoll(robot *Robot, msg *Message, captureGroups []string) error {
	pollName := captureGroups[1]
	poll := &Poll{}
	if err := GetDB().Where("uuid = ? AND stage = ?", pollName, "active").First(poll).Error; err != nil || poll.ID == 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Sorry about this but didn't not find a poll with the name %s", pollName))
		return err
	}

	if err := poll.AddResponse(msg.User, captureGroups[2]); err != nil {
		logrus.Error(err)
		robot.SendMessage(msg.Channel, "We were unable to add your response")
		return nil
	}

	return robot.SendMessage(msg.Channel, "Thanks for responding!")
}

func showPoll(robot *Robot, msg *Message, captureGroups []string) error {
	uuid := captureGroups[1]
	poll := &Poll{}
	if err := GetDB().Where("uuid = ?", uuid).First(poll).Error; err != nil {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Sorry about this but didn't not find a poll %s", uuid))
		return err
	}

	if poll.ID == 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Did not find a poll with the name %s", uuid))
		return fmt.Errorf("No poll found with name : %s", uuid)
	}

	attachment := poll.SlackPollSummary()
	return robot.PostMessage(msg.Channel, "", attachment)
}

func cancelPoll(robot *Robot, msg *Message, captureGroups []string) error {
	uuid := strings.TrimSpace(captureGroups[1])
	poll := &Poll{}
	GetDB().Where("uuid = ? ", uuid).First(poll)
	if poll.ID == 0 {
		robot.SendMessage(msg.Channel, "Oops, couldn't find the poll for you")
		return fmt.Errorf("Unable to find poll with uuid %s", uuid)
	}

	if err := GetDB().Delete(poll).Error; err != nil {
		return err
	}

	return robot.SendMessage(msg.Channel, "Okay, cancelling the poll for you")
}

func getQuestion(robot *Robot, msg *Message, poll *Poll) error {
	nextStage := ""
	response := ""

	switch poll.Kind {
	case FeedbackPoll:
		nextStage = "getRecipients"
		response = "Who should we send this to?"
	case ResponsePoll:
		nextStage = "getAnswers"
		response = "What are the possible responses (comma separated)?"
	default:
		logrus.Panic("Unknown kind of poll %s", poll.Kind)
	}

	poll.Question = msg.Text
	if err := poll.TransitionTo(nextStage); err != nil {
		return err
	}
	return robot.SendMessage(msg.Channel, response)
}

func getAnswers(robot *Robot, msg *Message, poll *Poll) error {
	answerStrings := strings.Split(msg.Text, ",")
	answers := []PossibleAnswer{}
	for _, answer := range answerStrings {
		answer = strings.Trim(answer, " ")
		answers = append(answers, PossibleAnswer{Value: answer})
	}

	poll.PossibleAnswers = answers
	if err := poll.TransitionTo("getRecipients"); err != nil {
		return err
	}

	return robot.SendMessage(msg.Channel, "Who should we send this to?")
}

func getRecipients(robot *Robot, msg *Message, poll *Poll) error {
	recipients := parseRecpientsText(robot, *msg)

	if err := poll.SetRecipients(recipients); err != nil {
		robot.SendMessage(msg.Channel, "Had trouble setting the recipients. Make sure they are valid channel names and try again")
		return err
	}

	if err := poll.TransitionTo("sendPoll"); err != nil {
		robot.SendMessage(msg.Channel, "Error saving the poll. Try again to set the recpients")
		return err
	}
	return robot.PostMessage(msg.Channel, "Here's a preview of what we are going to send:", poll.SlackPreviewAttachment())
}

func sendPoll(robot *Robot, msg *Message, poll *Poll) error {
	if msg.Text != "yes" {
		return robot.SendMessage(msg.Channel, fmt.Sprintf("Okay not going to send poll. You can cancel with `cancel poll %s`", poll.UUID))
	}

	if err := poll.TransitionTo("active"); err != nil {
		return err
	}

	recipients, err := poll.GetRecipients()
	if err != nil {
		robot.SendMessage(msg.Channel, "hummmmm something seems to be wrong with getting the list of recipients")
		return err
	}

	for _, recipient := range recipients {
		robot.PostMessage(recipient.SlackID, "", poll.SlackRecipientAttachment())
	}

	return robot.SendMessage(msg.Channel, fmt.Sprintf("Poll is live you can check in by asking me to `show poll %s`", poll.UUID))
}
