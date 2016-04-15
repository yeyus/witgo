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

package witgo

import (
	"fmt"
)

type Value struct {
	Expressions []string `json:expressions`
	Value       string   `json:value`
}

type Entity struct {
	Lang    string   `json:lang`
	Closed  bool     `json:closed`
	Exotic  bool     `json:exotic`
	Value   string   `json:value`
	Values  []*Value `json:values`
	Builtin bool     `json:builtin`
	Doc     string   `json:doc`
	Name    string   `json:name`
	ID      string   `json:id`
}

type EntityMap map[string][]*Entity

func (m EntityMap) FirstEntityValue(key string) (out string, err error) {
	var (
		entities []*Entity
		found    bool
	)
	if entities, found = m[key]; !found || len(entities) == 0 {
		err = fmt.Errorf("No entities associated with key %v", key)
		return
	}
	out = entities[0].Value
	return
}

type ConverseResponse struct {
	Type       string    `json:type`
	Msg        string    `json:msg`
	Action     string    `json:action`
	Entities   EntityMap `json:entities`
	Confidence float64   `json:confidence`
}
