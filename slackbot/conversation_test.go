package slackbot

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/websocket"
)

func TestUserIDRegex(t *testing.T) {
	var testTable = []struct {
		TestMsg    string
		ExpectedID string
	}{
		{
			TestMsg:    "<@U12341>",
			ExpectedID: "U12341",
		},
		{
			TestMsg:    "<@U12341",
			ExpectedID: "",
		},
		{
			TestMsg:    "<U12341>",
			ExpectedID: "",
		},
	}
	for _, testCase := range testTable {
		output := userIDRegex.FindStringSubmatch(testCase.TestMsg)

		// NOTE: Derp no matches think of something
		if len(output) < 1 {
			output = []string{"", ""}
		}

		result := output[1]
		if strings.Compare(result, testCase.ExpectedID) != 0 {
			t.Error("Expected ID: ", testCase.ExpectedID, "got: ", result)
		}
	}
}

func TestGetQuestion(t *testing.T) {
	robot := CleanSetup()

	outgoing := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	poll := Poll{Creator: "1", Channel: "a"}
	msg := Message{Text: "Incoming question"}
	getQuestion(&robot, &msg, &poll)

	if poll.Stage != "getAnswers" {
		t.Error("Expected stage to be: getAnswers got:", poll.Stage)
	}
	expectedResponse := []byte("What are the possible responses (comma separated)?")
	if bytes.Compare(outgoing, expectedResponse) != 0 {
		t.Error("Expected response to be: ", string(expectedResponse), " got: ", string(outgoing))
	}
}

func TestGetRecipients(t *testing.T) {
	robot := CleanSetup()

	outgoing := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	var testTable = []struct {
		TestPoll               Poll
		RecipientsMsg          string
		ExpectedResponsesCount int
	}{
		{
			TestPoll:               Poll{Name: "test1"},
			RecipientsMsg:          "<@U1231231>",
			ExpectedResponsesCount: 1,
		},
		{
			TestPoll:               Poll{Name: "test2"},
			RecipientsMsg:          "<@U1231231>, <@U1231256>",
			ExpectedResponsesCount: 2,
		},
		{
			TestPoll:               Poll{Name: "test3"},
			RecipientsMsg:          "here are the recipients: <@U1231231>, <@U1231256>",
			ExpectedResponsesCount: 2,
		},
	}

	for _, testEntry := range testTable {
		poll := testEntry.TestPoll
		msg := Message{Text: testEntry.RecipientsMsg}
		getRecipients(&robot, &msg, &poll)

		result := []Recipient{}
		GetDB().Model(&poll).Related(&result)
		if len(result) != testEntry.ExpectedResponsesCount {
			t.Error("Expected: ", testEntry.ExpectedResponsesCount, " recipients got: ", len(result))
		}
	}
}

func TestSendPoll(t *testing.T) {
	robot := CleanSetup()

	outgoing := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	var testTable = []struct {
		InputPoll           Poll
		InputMessage        Message
		ExpectedMessage     []byte
		ExpectedPostMessage string
		ExpectedStage       string
		ExpectedRequests    int
	}{
		// Test we don't send the poll unless we send yes
		{
			InputPoll:           Poll{Name: "test1", Creator: "derp", Channel: "dorp"},
			InputMessage:        Message{Text: "no", User: "derp", Channel: "dorp"},
			ExpectedMessage:     []byte("Okay not going to send poll. You can cancel with `cancel poll test1`"),
			ExpectedPostMessage: "",
			ExpectedStage:       "",
			ExpectedRequests:    0,
		},
		// Test when reply with yes we transition poll to active and send message
		// no recipients so no requests
		{
			InputPoll:           Poll{Name: "test2", Creator: "derp", Channel: "dorp"},
			InputMessage:        Message{Text: "yes", User: "derp", Channel: "dorp"},
			ExpectedMessage:     []byte("Poll is live you can check in by asking me to `check poll test2`"),
			ExpectedPostMessage: "",
			ExpectedStage:       "active",
			ExpectedRequests:    0,
		},
		{
			// Test poll has a single recipient and should progress to the active state and try posting one message to
			// the recipient
			InputPoll:           Poll{Name: "test3", Creator: "derp", Channel: "dorp", Recipients: []Recipient{Recipient{SlackID: "Ben", SlackName: "Oro"}}},
			InputMessage:        Message{Text: "yes", User: "derp", Channel: "dorp"},
			ExpectedMessage:     []byte("Poll is live you can check in by asking me to `check poll test3`"),
			ExpectedPostMessage: "",
			ExpectedStage:       "active",
			ExpectedRequests:    1,
		},
	}

	for _, testCase := range testTable {
		outgoing = []byte("")
		sendPoll(&robot, &testCase.InputMessage, &testCase.InputPoll)

		resultPoll := &Poll{}
		GetDB().Where("creator = ? AND channel = ?", testCase.InputMessage.User, testCase.InputMessage.Channel).First(&resultPoll)

		if bytes.Compare(outgoing, testCase.ExpectedMessage) != 0 {
			t.Error("Expected response message: ", string(testCase.ExpectedMessage), " got: ", string(outgoing))
		}

		if resultPoll.Stage != testCase.ExpectedStage {
			t.Error("Expected poll to be in stage: ", testCase.ExpectedStage, " got: ", resultPoll.Stage)
		}

		client := robot.Client.(*MockHTTPClient)
		if testCase.ExpectedRequests != len(client.Requests) {
			t.Error("Expected requests: ", testCase.ExpectedRequests, " got: ", len(client.Requests))
		}
	}
}

func TestAnswerPollSavesResponse(t *testing.T) {
	robot := CleanSetup()

	outgoing := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	var testCases = []struct {
		InputPoll            Poll
		InputMsg             Message
		InputCaptures        []string
		ExpectedSuccess      bool
		ExpectedResponse     PollResponse
		ExpectedRobotMessage []byte
	}{
		// Perfect case where the responder is able to save the response to the recipient
		{
			InputPoll:            Poll{Name: "test-1", Channel: "dorp", Creator: "Merv", Stage: "active", Recipients: []Recipient{{SlackID: "1234"}}},
			InputMsg:             Message{User: "1234", Channel: "Private_Channel"},
			InputCaptures:        []string{"everything", "test-1", "Why do we not get ice cream on thursdays?"},
			ExpectedSuccess:      true,
			ExpectedResponse:     PollResponse{SlackID: "1234", Value: "Why do we not get ice cream on thursdays?"},
			ExpectedRobotMessage: []byte("Thanks for responding!"),
		},
		// Poll is not in the active stage so not ready to answer and will fail
		{
			InputPoll:            Poll{Name: "test-2", Channel: "dorp", Creator: "Merv", Stage: "sendPoll", Recipients: []Recipient{{SlackID: "1234"}}},
			InputMsg:             Message{User: "1234", Channel: "Private_Channel"},
			InputCaptures:        []string{"everything", "test-2", "Why do we not get ice cream on thursdays?"},
			ExpectedSuccess:      false,
			ExpectedResponse:     PollResponse{SlackID: "1234", Value: "Why do we not get ice cream on thursdays?"},
			ExpectedRobotMessage: []byte("Sorry about this but didn't not find a poll with the name test-2"),
		},
	}

	for _, testCase := range testCases {
		outgoing = []byte("")
		GetDB().Save(&testCase.InputPoll)

		answerPoll(&robot, &testCase.InputMsg, testCase.InputCaptures)
		savedPoll, _ := FindFirstActivePollByName(testCase.InputPoll.Name)
		resultResponse := &PollResponse{}
		GetDB().Where("poll_id = ? AND slack_id =?", savedPoll.ID, testCase.InputMsg.User).First(resultResponse)

		ExpectedResponse := testCase.ExpectedResponse
		if testCase.ExpectedSuccess && strings.Compare(ExpectedResponse.Value, resultResponse.Value) != 0 {
			t.Error("Expected recipient response: ", ExpectedResponse.Value, " but got: ", resultResponse.Value)
		}

		if bytes.Compare(outgoing, testCase.ExpectedRobotMessage) != 0 {
			t.Error("Expected the robot to say: ", string(testCase.ExpectedRobotMessage), " but got: ", string(outgoing))
		}
	}
}
