package slackbot

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

// MockHttpClient so we can capture requests and check we called what
// we think we called
type MockHTTPClient struct {
	Requests []http.Request
}

func (client *MockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	client.Requests = append(client.Requests, *req)
	response := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte("{\"ok\": true}"))),
	}

	return response, nil
}

func init() {
	os.Setenv("GOENV", "test")
	debug := false
	debugEnv := os.Getenv("DEBUG")
	if debugEnv == "true" {
		debug = true
	}

	databaseURL := "postgres://postgres:@127.0.0.1/carlos_test?sslmode=disable"
	conf := &Config{
		DatabaseURL: databaseURL,
		Debug:       debug,
	}
	SetupDatabase(conf.DatabaseURL, conf.Debug)
}

var testMatch = func(handlers map[string]Handler, msg *Message) (cmd *Handler, capture []string) {
	msg.Handled = true
	return
}

var testHandler = &MessageHandler{
	Handlers: make(map[string]Handler),
	Matcher:  testMatch,
}

func SetupTestDatabase() {
	DropDatabaseTables()
	Migrate()
}

func CleanSetup() Robot {
	SetupTestDatabase()
	robot := Robot{
		ID:         "1",
		Origin:     "",
		APIToken:   "",
		Handler:    defaultMessageHandler,
		Client:     &MockHTTPClient{},
		ListenChan: make(chan Message),
	}
	robot.RegisterCommands(registeredCommands)

	return robot
}

func testDispatch(t *testing.T) {
	robot := CleanSetup()
	robot.Handler = testHandler

	var testMessages = []struct {
		Text            string
		Channel         string
		DirectMention   bool
		expectedHandled bool
		Reason          string
	}{
		{
			Text:            "the fox and the fields of brown",
			DirectMention:   false,
			Channel:         "CDFSDSFS",
			expectedHandled: false,
			Reason:          "Is not handled because not direct mention",
		},
		{
			Text:            "the fox and the fields of brown",
			DirectMention:   true,
			Channel:         "CDFSDSFS",
			expectedHandled: true,
			Reason:          "Is handled because is direct mention",
		},
		{
			Text:            "the fox and the fields of brown",
			DirectMention:   false,
			Channel:         "Dprivatemessage",
			expectedHandled: true,
			Reason:          "Is handled because is private message",
		},
	}

	for _, testCase := range testMessages {
		testMsg := &Message{
			Text:          testCase.Text,
			Channel:       testCase.Channel,
			DirectMention: testCase.DirectMention,
		}

		robot.ProcessMessage(testMsg)
		if testCase.expectedHandled != testMsg.Handled {
			t.Errorf("Expected: '%t' ,but got: '%t'", testCase.expectedHandled, testMsg.Handled)
		}
	}
}

func TestCommandProcessMessageStripsDirectMentionAndModifiesMessage(t *testing.T) {
	robot := CleanSetup()
	robot.Handler = testHandler

	var testMessages = []struct {
		Text            string
		Type            string
		expectedText    string
		expectedMention bool
	}{
		{
			Type:            "message",
			Text:            "the fox and the fields of brown",
			expectedText:    "the fox and the fields of brown",
			expectedMention: false},
		{
			Type:            "message",
			Text:            "<@1>: the fox and the fields of brown",
			expectedText:    "the fox and the fields of brown",
			expectedMention: true},
	}

	for _, testCase := range testMessages {
		testMsg := &Message{
			Text: testCase.Text,
			Type: testCase.Type,
		}

		robot.ProcessMessage(testMsg)
		if strings.Compare(testMsg.Text, testCase.expectedText) != 0 {
			t.Errorf("Expected: '%s' ,but got: '%s'", testCase.expectedText, testMsg.Text)
		}
		if testMsg.DirectMention != testCase.expectedMention {
			t.Errorf("Expected: '%t' ,but got: '%t'", testCase.expectedMention, testMsg.DirectMention)
		}
	}
}
