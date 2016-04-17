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
	"github.com/kurrik/witgo/v1/witgo"
	"os"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Action(session *witgo.Session, action string) (response *witgo.Session, err error) {
	response = session
	response.Context.Set("forecast", "sunny")
	return
}

func (h *Handler) Say(session *witgo.Session, msg string) (response *witgo.Session, err error) {
	response = session
	fmt.Printf("< %v\n", msg)
	return
}

func (h *Handler) Merge(session *witgo.Session, entities witgo.EntityMap) (response *witgo.Session, err error) {
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

func (h *Handler) Error(session *witgo.Session, msg string) {
}

func processError(err error) {
	fmt.Printf("There was an error running the script: %v\n", err)
	os.Exit(1)
}

func main() {
	var (
		client *witgo.Client
		wg     *witgo.Witgo
		input  witgo.Input
		err    error
		token  string
		debug  bool
	)
	flag.StringVar(&token, "token", "", "Server access token for wit.ai")
	flag.BoolVar(&debug, "debug", false, "Print extra debugging information")
	flag.Parse()
	if token == "" {
		processError(fmt.Errorf("You must specify a server access token using the -token flag!"))
	}
	client = witgo.NewClient(token)
	if debug {
		client.HttpClient = witgo.NewLoggingHttpClient(os.Stderr, client.HttpClient)
	}
	wg = witgo.NewWitgo(client, NewHandler())
	input = witgo.NewInteractiveInput()
	if err = wg.Process(input); err != nil {
		processError(err)
	}
}
