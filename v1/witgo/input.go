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
	"bufio"
	"fmt"
	"os"
	"strings"
)

type InputRecord struct {
	SessionID
	Query string
}

type Input interface {
	Run() (requests chan<- SessionID, records <-chan InputRecord)
}

type InteractiveInput struct {
}

func NewInteractiveInput() *InteractiveInput {
	return &InteractiveInput{}
}

func (i *InteractiveInput) run(requests <-chan SessionID, records chan<- InputRecord) {
	var (
		reader  *bufio.Reader
		session SessionID
		line    string
		err     error
	)
	reader = bufio.NewReader(os.Stdin)
	defer close(records)
	fmt.Printf("Interactive mode (use ':quit' to stop)\n")
	for true {
		session = <-requests
		fmt.Printf("%v> ", session)
		if line, err = reader.ReadString('\n'); err != nil {
			return
		}
		if strings.ToLower(strings.TrimSpace(line)) == ":quit" {
			return
		}
		records <- InputRecord{
			SessionID: session,
			Query:     line,
		}
	}
	return
}

func (i *InteractiveInput) Run() (chan<- SessionID, <-chan InputRecord) {
	var (
		requests = make(chan SessionID)
		records  = make(chan InputRecord)
	)
	go i.run(requests, records)
	requests <- "interactive" // Start interactive loop.
	return requests, records
}
