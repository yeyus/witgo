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

// Implements a Wit.ai client library in Go.
package witgo

import (
	"strings"
)

type Handler interface {
	Action(session *Session, action string) (response *Session, err error)
	Say(session *Session, msg string) (response *Session, err error)
	Merge(session *Session, entities EntityMap) (response *Session, err error)
	Error(session *Session, msg string)
}

type Witgo struct {
	client  *Client
	handler Handler
}

func NewWitgo(client *Client, handler Handler) *Witgo {
	return &Witgo{
		client:  client,
		handler: handler,
	}
}

func (w *Witgo) process(session *Session, q string) (out *Session, err error) {
	var (
		response *Response
		converse *ConverseResponse
		done     bool = false
	)
	for !done {
		if response, err = w.client.Converse(session.ID(), q, session.Context); err != nil {
			return
		}
		if err = response.Parse(&converse); err != nil {
			return
		}
		switch strings.ToLower(converse.Type) {
		case "action":
			if session, err = w.handler.Action(session, converse.Action); err != nil {
				return
			}
		case "msg":
			if session, err = w.handler.Say(session, converse.Msg); err != nil {
				return
			}
		case "merge":
			if session, err = w.handler.Merge(session, converse.Entities); err != nil {
				return
			}
		case "stop":
			done = true
		default:
			done = true
		}
		q = ""
	}
	out = session
	return
}

func (w *Witgo) Process(input Input) (err error) {
	var (
		record   InputRecord
		session  *Session
		found    bool
		sessions = map[SessionID]*Session{}
		requests chan<- SessionID
		records  <-chan InputRecord
	)
	requests, records = input.Run()
	for record = range records {
		if session, found = sessions[record.SessionID]; !found {
			session = NewSession(record.SessionID)
		}
		if session, err = w.process(session, record.Query); err != nil {
			return
		}
		sessions[record.SessionID] = session
		requests <- record.SessionID
	}
	return
}
