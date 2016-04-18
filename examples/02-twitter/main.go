// Copyright 2016 Arne Roomann-Kurrik
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"github.com/kurrik/witgo/v1/witgo"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MINWAIT = time.Duration(10) * time.Second

type TwitterUser struct {
	ID         int64  `json:id`
	Name       string `json:name`
	ScreenName string `json:screen_name`
	Following  bool   `json:following`
}

type DirectMessage struct {
	ID        int64       `json:id`
	Text      string      `json:text`
	Sender    TwitterUser `json:sender`
	Recipient TwitterUser `json:recipient`
	CreatedAt string      `json:created_at`
}

type DirectMessageList []DirectMessage

func (l DirectMessageList) Len() int           { return len(l) }
func (l DirectMessageList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l DirectMessageList) Less(i, j int) bool { return l[i].ID < l[j].ID }

type directMessageRequest struct {
	Text   string
	UserID int64
}

type TwitterClient struct {
	client        *twittergo.Client
	processedToID int64
	PollInterval  time.Duration
	outgoing      chan directMessageRequest
}

func NewTwitterClient(cred Credentials) (out *TwitterClient) {
	var (
		config *oauth1a.ClientConfig
		user   *oauth1a.UserConfig
	)
	config = &oauth1a.ClientConfig{
		ConsumerKey:    cred.TwitterConsumerKey,
		ConsumerSecret: cred.TwitterConsumerSecret,
	}
	user = oauth1a.NewAuthorizedConfig(
		cred.TwitterAccessToken,
		cred.TwitterAccessTokenSecret,
	)
	out = &TwitterClient{
		client:       twittergo.NewClient(config, user),
		PollInterval: time.Minute,
	}
	return
}

func (c *TwitterClient) SetProcessedMarkerToCurrent() (err error) {
	var (
		existing DirectMessageList
	)
	if existing, err = c.fetchDirectMessages(0, 1); err != nil {
		return
	}
	if len(existing) > 0 {
		c.processedToID = existing[0].ID
	}
	return
}

func (c *TwitterClient) fetchDirectMessages(sinceID int64, count int) (data DirectMessageList, err error) {
	var (
		req    *http.Request
		reqURL string
		query  url.Values
		resp   *twittergo.APIResponse
	)
	query = url.Values{}
	query.Set("since_id", fmt.Sprintf("%v", sinceID))
	query.Set("count", fmt.Sprintf("%v", count))
	reqURL = fmt.Sprintf("/1.1/direct_messages.json?%v", query.Encode())
	if req, err = http.NewRequest("GET", reqURL, nil); err != nil {
		return
	}
	if resp, err = c.client.SendRequest(req); err != nil {
		return
	}
	if err = resp.Parse(&data); err != nil {
		return
	}
	return
}

func (c *TwitterClient) handleFetchError(err error) error {
	switch e := err.(type) {
	case twittergo.RateLimitError:
		dur := e.Reset.Sub(time.Now()) + time.Second
		if dur < MINWAIT {
			dur = MINWAIT
		}
		fmt.Printf("Rate limited, reset at %v, waiting for %v\n", e.Reset, dur)
		time.Sleep(dur)
		return nil
	}
	return err
}

func (c *TwitterClient) makeSessionID(userID int64) witgo.SessionID {
	return witgo.SessionID(fmt.Sprintf("%v-%v", userID, time.Now().Unix()))
}

func (c *TwitterClient) parseSessionID(sessionID witgo.SessionID) (userID int64, err error) {
	var lines []string
	lines = strings.Split(string(sessionID), "-")
	userID, err = strconv.ParseInt(lines[0], 10, 64)
	return
}

// Drains the channel, we don't do anything with these requests.
func (c *TwitterClient) runPending(requests <-chan witgo.SessionID) {
	var req witgo.SessionID
	for req = range requests {
		fmt.Printf("Discarding request %v\n", req)
	}

}

func (c *TwitterClient) runFetch(records chan<- witgo.InputRecord) {
	var (
		messages DirectMessageList
		message  DirectMessage
		err      error
		tick     <-chan time.Time
	)
	defer close(records)
	tick = time.Tick(c.PollInterval)
	for true {
		fmt.Printf("Requesting direct messages\n")
		if messages, err = c.fetchDirectMessages(c.processedToID, 100); err != nil {
			if err = c.handleFetchError(err); err != nil {
				fmt.Printf("ERROR: %v\n", err)
				return
			}
		}
		fmt.Printf("Got %v messages\n", len(messages))
		sort.Sort(messages)
		for _, message = range messages {
			if message.ID > c.processedToID {
				records <- witgo.InputRecord{
					SessionID: c.makeSessionID(message.Sender.ID),
					Query:     message.Text,
				}
				c.processedToID = message.ID
			}
		}
		fmt.Printf("Waiting for next poll interval\n")
		<-tick
	}
	return
}

func (c *TwitterClient) sendMessage(msg directMessageRequest) (err error) {
	var (
		req    *http.Request
		reqURL string
		query  url.Values
	)
	query = url.Values{}
	query.Set("user_id", fmt.Sprintf("%v", msg.UserID))
	query.Set("text", msg.Text)
	reqURL = fmt.Sprintf("/1.1/direct_messages/new.json?%v", query.Encode())
	if req, err = http.NewRequest("POST", reqURL, nil); err != nil {
		return
	}
	if _, err = c.client.SendRequest(req); err != nil {
		return
	}
	return
}

func (c *TwitterClient) runWrite(messages <-chan directMessageRequest) {
	var (
		msg   directMessageRequest
		err   error
		retry bool
	)
	for msg = range messages {
		retry = true
		for retry {
			fmt.Printf("Sending `%v` to user %v\n", msg.Text, msg.UserID)
			if err = c.sendMessage(msg); err != nil {
				if err = c.handleFetchError(err); err != nil {
					fmt.Printf("ERROR: %v, will not retry write\n", err)
					retry = false
				}
			} else {
				retry = false
			}
		}
	}
	return
}

func (c *TwitterClient) Run() (chan<- witgo.SessionID, <-chan witgo.InputRecord) {
	var (
		requests = make(chan witgo.SessionID)
		records  = make(chan witgo.InputRecord)
	)
	c.outgoing = make(chan directMessageRequest, 10)
	go c.runPending(requests)
	go c.runFetch(records)
	go c.runWrite(c.outgoing)
	return requests, records
}

func (c *TwitterClient) Action(session *witgo.Session, action string) (response *witgo.Session, err error) {
	response = session
	response.Context.Set("forecast", "sunny")
	return
}

func (c *TwitterClient) Say(session *witgo.Session, msg string) (response *witgo.Session, err error) {
	var userID int64
	response = session
	if userID, err = c.parseSessionID(session.ID()); err != nil {
		return
	}
	c.outgoing <- directMessageRequest{
		UserID: userID,
		Text:   msg,
	}
	return
}

func (c *TwitterClient) Merge(session *witgo.Session, entities witgo.EntityMap) (response *witgo.Session, err error) {
	var (
		value string
	)
	response = session
	if value, err = entities.FirstEntityValue("location"); err != nil {
		response.Context = witgo.Context{}
		err = nil
	} else {
		response.Context.Merge(witgo.Context{
			"loc": value,
		})
	}
	return
}

func (c *TwitterClient) Error(session *witgo.Session, msg string) {
}

var noCredentialsErr = fmt.Errorf(`You must specify a credentials path using the -credentials flag!
This must point to a 4-line file with the following format:

<Twitter consumer key>
<Twitter consumer secret>
<Twitter access token>
<Twitter access token secret>
<Wit.ai token>`)

type Credentials struct {
	TwitterConsumerKey       string
	TwitterConsumerSecret    string
	TwitterAccessToken       string
	TwitterAccessTokenSecret string
	WitgoServerToken         string
}

func loadCredentials(path string) (out Credentials, err error) {
	var (
		credentials []byte
		lines       []string
	)
	if path == "" {
		err = noCredentialsErr
		return
	}
	if credentials, err = ioutil.ReadFile(path); err != nil {
		return
	}
	lines = strings.Split(string(credentials), "\n")
	if len(lines) < 5 {
		err = fmt.Errorf("Credentials file did not have enough lines!")
		return
	}
	out = Credentials{
		TwitterConsumerKey:       lines[0],
		TwitterConsumerSecret:    lines[1],
		TwitterAccessToken:       lines[2],
		TwitterAccessTokenSecret: lines[3],
		WitgoServerToken:         lines[4],
	}
	return
}

func processError(err error) {
	switch e := err.(type) {
	case twittergo.RateLimitError:
		fmt.Printf("Rate limited, reset at %v\n", e.Reset)
	case twittergo.Errors:
		for i, val := range e.Errors() {
			fmt.Printf("Error #%v - ", i+1)
			fmt.Printf("Code: %v ", val.Code())
			fmt.Printf("Msg: %v\n", val.Message())
		}
	default:
		fmt.Printf("There was an error running the script: %v\n", err)
	}
	os.Exit(1)
}

func main() {
	var (
		credentialsPath string
		credentials     Credentials
		processedTo     int64
		err             error
		twitter         *TwitterClient
		witai           *witgo.Witgo
	)
	flag.StringVar(&credentialsPath, "credentials", "", "Path to credentials file")
	flag.Int64Var(&processedTo, "processedTo", -1, "Override ID to start processing from")
	flag.Parse()
	if credentials, err = loadCredentials(credentialsPath); err != nil {
		processError(err)
	}
	twitter = NewTwitterClient(credentials)
	if processedTo == -1 {
		if err = twitter.SetProcessedMarkerToCurrent(); err != nil {
			processError(err)
		}
	} else {
		twitter.processedToID = processedTo
	}
	fmt.Printf("Processed to: %v\n", twitter.processedToID)
	witai = witgo.NewWitgo(
		witgo.NewClient(credentials.WitgoServerToken),
		twitter,
	)
	if err = witai.Process(twitter); err != nil {
		processError(err)
	}
}
