/*
 * Copyright 2020 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package url

import (
	"net/url"
	"sort"
	"strings"
)

type nameValuePair struct {
	name, value string
}

type searchParams struct {
	url    *Url
	params []*nameValuePair
}

func (s *searchParams) init(query string) {
	p := strings.Split(query, "&")
	for _, q := range p {
		if q == "" {
			continue
		}
		kv := strings.SplitN(q, "=", 2)
		name, _ := url.QueryUnescape(kv[0])
		nvp := &nameValuePair{name: name}
		if len(kv) == 2 {
			value, _ := url.QueryUnescape(kv[1])
			nvp.value = value
		}
		s.params = append(s.params, nvp)
	}
}

func (s *searchParams) update() {
	query := s.String()
	if query == "" {
		s.url.search = nil
	} else {
		s.url.search = &query
	}
}

func (s *searchParams) Append(name, value string) {
	s.params = append(s.params, &nameValuePair{name: name, value: value})
	s.update()
}

func (s *searchParams) Delete(name string) {
	var result []*nameValuePair
	for _, nvp := range s.params {
		if nvp.name != name {
			result = append(result, nvp)
		}
	}
	s.params = result
	s.update()
}

func (s *searchParams) Get(name string) string {
	for _, nvp := range s.params {
		if nvp.name == name {
			return nvp.value
		}
	}
	return ""
}

func (s *searchParams) GetAll(name string) []string {
	var result []string
	for _, nvp := range s.params {
		if nvp.name == name {
			result = append(result, nvp.value)
		}
	}
	return result
}

func (s *searchParams) Has(name string) bool {
	for _, nvp := range s.params {
		if nvp.name == name {
			return true
		}
	}
	return false
}

func (s *searchParams) Set(name, value string) {
	isSet := false
	for idx, nvp := range s.params {
		if nvp.name == name {
			if isSet {
				s.params = append(s.params[:idx], s.params[idx+1:]...)
			} else {
				nvp.value = value
				isSet = true
			}
		}
	}
	if !isSet {
		s.Append(name, value)
	}
	s.update()
}

func (s *searchParams) Sort() {
	sort.SliceStable(s.params, func(i, j int) bool {
		return s.params[i].name < s.params[j].name
	})
	s.update()
}

func (s *searchParams) String() string {
	output := strings.Builder{}
	for idx, nvp := range s.params {
		if idx > 0 {
			output.WriteRune('&')
		}
		output.WriteString(url.QueryEscape(nvp.name))
		if nvp.value != "" {
			output.WriteRune('=')
			output.WriteString(url.QueryEscape(nvp.value))
		}
	}
	return output.String()
}
