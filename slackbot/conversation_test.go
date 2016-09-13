package slackbot

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/websocket"
)

type BySlackID []Recipient

func (a BySlackID) Len() int           { return len(a) }
func (a BySlackID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySlackID) Less(i, j int) bool { return a[i].SlackID < a[j].SlackID }

func TestParseRecipientsText(t *testing.T) {
	robot := CleanSetup()

	var tests = []struct {
		Input         Message
		InputChannels map[string]Channel
		Expected      []Recipient
	}{
		{
			// identify a single user id
			Input:    Message{Text: "<@UDF123>"},
			Expected: []Recipient{{SlackID: "UDF123"}},
		},
		{
			// Identify two user ids
			Input:    Message{Text: "<@UDF123> and <@USDF>"},
			Expected: []Recipient{{SlackID: "UDF123"}, {SlackID: "USDF"}},
		},
		{
			// Identify a single user and a channel with multiple users
			Input: Message{Text: "<@UDF123> and <#C1U41SHTK|general>"},
			InputChannels: map[string]Channel{
				"C1U41SHTK": Channel{Members: []string{"derp1", "derp2"}},
			},
			Expected: []Recipient{
				{SlackID: "UDF123"},
				{SlackID: "derp1"},
				{SlackID: "derp2"},
			},
		},
	}

	for _, test := range tests {
		robot.Channels = test.InputChannels
		result := parseRecpientsText(&robot, test.Input)
		sort.Sort(BySlackID(result))
		sort.Sort(BySlackID(test.Expected))

		if len(result) != len(test.Expected) {
			t.Fatal("Expected:", len(test.Expected), "recipients got:", len(result))
		}

		for k, v := range test.Expected {
			if strings.Compare(v.SlackID, result[k].SlackID) != 0 {
				t.Fatal("Failed to parse recipient Id. Expected:", v.SlackID, "got:", result[k].SlackID)
			}
		}
	}
}

func TestSlackIDRegex(t *testing.T) {
	var testTable = []struct {
		TestMsg  string
		Expected []string
	}{
		{
			TestMsg:  "<@U12341>",
			Expected: []string{"<@U12341>", "U12341"},
		},
		{
			TestMsg:  "<@U12341",
			Expected: []string{},
		},
		{
			TestMsg:  "<U12341>",
			Expected: []string{},
		},
	}

	for _, testCase := range testTable {
		result := slackIDRegex.FindStringSubmatch(testCase.TestMsg)

		if len(result) != len(testCase.Expected) {
			t.Fatal("Did not match all ids in input", result)
		}

		for k, x := range result {
			if strings.Compare(x, testCase.Expected[k]) != 0 {
				t.Fatal("Expected:", x, "got:", testCase.Expected[k])
			}
		}
	}
}

func TestCreatePoll(t *testing.T) {
	outgoing := []byte{}

	GenerateUUID = func() string {
		return "amazing"
	}

	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	var testTable = []struct {
		InputMessage    Message
		InputCaptures   []string
		ExpectedError   bool
		ExpectedPoll    Poll
		ExpectedMessage []byte
	}{
		// Generate response poll
		{
			InputMessage:    Message{Text: "bananas", User: "Balony", Channel: "coffee"},
			InputCaptures:   []string{"", "response"},
			ExpectedError:   false,
			ExpectedMessage: []byte("Creating a response poll. You can cancel the poll any time with `cancel poll amazing`\nWhat was the question you wanted to ask?"),
			ExpectedPoll:    Poll{Stage: "initial", Kind: ResponsePoll},
		},
		// Generate feedback poll
		{
			InputMessage:    Message{Text: "bananas", User: "Balony2", Channel: "coffee3"},
			InputCaptures:   []string{"", "feedback"},
			ExpectedError:   false,
			ExpectedMessage: []byte("Creating a feedback poll. You can cancel the poll any time with `cancel poll amazing`\nWhat was the question you wanted to ask?"),
			ExpectedPoll:    Poll{Stage: "initial", Kind: FeedbackPoll},
		},
		// Unable to generate poll
		{
			InputMessage:    Message{Text: "bananas", User: "Balony2", Channel: "coffee3"},
			InputCaptures:   []string{"", "not a known type"},
			ExpectedError:   true,
			ExpectedMessage: []byte("Poll must be of type response or feedback cannot be not a known type"),
			ExpectedPoll:    Poll{},
		},
	}

	for _, test := range testTable {
		robot := CleanSetup()
		outgoing = []byte("")

		err := createPoll(&robot, &test.InputMessage, test.InputCaptures)
		if err != nil && test.ExpectedError != true {
			t.Fatal(err)
		}

		poll, err := FindFirstInactivePollByMessage(&test.InputMessage)
		if err != nil && test.ExpectedError != true {
			t.Fatal("Unable to find poll which was expected to be there")
		}

		expectedPoll := test.ExpectedPoll
		if poll.Stage != expectedPoll.Stage {
			t.Fatal("Expected stage to be:", expectedPoll.Stage, "but got:", poll.Stage)
		}

		if strings.Compare(poll.Kind, expectedPoll.Kind) != 0 {
			t.Fatal("Expected poll to be of kind:", expectedPoll.Kind, "but got:", poll.Kind)
		}

		if bytes.Compare(outgoing, test.ExpectedMessage) != 0 {
			t.Fatal("Expected response message: ", string(test.ExpectedMessage), " got: ", string(outgoing))
		}

	}
}

func TestCreatePollReturnsErrorWhenExistingPollIsBeingCreated(t *testing.T) {
	robot := CleanSetup()
	outgoing := []byte{}

	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	testMsg := Message{Text: "bananas", User: "Balony2", Channel: "coffee3"}
	testCaptures := []string{"", "feedback"}

	err := createPoll(&robot, &testMsg, testCaptures)
	if err != nil {
		t.Fatal("Was not expecting error to be thrown")
	}

	err = createPoll(&robot, &testMsg, testCaptures)
	if err != ErrExistingInactivePoll {
		t.Fatal("Was expecting a poll to already exist")
	}
}

func TestGetQuestion(t *testing.T) {
	robot := CleanSetup()

	responseMessage := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		responseMessage = append(responseMessage, msg.Text...)
		return nil
	}

	var testTable = []struct {
		InputMessage     Message
		InputPoll        Poll
		ExpectedPoll     Poll
		ExpectedResponse []byte
	}{
		// Response poll should save the question and transition to getAnswers
		{
			InputMessage:     Message{Text: "aww yiss", User: "1", Channel: "a"},
			InputPoll:        Poll{Kind: "response", Creator: "1", Channel: "a", UUID: "a"},
			ExpectedPoll:     Poll{Kind: "response", Creator: "1", Channel: "a", Stage: "getAnswers"},
			ExpectedResponse: []byte("What are the possible responses (comma separated)?"),
		},
		//	Feedback poll should save the question and transition to getRecipients
		{
			InputMessage:     Message{Text: "aww yiss", User: "1", Channel: "b"},
			InputPoll:        Poll{Kind: FeedbackPoll, Creator: "1", Channel: "b", UUID: "b"},
			ExpectedPoll:     Poll{Kind: FeedbackPoll, Creator: "1", Channel: "b", Stage: "getRecipients"},
			ExpectedResponse: []byte("Who should we send this to?"),
		},
	}

	msg := Message{Text: "Incoming question"}

	for _, test := range testTable {
		responseMessage = []byte("")

		getQuestion(&robot, &msg, &test.InputPoll)

		exp := test.ExpectedPoll
		output, err := FindFirstInactivePollByMessage(&test.InputMessage)
		if err != nil {
			t.Fatal("Was not expecting error", err)
		}

		if output.Stage != exp.Stage {
			t.Fatal("Expected stage to be:", exp.Stage, "got:", output.Stage)
		}
		if bytes.Compare(responseMessage, test.ExpectedResponse) != 0 {
			t.Fatal("Expected response to be: ", string(test.ExpectedResponse), " got: ", string(responseMessage))
		}
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
			TestPoll:               Poll{Kind: "response", UUID: "1"},
			RecipientsMsg:          "<@U1231231>",
			ExpectedResponsesCount: 1,
		},
		{
			TestPoll:               Poll{Kind: "response", UUID: "2"},
			RecipientsMsg:          "<@U1231231>, <@U1231256>",
			ExpectedResponsesCount: 2,
		},
		{
			TestPoll:               Poll{Kind: "response", UUID: "3"},
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
			InputPoll:           Poll{Kind: "response", UUID: "1", Creator: "derp", Channel: "dorp"},
			InputMessage:        Message{Text: "no", User: "derp", Channel: "dorp"},
			ExpectedMessage:     []byte("Okay not going to send poll. You can cancel with `cancel poll 1`"),
			ExpectedPostMessage: "",
			ExpectedStage:       "",
			ExpectedRequests:    0,
		},
		// Test when reply with yes we transition poll to active and send message
		// no recipients so no requests
		{
			InputPoll:           Poll{Kind: "response", UUID: "2", Creator: "derp", Channel: "dorp"},
			InputMessage:        Message{Text: "yes", User: "derp", Channel: "dorp"},
			ExpectedMessage:     []byte("Poll is live you can check in by asking me to `show poll 2`"),
			ExpectedPostMessage: "",
			ExpectedStage:       "active",
			ExpectedRequests:    0,
		},
		{
			// Test poll has a single recipient and should progress to the active state and try posting one message to
			// the recipient
			InputPoll:           Poll{Kind: "response", UUID: "3", Creator: "derp", Channel: "dorp", Recipients: []Recipient{Recipient{SlackID: "Ben", SlackName: "Oro"}}},
			InputMessage:        Message{Text: "yes", User: "derp", Channel: "dorp"},
			ExpectedMessage:     []byte("Poll is live you can check in by asking me to `show poll 3`"),
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

func TestCancelPoll(t *testing.T) {
	SetupTestDatabase()

	outgoing := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	var testTable = []struct {
		TargetPoll       Poll
		TargetMessage    Message
		ExpectedResponse []byte
		ExpectDeleted    bool
	}{
		{
			TargetPoll:       Poll{Kind: "response", UUID: "1"},
			ExpectedResponse: []byte("Okay, cancelling the poll for you"),
			ExpectDeleted:    true,
			TargetMessage:    Message{User: "blarg", Channel: "Wootzone", Text: "cancel poll 1", DirectMention: true},
		},
		{
			TargetPoll:       Poll{Kind: "response", UUID: "not_going_to_find"},
			ExpectedResponse: []byte("Oops, couldn't find the poll for you"),
			ExpectDeleted:    false,
			TargetMessage:    Message{User: "blarg", Channel: "Wootzone", Text: "cancel poll 2", DirectMention: true},
		},
	}

	for _, testEntry := range testTable {
		robot := CleanSetup()
		GetDB().Save(&testEntry.TargetPoll)

		outgoing = []byte("")
		robot.Dispatch(&testEntry.TargetMessage)

		if bytes.Compare(outgoing, testEntry.ExpectedResponse) != 0 {
			t.Error("Got unexpected robot response: '", string(outgoing), "' expected: '", string(testEntry.ExpectedResponse), "'")
		}

		if testEntry.ExpectDeleted == true {
			poll, _ := FindFirstPreActivePollByName(testEntry.TargetPoll.UUID)
			if poll.DeletedAt != nil {
				t.Error("Expected poll to be deleted")
			}
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
			InputPoll:            Poll{Kind: "response", UUID: "1", Channel: "dorp", Creator: "Merv", Stage: "active", Recipients: []Recipient{{SlackID: "1234"}}},
			InputMsg:             Message{User: "1234", Channel: "Private_Channel"},
			InputCaptures:        []string{"everything", "1", "Why do we not get ice cream on thursdays?"},
			ExpectedSuccess:      true,
			ExpectedResponse:     PollResponse{SlackID: "1234", Value: "Why do we not get ice cream on thursdays?"},
			ExpectedRobotMessage: []byte("Thanks for responding!"),
		},
		// Poll is not in the active stage so not ready to answer and will fail
		{
			InputPoll:            Poll{Kind: "response", UUID: "2", Channel: "dorp", Creator: "Merv", Stage: "sendPoll", Recipients: []Recipient{{SlackID: "1234"}}},
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
		savedPoll, _ := FindFirstActivePollByUUID(testCase.InputPoll.UUID)
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

func TestConversationFlowForResponsePoll(t *testing.T) {
	robot := CleanSetup()
	testMessage := Message{
		User:          "bloop",
		Channel:       "blarg",
		Text:          "",
		DirectMention: true,
	}

	outgoing := []byte{}
	sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
		outgoing = append(outgoing, msg.Text...)
		return nil
	}

	GenerateUUID = func() string {
		return "blah"
	}

	var testMessages = []struct {
		ExpectedStage  string
		ExpectedText   []byte
		NextMsg        string
		UsePostMessage bool
	}{
		{
			// Stage 1: Initial message that is sent by the user
			ExpectedStage: "",
			ExpectedText:  []byte(""),
			NextMsg:       "create response poll",
		},
		{
			ExpectedStage: "initial",
			ExpectedText:  []byte("Creating a response poll. You can cancel the poll any time with `cancel poll blah`\nWhat was the question you wanted to ask?"),
			NextMsg:       "Here is your question",
		},
		{
			ExpectedStage: "getAnswers",
			ExpectedText:  []byte("What are the possible responses (comma separated)?"),
			NextMsg:       "Here are my answers",
		},
		{
			ExpectedStage: "getRecipients",
			ExpectedText:  []byte("Who should we send this to?"),
			NextMsg:       "<@U123>,<@U12415>",
		},
		{
			ExpectedStage:  "sendPoll",
			ExpectedText:   []byte(""),
			NextMsg:        "send it!",
			UsePostMessage: true,
		},
	}

	for _, testStage := range testMessages {
		poll, _ := FindFirstInactivePollByMessage(&testMessage)
		if poll.Stage != testStage.ExpectedStage {
			t.Fatal("Expected stage:", testStage.ExpectedStage, "got:", poll.Stage)
		}

		if bytes.Compare(outgoing, testStage.ExpectedText) != 0 {
			t.Fatal("Expected output messages: ", string(testStage.ExpectedText), "got: ", string(outgoing))
		}

		outgoing = []byte("")
		testMessage.Text = testStage.NextMsg
		robot.Dispatch(&testMessage)
	}
}
