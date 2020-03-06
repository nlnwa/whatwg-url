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
	parser           *Parser
}

func (u *Url) Href(excludeFragment bool) string {
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

	if !excludeFragment && u.hash != nil {
		output += "#" + *u.hash
	}

	return output
}

func (u *Url) Protocol() string {
	return u.protocol + ":"
}

func (u *Url) SetProtocol(scheme string) {
	u.parser.basicParser(scheme+":", nil, u, stateSchemeStart)
}

func (u *Url) Username() string {
	return u.username
}

func (u *Url) SetUsername(username string) {
	if u.host == nil || *u.host == "" || u.cannotBeABaseUrl || u.protocol == "file" {
		return
	}
	u.username = u.parser.PercentEncodeString(username, UserInfoPercentEncodeSet)
}

func (u *Url) Password() string {
	return u.password
}

func (u *Url) SetPassword(password string) {
	if u.host == nil || *u.host == "" || u.cannotBeABaseUrl || u.protocol == "file" {
		return
	}
	u.password = u.parser.PercentEncodeString(password, UserInfoPercentEncodeSet)
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
	u.parser.basicParser(host, nil, u, stateHost)
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
	u.parser.basicParser(host, nil, u, stateHostname)
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
		u.parser.basicParser(port, nil, u, statePort)
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
	if u.cannotBeABaseUrl && !u.parser.AllowSettingPathForNonBaseUrl {
		return
	}
	if u.path != nil {
		u.path = u.path[:0]
	}
	if u.cannotBeABaseUrl {
		u.parser.basicParser(path, nil, u, stateCannotBeABaseUrl)
	} else {
		u.parser.basicParser(path, nil, u, statePathStart)
	}
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
	if u.search == nil {
		u.search = new(string)
	}
	*u.search = ""
	_, _ = u.parser.basicParser(query, nil, u, stateQuery)
	if u.searchParams == nil {
		u.newUrlSearchParams()
	} else {
		u.searchParams.init(*u.search)
	}
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
	u.hash = new(string)
	u.parser.basicParser(fragment, nil, u, stateFragment)
}

func (u *Url) String() string {
	return u.Href(false)
}

func (u *Url) ValidationErrors() []error {
	return u.validationErrors
}

func (u *Url) newUrlSearchParams() {
	usp := &searchParams{url: u}
	if u.search != nil {
		usp.init(*u.search)
	}
	u.searchParams = usp
}
