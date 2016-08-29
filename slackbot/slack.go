package slackbot

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"

	"golang.org/x/net/websocket"
)

var (
	counter uint64 //All message sent during a session need to have an monotonically increasing id
)

const (
	PublicChannelID SlackIDType = iota
	GroupChannelID  SlackIDType = iota
	UserID          SlackIDType = iota
	UnknownID       SlackIDType = iota
)

type SlackID struct {
	Value string
}

type SlackIDType int

func (id *SlackID) Kind() SlackIDType {
	switch string(id.Value[0]) {
	case "G":
		return GroupChannelID
	case "C":
		return PublicChannelID
	case "U":
		return UserID
	default:
		logrus.Error("Unable to identify the kind of channel from id:", id.Value)
		return UnknownID
	}
}

type responseRTMSelf struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelList struct {
	Ok       bool      `json:"ok"`
	Channels []Channel `json:"channels"`
	Error    string    `json:"error,omitempty"`
}

type Channel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	IsChannel string   `json:"is_channel"`
	Members   []string `json:"members"`
}

type GroupList struct {
	Ok     bool    `json:"ok"`
	Groups []Group `json:"groups"`
	Error  string  `json:"error,omitempty"`
}

type Group struct {
	ID         string   `json:"id"`
	Created    uint     `json:"created"`
	Creator    string   `json:"creator"`
	IsArchived bool     `json:"is_archived"`
	IsGroup    bool     `json:"is_group"`
	IsMpim     bool     `json:"is_mpim"`
	Members    []string `json:"members"`
	Name       string   `json:"name"`
}

type UserList struct {
	Ok      bool   `json:"ok"`
	Members []User `json:"members"`
	Error   string `json:"error,omitempty"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResponseRTMStart struct {
	Ok    bool             `json:"ok"`
	Error string           `json:"error"`
	URL   string           `json:"url"`
	Self  *responseRTMSelf `json:"self"`
}

type HandlerFunc func(*Robot, *Message, []string) error

type Stage func(*Robot, *Message, *Poll) error

type MessageHandler struct {
	Handlers map[string]Handler
	Matcher  func(handlers map[string]Handler, msg *Message) (cmd *Handler, result []string)
}

type Handler struct {
	pattern     *regexp.Regexp
	handlerFunc HandlerFunc
}

type WebClienter interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type SlackWebClient struct {
	HTTPClient *http.Client
}

func (client *SlackWebClient) Do(req *http.Request) (resp *http.Response, err error) {
	return client.HTTPClient.Do(req)
}

type Robot struct {
	ID         string
	Name       string
	Origin     string
	APIToken   string
	Users      map[string]User
	Client     WebClienter // http.Client
	Handler    *MessageHandler
	Shutdown   chan int
	Channels   map[string]Channel
	Groups     map[string]Group
	Connection *websocket.Conn
	ListenChan chan Message
}

type Message struct {
	ID            uint64    `json:"id"`
	Type          string    `json:"type"`
	Subtype       string    `json:"subtype"`
	Channel       string    `json:"channel"`
	User          string    `json:"user"`
	Text          string    `json:"text"`
	Timestamp     string    `json:"ts"`
	Handled       bool      `json:"-"` // Did message match a handler?
	DirectMention bool      `json:"-"` // Does message contain a direct mention
	CaptureGroup  []string  `json:"-"` // hold the capture group when a command is matched
	Processed     chan bool `json:"-"` // Channel to signal message has been processed
}

type Attachment struct {
	Title   string            `json:"title"`
	Text    string            `json:"text"`
	Pretext string            `json:"pretext"`
	Fields  []AttachmentField `json:"fields"`
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

var GenerateUUID = func() string {
	return uuid.NewV4().String()
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

func downloadGroupList(token string) (GroupList, error) {
	resp, err := http.Get(fmt.Sprintf("https://slack.com/api/groups.list?token=%s", token))
	if err != nil {
		logrus.Error(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
	}

	var groupList GroupList
	err = json.Unmarshal(body, &groupList)

	return groupList, err
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

func NewRobot(origin, token string) *Robot {
	return &Robot{
		Origin:     origin,
		APIToken:   token,
		Handler:    defaultMessageHandler,
		Client:     &SlackWebClient{HTTPClient: &http.Client{}},
		ListenChan: make(chan Message, 100),
		Shutdown:   make(chan int),
	}
}

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

var receiveOverWebsocket = func(conn *websocket.Conn, msg *Message) error {
	return websocket.JSON.Receive(conn, msg)
}

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

func (robot *Robot) Listen() {
	go func() {
		for {
			msg := &Message{}
			err := receiveOverWebsocket(robot.Connection, msg)
			if err != nil {
				logrus.Error("Error receiving over websocket: ", err.Error())
			}

			if msg.isMessage() != true {
				continue
			}

			msg.Processed = make(chan bool)
			robot.ListenChan <- *msg
		}
	}()
}

func (robot Robot) SendMessage(channel, msg string) (err error) {
	message := &Message{
		ID:      atomic.AddUint64(&counter, 1),
		Type:    "message",
		Channel: channel,
		Text:    msg,
	}
	return sendOverWebsocket(robot.Connection, message)
}

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
			"poll_uuid": poll.UUID,
		}).Error("Error continuing conversation: ", err)
	}

}

func (robot *Robot) Dispatch(msg *Message) {
	if msg.DirectMention == true || msg.isPrivate() == true {
		cmd, captureGroups := robot.match(msg)

		if cmd != nil {
			logrus.WithFields(logrus.Fields{
				"Channel": msg.Channel,
				"User":    msg.User,
				"Text":    msg.Text,
			}).Info("Matched command")

			if err := cmd.handlerFunc(robot, msg, captureGroups); err != nil {
				logrus.Error(err)
			}
			return
		}
		robot.continueConversation(msg)
	}
}

func (robot Robot) ProcessMessage(msg *Message) {
	if strings.HasPrefix(msg.Text, "<@"+robot.ID+">") {
		msg.DirectMention = true
		msg.Text = strings.Replace(msg.Text, robot.SlackIDString()+":", "", -1)
		msg.Text = strings.Trim(msg.Text, " ")
	}

	if msg.User != robot.ID {
		robot.Dispatch(msg)
	}

	msg.Processed <- true
}

func (robot *Robot) SlackIDString() string {
	return "<@" + robot.ID + ">"
}

func (robot *Robot) DownloadGroups() {
	groups, _ := downloadGroupList(robot.APIToken)
	if !groups.Ok {
		logrus.Fatal("Unable to download channels list from Slack: ", groups.Error)
	}

	groupMap := make(map[string]Group)
	for _, group := range groups.Groups {
		groupMap[group.ID] = group
	}
	robot.Groups = groupMap
}

func (robot *Robot) DownloadChannels() {
	channels, _ := downloadChannelList(robot.APIToken)
	if !channels.Ok {
		logrus.Fatal("Unable to download channels list from Slack: ", channels.Error)
	}

	channelMap := make(map[string]Channel)
	for _, channel := range channels.Channels {
		channelMap[channel.ID] = channel
	}
	robot.Channels = channelMap
}

func (robot *Robot) DownloadUsers() {
	users, _ := downloadUserList(robot.APIToken)
	if !users.Ok {
		logrus.Fatal("Unable to download users list from Slack: ", users.Error)
	}
	userMap := make(map[string]User)
	for _, user := range users.Members {
		userMap[user.ID] = user
	}
	robot.Users = userMap
}

func (robot *Robot) DownloadUsersMap() {
	logrus.Info("Downloading information from slack")
	robot.DownloadUsers()
	robot.DownloadChannels()
	robot.DownloadGroups()
	logrus.Info("Finished downloading users, channels, and group information")
}

func HerokuServer() {
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8000"
		}

		logrus.Info("listening on port:", port)
		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "pong")
		})

		err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)

		if err != nil {
			logrus.Fatal(err)
		}
	}()
}

func HerokuPing() {
	go func() {
		pingInterval := time.NewTicker(time.Duration(5) * time.Minute)
		for {
			select {
			case <-pingInterval.C:
				logrus.Info("Pinging Heroku server")
				http.Get("https://carlos-the-curious.herokuapp.com/status")
			}
		}
	}()
}
	for {
		select {
		case <-pingInterval.C:
			logrus.Info("Pinging Heroku server")
			http.Get("https://carlos-the-curious.herokuapp.com/status")
		}
	}
}

func (robot Robot) Run() {
func SetupHerokuKeepAlive() {
	if os.Getenv("PLATFORM") == "HEROKU" {
		logrus.Info("Heroku Platform detected running webserver and keepalive status ping")
		HerokuServer()
		HerokuPing()
	}
}
	SetupHerokuKeepAlive()

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
