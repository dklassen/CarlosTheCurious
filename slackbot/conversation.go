package slackbot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

var (
	userIDRegex = regexp.MustCompile("<@([a-zA-Z0-9]+)>")
	stageLookup = map[string]Stage{
		"getQuestion":   getQuestion,
		"getAnswers":    getAnswers,
		"getRecipients": getRecipients,
		"sendPoll":      sendPoll,
	}

	registeredCommands = map[string]HandlerFunc{
		"^show poll (.*)$":                     showPoll,
		"^create poll (.*)$":                   createPoll,
		"^cancel poll ([a-zA-Z-0-9-_]+)$":      cancelPoll,
		"^answer poll ([a-zA-Z-0-9-_]+) (.*$)": answerPoll,
		"^help": usage,
	}
)

func usage(robot *Robot, msg *Message, captureGroups []string) (err error) {
func parseRecpientsText(msg Message) []Recipient {
	recipients := []Recipient{}
	for _, match := range userIDRegex.FindAllStringSubmatch(msg.Text, -1) {
		recipient := Recipient{SlackID: match[1]}
		recipients = append(recipients, recipient)
	}
	return recipients
}
	usage := `*Description*

Carlos the Curious at your service! Create and gather feedback to simple survey questions. Follow the commands below to create your poll and send it to either all members of a channel or to specific individuals. Once the poll has been created it will be sent and the responses will be collected.

*Commands*

*'create poll {poll_name}'* - begin the process of creating a poll. Carlos will
ask you follow up questions to build the survey don't worry you can cancel at
any time.

*'cancel poll {poll_name}'* - Cancel a currently active or inprogress poll.

*'answer poll {poll_name} {answer}'* - When Carlos sends you a direct message you can
answer the poll with the above command. Everything after the poll_name can be free
text.

*'show results {poll_name}'* - Display the results for the mentioned poll

*'help'* - Display the help but you already knew that
`
	robot.SendMessage(msg.Channel, usage)
	return err
}

func answerPoll(robot *Robot, msg *Message, captureGroups []string) error {
	pollName := captureGroups[1]
	poll := &Poll{}
	if err := GetDB().Where("name = ? AND stage = ?", pollName, "active").First(poll).Error; err != nil || poll.ID == 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Sorry about this but didn't not find a poll with the name %s", pollName))
		return err
	}

	answer := captureGroups[2]
	GetDB().Model(poll).Association("Responses").Append(PollResponse{Value: answer, SlackID: msg.User})

	robot.SendMessage(msg.Channel, "Thanks for responding!")
	return nil
}

func createPoll(robot *Robot, msg *Message, captureGroups []string) error {
	existingPoll := Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage != ?", msg.User, msg.Channel, "active").First(&existingPoll)
	if existingPoll.ID != 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("There is already a poll being created. Cancel the poll with: 'cancel poll %s'", existingPoll.Name))
		return fmt.Errorf("Active poll already exists")
	}

	pollName := captureGroups[1]
	poll := &Poll{
		Name:            strings.TrimSpace(pollName),
		Creator:         msg.User,
		Channel:         msg.Channel,
		Stage:           "getQuestion",
		PossibleAnswers: []PossibleAnswer{},
	}

	if err := GetDB().Debug().Save(poll).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			robot.SendMessage(msg.Channel, "Sigh at the moment we need uniquely named polls. Sorry")
		} else {
			robot.SendMessage(msg.Channel, "Something has gone wrong. We are looking into it")
		}
		return err
	}

	robot.SendMessage(msg.Channel, fmt.Sprintf("Creating poll named %s", pollName))
	robot.SendMessage(msg.Channel, "What was the question you wanted to ask?")
	return nil
}

func showPoll(robot *Robot, msg *Message, captureGroups []string) error {
	pollName := captureGroups[1]
	poll := &Poll{}
	if err := GetDB().Where("name = ? ", pollName).First(poll).Error; err != nil {
		return err
	}

	if poll.ID == 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Did not find a poll with the name %s", pollName))
		return fmt.Errorf("No poll found with name : %s", pollName)
	}

	attachment := poll.SlackPollSummary()
	robot.PostMessage(msg.Channel, "Getting the results so far.....", attachment)
	return nil
}

func cancelPoll(robot *Robot, msg *Message, captureGroups []string) error {
	pollName := strings.TrimSpace(captureGroups[1])
	poll := &Poll{}
	GetDB().Where("name = ? ", pollName).First(poll)
	if poll.ID == 0 {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Did not find a poll with the name %s", pollName))
		return fmt.Errorf("Did not find a poll with the name %s", pollName)
	}

	GetDB().Delete(poll)
	robot.SendMessage(msg.Channel, fmt.Sprintf("Okay, cancelling the poll %s for you", poll.Name))
	return nil
}

func getQuestion(robot *Robot, msg *Message, poll *Poll) error {
	poll.Question = msg.Text
	poll.Stage = "getAnswers"
	if err := GetDB().Save(poll).Error; err != nil {
		return err
	}

	robot.SendMessage(msg.Channel, "What are the possible responses (comma separated)?")
	return nil
}

func getAnswers(robot *Robot, msg *Message, poll *Poll) error {
	answerStrings := strings.Split(msg.Text, ",")
	answers := []PossibleAnswer{}
	for _, answer := range answerStrings {
		answer = strings.Trim(answer, " ")
		answers = append(answers, PossibleAnswer{Value: answer})
	}

	poll.PossibleAnswers = answers
	poll.Stage = "getRecipients"
	GetDB().Save(poll)

	robot.SendMessage(msg.Channel, "Who should we send this to?")
	return nil
}

func getRecipients(robot *Robot, msg *Message, poll *Poll) error {
	poll.Stage = "sendPoll"
	poll.Recipients = parseRecpientsText(*msg)
	if err := GetDB().Save(&poll).Error; err != nil {
		return err
	}

	robot.PostMessage(msg.Channel, "Here's a preview of what we are going to send:", poll.SlackPreviewAttachment())
	return nil
}

func sendPoll(robot *Robot, msg *Message, poll *Poll) error {
	if msg.Text != "yes" {
		robot.SendMessage(msg.Channel, fmt.Sprintf("Okay not going to send poll. You can cancel with `cancel poll %s`", poll.Name))
		return nil
	}

	poll.Stage = "active"
	GetDB().Save(poll)

	recipients := []Recipient{}
	GetDB().Model(&poll).Related(&recipients)
	for _, recipient := range recipients {
		robot.PostMessage(recipient.SlackID, "", poll.SlackRecipientAttachment())
	}

	robot.SendMessage(msg.Channel, fmt.Sprintf("Poll is live you can check in by asking me to `check poll %s`", poll.Name))
	return nil
}
