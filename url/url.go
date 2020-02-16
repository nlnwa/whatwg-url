/*
 * Copyright 2019 National Library of Norway.
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

type Url struct {
	inputUrl         string
	protocol         string
	username         string
	password         string
	host             *string
	port             *string
	path             []string
	search           *string
	hash             *string
	cannotBeABaseUrl bool
	searchParams     *searchParams
	validationErrors []error
}

func (u *Url) Href() string {
	output := u.protocol + ":"
	if u.host != nil {
		output += "//"
		if u.username != "" || u.password != "" {
			output += u.username
			if u.password != "" {
				output += ":" + u.password
			}
			output += "@"
		}
		output += u.Host()
	} else if u.host == nil && u.protocol == "file" {
		output += "//"
	}
	output += u.Pathname()

	if u.search != nil {
		output += "?" + *u.search
	}

	if u.hash != nil {
		output += "#" + *u.hash
	}

	return output
}

func (u *Url) Protocol() string {
	return u.protocol + ":"
}

func (u *Url) SetProtocol(scheme string) {
	defaultParser.basicParser(scheme+":", nil, u, stateSchemeStart)
}

func (u *Url) Username() string {
	return u.username
}

func (u *Url) SetUsername(username string) {
	if u.host == nil || *u.host == "" || u.cannotBeABaseUrl || u.protocol == "file" {
		return
	}
	// TODO: percent encode
	u.username = username
}

func (u *Url) Password() string {
	return u.password
}

func (u *Url) SetPassword(password string) {
	if u.host == nil || *u.host == "" || u.cannotBeABaseUrl || u.protocol == "file" {
		return
	}
	// TODO: percent encode
	u.password = password
}

func (u *Url) Host() string {
	if u.host == nil {
		return ""
	}
	if u.port == nil {
		return *u.host
	}
	return *u.host + ":" + *u.port
}

func (u *Url) SetHost(host string) {
	if u.cannotBeABaseUrl {
		return
	}
	defaultParser.basicParser(host, nil, u, stateHost)
}

func (u *Url) Hostname() string {
	if u.host == nil {
		return ""
	}
	return *u.host
}

func (u *Url) SetHostname(host string) {
	if u.cannotBeABaseUrl {
		return
	}
	defaultParser.basicParser(host, nil, u, stateHostname)
}

func (u *Url) Port() string {
	if u.port == nil {
		return ""
	}
	return *u.port
}

func (u *Url) SetPort(port string) {
	if u.host == nil || *u.host == "" || u.cannotBeABaseUrl || u.protocol == "file" {
		return
	}
	if port == "" {
		u.port = nil
	} else {
		defaultParser.basicParser(port, nil, u, statePort)
	}
}

func (u *Url) Pathname() string {
	if u.cannotBeABaseUrl {
		if len(u.path) == 0 {
			return ""
		} else {
			return u.path[0]
		}
	}
	if len(u.path) == 0 {
		return ""
	}
	return "/" + strings.Join(u.path, "/")
}

func (u *Url) SetPathname(path string) {
	if u.cannotBeABaseUrl {
		return
	}
	if u.path != nil {
		u.path = u.path[:0]
	}
	defaultParser.basicParser(path, nil, u, statePathStart)
}

func (u *Url) Search() string {
	if u.search == nil || len(*u.search) == 0 {
		return ""
	}
	return "?" + *u.search
}

func (u *Url) SetSearch(query string) {
	if query == "" {
		u.search = nil
		if u.searchParams != nil {
			u.searchParams.params = u.searchParams.params[:0]
		}
		return
	}
	query = strings.TrimPrefix(query, "?")
	*u.search = ""
	defaultParser.basicParser(query, nil, u, stateQuery)
	u.searchParams.init(query)
}

func (u *Url) SearchParams() *searchParams {
	if u.searchParams == nil {
		u.newUrlSearchParams()
	}
	return u.searchParams
}

func (u *Url) Hash() string {
	if u.hash == nil || len(*u.hash) == 0 {
		return ""
	}
	return "#" + *u.hash
}

func (u *Url) SetHash(fragment string) {
	if fragment == "" {
		u.hash = nil
		return
	}
	fragment = strings.TrimPrefix(fragment, "#")
	*u.hash = ""
	defaultParser.basicParser(fragment, nil, u, stateFragment)
}

func (u *Url) String() string {
	return u.Href()
}

func (u *Url) ValidationErrors() []error {
	return u.validationErrors
}

type nameValuePair struct {
	name, value string
}

type searchParams struct {
	url    *Url
	params []*nameValuePair
}

func (u *Url) newUrlSearchParams() {
	usp := &searchParams{url: u}
	if u.search != nil {
		usp.init(*u.search)
	}
	u.searchParams = usp
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
		output.WriteRune('=')
		output.WriteString(url.QueryEscape(nvp.value))
	}
	return output.String()
}
