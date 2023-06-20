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
	"sort"
	"strings"
)

type NameValuePair struct {
	Name, Value string
}

type SearchParams struct {
	url    *Url
	params []*NameValuePair
}

func (s *SearchParams) init(query string) {
	s.params = s.params[:0]
	p := strings.Split(query, "&")
	for _, q := range p {
		if q == "" {
			continue
		}
		kv := strings.SplitN(q, "=", 2)
		name := s.url.parser.DecodePercentEncoded(kv[0])
		name = strings.ReplaceAll(name, "+", " ")
		nvp := &NameValuePair{Name: name}
		if len(kv) == 2 {
			value := s.url.parser.DecodePercentEncoded(kv[1])
			value = strings.ReplaceAll(value, "+", " ")
			nvp.Value = value
		}
		s.params = append(s.params, nvp)
	}
}

func (s *SearchParams) update() {
	if s.url == nil {
		return
	}
	query := s.String()
	if (query == "" && s.url.query != nil) || query != "" {
		s.url.query = &query
	}
}

func (s *SearchParams) Append(name, value string) {
	s.params = append(s.params, &NameValuePair{Name: name, Value: value})
	s.update()
}

func (s *SearchParams) Delete(name string) {
	var result []*NameValuePair
	for _, nvp := range s.params {
		if nvp.Name != name {
			result = append(result, nvp)
		}
	}
	s.params = result
	s.update()
}

func (s *SearchParams) Get(name string) string {
	for _, nvp := range s.params {
		if nvp.Name == name {
			return nvp.Value
		}
	}
	return ""
}

func (s *SearchParams) GetAll(name string) []string {
	var result []string
	for _, nvp := range s.params {
		if nvp.Name == name {
			result = append(result, nvp.Value)
		}
	}
	return result
}

func (s *SearchParams) Has(name string) bool {
	for _, nvp := range s.params {
		if nvp.Name == name {
			return true
		}
	}
	return false
}

func (s *SearchParams) Set(name, value string) {
	isSet := false
	params := s.params[:0]
	for i, nvp := range s.params {
		if nvp.Name == name {
			if isSet {
				s.params[i] = nil
				continue
			}
			nvp.Value = value
			isSet = true
		}
		params = append(params, nvp)
	}
	if !isSet {
		s.params = append(params, &NameValuePair{Name: name, Value: value})
	} else {
		s.params = params
	}
	s.update()
}

func (s *SearchParams) Sort() {
	sort.SliceStable(s.params, func(i, j int) bool {
		return s.params[i].Name < s.params[j].Name
	})
	s.update()
}

func (s *SearchParams) SortAbsolute() {
	sort.SliceStable(s.params, func(i, j int) bool {
		return s.params[i].Name+s.params[i].Value < s.params[j].Name+s.params[j].Value
	})
	s.update()
}

func (s *SearchParams) Iterate(f func(pair *NameValuePair)) {
	for _, nvp := range s.params {
		f(nvp)
	}
	s.update()
}

func (s *SearchParams) String() string {
	output := strings.Builder{}
	for idx, nvp := range s.params {
		if idx > 0 {
			output.WriteRune('&')
		}

		s.QueryEscape(nvp.Name, &output)
		if !s.url.parser.opts.skipEqualsForEmptySearchParamsValue || nvp.Value != "" {
			output.WriteRune('=')
		}
		if nvp.Value != "" {
			s.QueryEscape(nvp.Value, &output)
		}
	}
	return output.String()
}

func (s *SearchParams) QueryEscape(st string, output *strings.Builder) {
	for _, b := range st {
		if b == 0x0020 {
			output.WriteRune(0x002B)
		} else {
			output.WriteString(s.url.parser.percentEncodeRune(b, s.url.parser.opts.queryPercentEncodeSet))
		}
	}
}
