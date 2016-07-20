package slackbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/websocket"
)

var (
	counter uint64 //All message sent during a session need to have an monotonically increasing id
)

type responseRTMSelf struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ChannelList is a list of channels from slack
type ChannelList struct {
	Ok       bool      `json:"ok"`
	Channels []Channel `json:"channels"`
	Error    string    `json:"error,omitempty"`
}

// Channel is a Slack channel description
type Channel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	IsChannel string   `json:"is_channel"`
	Members   []string `json:"members"`
}

//UserList  slack returned json object
type UserList struct {
	Ok      bool   `json:"ok"`
	Members []User `json:"members"`
	Error   string `json:"error,omitempty"`
}

// User is a Slack user entry
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ResponseRTMStart is the response object we get back from slack on the first
// initialization call to the api
type ResponseRTMStart struct {
	Ok    bool             `json:"ok"`
	Error string           `json:"error"`
	URL   string           `json:"url"`
	Self  *responseRTMSelf `json:"self"`
}

// HandlerFunc is a registered function that when matched is passed the message
// to be processed .
type HandlerFunc func(*Robot, *Message, []string) error

// Stage is a stage in the polling conversation
type Stage func(*Robot, *Message, *Poll) error

// MessageHandler is a Slack msssage multiplexer.
// It matches the message of incoming messages to registered
// commands based on first match to a regex
type MessageHandler struct {
	Handlers map[string]Handler
	Matcher  func(handlers map[string]Handler, msg *Message) (cmd *Handler, result []string)
}

// Handler holds the pattern and associated commands to call for a given mesage
// if a Handler has a defined next stage it becomes part of a dialog
type Handler struct {
	pattern     *regexp.Regexp
	handlerFunc HandlerFunc
}

// WebClienter interface will allow us to mock out http.Client
// during testing. This should be http.Client
type WebClienter interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// SlackWebClient struct holds the client for sending http requests
type SlackWebClient struct {
	HTTPClient *http.Client
}

// Do does what it is supposed to do
func (client *SlackWebClient) Do(req *http.Request) (resp *http.Response, err error) {
	return client.HTTPClient.Do(req)
}

// Robot is the primary abstraction we deal with for handling messge sending/recieving
// with the Slack API
type Robot struct {
	ID         string
	Name       string
	Origin     string
	APIToken   string
	Users      []User
	Client     WebClienter // http.Client
	Handler    *MessageHandler
	SendChan   chan Message
	Shutdown   chan int
	Channels   []Channel
	Connection *websocket.Conn
	ListenChan chan Message
}

// Message holds json object we transmit/recieve via Slack api
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

// Attachment holds information for the slack PostMessage attachement field
type Attachment struct {
	Title   string            `json:"title"`
	Text    string            `json:"text"`
	Pretext string            `json:"pretext"`
	Fields  []AttachmentField `json:"fields"`
}

// AttachmentField is a subfield of the slack PostMessage attachement field
type AttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// PostResponse is a response message from the SlackApi PostMessage upon posting a message
type PostResponse struct {
	Ok      bool   `json:"ok"`
	TS      string `json:"ts"`
	Channel string `json:"channel"`
	Error   string `json:"error"`
}

func GenerateUUID() string {
	output, err := exec.Command("uuidgen").Output()
	if err != nil {
		logrus.Fatal("Unable to generate a UUID using uuidgen check your system is compatible")
	}
	return strings.TrimSpace(string(output))
}

func downloadUserList(token string) (UserList, error) {
	resp, err := http.Get(fmt.Sprintf("https://slack.com/api/users.list?token=%s", token))
	if err != nil {
		logrus.Error(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
	}

	var userList UserList
	err = json.Unmarshal(body, &userList)

	return userList, err
}

func downloadChannelList(token string) (ChannelList, error) {
	resp, err := http.Get(fmt.Sprintf("https://slack.com/api/channels.list?token=%s", token))
	if err != nil {
		logrus.Error(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
	}

	var channelList ChannelList
	err = json.Unmarshal(body, &channelList)

	return channelList, err
}

func (msgHandler *MessageHandler) registerCommand(matchPattern string, h HandlerFunc) {
	r, err := regexp.Compile(matchPattern)

	if err != nil {
		logrus.WithField("pattern", matchPattern).Panic("Unable to compile command")
	}

	msgHandler.Handlers[matchPattern] = Handler{
		pattern:     r,
		handlerFunc: h,
	}
}

func (msg Message) isMessage() bool {
	return msg.Type == "message"
}

func (msg Message) isPrivate() bool {
	if msg.Channel != "" && strings.HasPrefix(msg.Channel, "D") {
		return true
	}
	return false
}

// Find a message command which matches the command regex
func (msgHandler MessageHandler) match(msg *Message) (*Handler, []string) {
	return msgHandler.Matcher(msgHandler.Handlers, msg)
}

func basicMatch(handlers map[string]Handler, msg *Message) (cmd *Handler, result []string) {
	for _, v := range handlers {
		result = v.pattern.FindStringSubmatch(msg.Text)
		if result != nil {
			cmd = &v
			return
		}
	}
	return
}

var defaultMessageHandler = &MessageHandler{
	Handlers: make(map[string]Handler),
	Matcher:  basicMatch,
}

// NewRobot create a new robot for us to play with
func NewRobot(origin, token string) *Robot {
	return &Robot{
		Origin:     origin,
		APIToken:   token,
		Handler:    defaultMessageHandler,
		Client:     &SlackWebClient{HTTPClient: &http.Client{}},
		ListenChan: make(chan Message), SendChan: make(chan Message),
		Shutdown: make(chan int),
	}
}

// SlackConnect main entry point to setup a websocket we can communicate with the
// Slack API over
func (robot *Robot) SlackConnect() {
	slackResponse, err := slackStart(robot.APIToken)
	if err != nil {
		logrus.Fatal(err)
	}

	websock, err := websocket.Dial(slackResponse.URL, "", robot.Origin)
	if err != nil {
		logrus.Fatal(err)
	}

	robot.ID = slackResponse.Self.ID
	robot.Name = slackResponse.Self.Name
	robot.Connection = websock

	logrus.WithFields(logrus.Fields{
		"robot_id":   slackResponse.Self.ID,
		"robot_name": slackResponse.Self.Name,
	}).Info("Connected to Slack!")
}

// TODO:: Remove these but still allow for dependency injection in tests
// Do so by mocking out the websocket instead
var receiveOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
	return websocket.JSON.Receive(conn, msg)
}

// TODO:: Remove these but still allow for dependency injection in tests
// Do so by mocking out the websocket instead
var sendOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
	return websocket.JSON.Send(conn, msg)
}

func sendViaRPC(client WebClienter, token, channel, text string, attachments []Attachment) (*http.Response, error) {
	a, err := json.Marshal(attachments)
	if err != nil {
		logrus.Error("Error PostMessage to slack: ", err)
	}

	req, _ := http.NewRequest("GET", "https://slack.com/api/chat.postMessage", nil)
	q := req.URL.Query()
	q.Add("token", token)
	q.Add("channel", channel)
	q.Add("text", text)
	q.Add("attachments", string(a))
	q.Add("as_user", "true")
	req.URL.RawQuery = q.Encode()
	return client.Do(req)
}

// Listen fires of a go routine which listens to messages recieved from the
// Websocket and adds the to the channel
func (robot *Robot) Listen() {
	go func() {
		for {
			msg := &Message{}
			err := receiveOverWebsocket(robot.Connection, msg)
			if err != nil {
				logrus.Error("Error receiving over websocket: ", err.Error())
			}
			robot.ListenChan <- *msg
		}
	}()

	return
}

// SendMessage to a message on slack
func (robot Robot) SendMessage(channel, msg string) (err error) {
	message := &Message{
		ID:      atomic.AddUint64(&counter, 1),
		Type:    "message",
		Channel: channel,
		Text:    msg,
	}
	return sendOverWebsocket(robot.Connection, message)
}

// PostMessage uses the slack API over the RTM api to allow rich messages to be
// created
func (robot Robot) PostMessage(channel, msg string, attachment Attachment) {
	attachments := []Attachment{attachment}
	resp, err := sendViaRPC(robot.Client, robot.APIToken, channel, msg, attachments)
	if err != nil {
		logrus.Error("Error posting to slack api: ", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
	}
	var postResponse PostResponse

	err = json.Unmarshal(body, &postResponse)
	if err != nil {
		logrus.Error("Unable to decode json response from post endpoint: ", err)
	}
	if !postResponse.Ok {
		logrus.Error("Error posting message: ", postResponse.Error)
	}
}

// RegisterCommands a command to check against incoming messages and the function
// to be triggered
func (robot Robot) RegisterCommands(cmds map[string]HandlerFunc) {
	for pattern, handler := range cmds {
		robot.Handler.registerCommand(pattern, handler)
	}
}

func (robot Robot) match(msg *Message) (cmd *Handler, result []string) {
	return robot.Handler.match(msg)
}

func (robot Robot) continueConversation(msg *Message) {
	poll := Poll{}
	GetDB().Where("creator = ? AND channel = ? AND stage != ? ", msg.User, msg.Channel, "active").First(&poll)

	nextCmd, ok := stageLookup[poll.Stage]
	if ok != true {
		logrus.WithFields(logrus.Fields{
			"Channel": msg.Channel,
			"User":    msg.User,
			"Text":    msg.Text,
			"Stage":   poll.Stage,
		}).Error("Stage lookup for poll failed")
		return
	}

	if err := nextCmd(&robot, msg, &poll); err != nil {
		logrus.WithFields(logrus.Fields{
			"command":   nextCmd,
			"user":      msg.User,
			"channel":   msg.Channel,
			"poll_id":   poll.ID,
			"poll_name": poll.Name,
		}).Error("Error continuing conversation: ", err)
	}

}

// Dispatch parses the incoming message and dispatches to the best matched
// command
func (robot *Robot) Dispatch(msg *Message) {
	if msg.DirectMention == true || msg.isPrivate() == true {
		cmd, captureGroups := robot.match(msg)

		if cmd != nil {

			logrus.WithFields(logrus.Fields{
				"Channel": msg.Channel,
				"User":    msg.User,
				"Text":    msg.Text,
			}).Info("Matched command")

			err := cmd.handlerFunc(robot, msg, captureGroups)
			logrus.Error(err)
			return
		}

		robot.continueConversation(msg)
	}
}

// ProcessMessage finds the approciate handler to start a conversation
// on building or resonding to a poll
func (robot Robot) ProcessMessage(msg *Message) {
	if msg.isMessage() != true {
		return
	}

	if strings.HasPrefix(msg.Text, "<@"+robot.ID+">") {
		msg.DirectMention = true
		msg.Text = strings.Replace(msg.Text, robot.SlackIDString()+":", "", -1)
		msg.Text = strings.Trim(msg.Text, " ")
	}

	if msg.User != robot.ID {
		robot.Dispatch(msg)
	}
}

// SlackIDString returns the slack formated string containing the robots identifier
func (robot *Robot) SlackIDString() string {
	return "<@" + robot.ID + ">"
}

// DownloadUsersMap retrieves the list of users and channels that are available
// we need this information for when recipients are listed to go and retrive
// names etc. We store in memory but should move to a cache later one.
// This should be refreshed periodically
func (robot *Robot) DownloadUsersMap() {
	logrus.Info("Downloading channel listings")
	channelList, _ := downloadChannelList(robot.APIToken)
	if !channelList.Ok {
		logrus.Panic("Unable to download channels list from Slack: ", channelList.Error)
	}
	logrus.Info("Finished downloading channel listings")
	logrus.Info("Downloading user listings")
	users, _ := downloadUserList(robot.APIToken)
	if !users.Ok {
		logrus.Panic("Unable to download users list from Slack: ", users.Error)
	}
	logrus.Info("Finished downloading user listings")

	robot.Channels = channelList.Channels
	robot.Users = users.Members
}

// Run is the entry point for Carlos to setup a connection with Slack and
// the channels we use for listening and posting messages
func (robot Robot) Run() {
	robot.SlackConnect()
	robot.DownloadUsersMap()
	robot.Listen()
	robot.RegisterCommands(registeredCommands)
	logrus.Info("Ready and waiting for messages")

	checkInterval := time.NewTicker(time.Duration(12) * time.Hour)
	for {
		select {
		case msg := <-robot.ListenChan:
			robot.ProcessMessage(&msg)
		case <-robot.Shutdown:
			return
		case <-checkInterval.C:
			go robot.DownloadUsersMap()
		}
	}
}

func slackStart(token string) (*ResponseRTMStart, error) {
	url := fmt.Sprintf("https://slack.com/api/rtm.start?no_unreads=true&token=%s", token)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var startResponse ResponseRTMStart
	if err = json.Unmarshal(body, &startResponse); err != nil {
		return nil, err
	}

	if !startResponse.Ok {
		err = fmt.Errorf("Slack initialization error: %s", startResponse.Error)
		return nil, err
	}

	return &startResponse, nil
}
