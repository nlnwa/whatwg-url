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
	decodedPort      int
	path             *path
	search           *string
	hash             *string
	searchParams     *searchParams
	validationErrors []error
	parser           *parser
	isIPv4           bool
	isIPv6           bool
}

// Href implements WHATWG url api (https://url.spec.whatwg.org/#api)
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
		output += *u.host
		if u.port != nil {
			output += ":" + *u.port
		}
	}
	if u.host == nil && !u.path.isOpaque() && len(u.path.p) > 1 && u.path.p[0] == "" {
		output += "/."
	}

	output += u.path.String()

	if u.search != nil {
		output += "?" + *u.search
	}

	if !excludeFragment && u.hash != nil {
		output += "#" + *u.hash
	}

	return output
}

// Protocol implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Protocol() string {
	return u.protocol + ":"
}

// SetProtocol implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetProtocol(scheme string) {
	if !strings.HasSuffix(scheme, ":") {
		scheme = scheme + ":"
	}
	u.parser.basicParser(scheme, nil, u, stateSchemeStart)
}

func (u *Url) Scheme() string {
	return u.protocol
}

// Username implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Username() string {
	return u.username
}

// SetUsername implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetUsername(username string) {
	if u.host == nil || *u.host == "" || u.protocol == "file" {
		return
	}
	u.username = u.parser.PercentEncodeString(username, UserInfoPercentEncodeSet)
}

// Password implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Password() string {
	return u.password
}

// SetPassword implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPassword(password string) {
	if u.host == nil || *u.host == "" || u.protocol == "file" {
		return
	}
	u.password = u.parser.PercentEncodeString(password, UserInfoPercentEncodeSet)
}

// Host implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Host() string {
	if u.host == nil {
		return ""
	}
	if u.port == nil {
		return *u.host
	}
	return *u.host + ":" + *u.port
}

// SetHost implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHost(host string) {
	if u.path.isOpaque() {
		return
	}
	u.parser.basicParser(host, nil, u, stateHost)
}

// Hostname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Hostname() string {
	if u.host == nil {
		return ""
	}
	return *u.host
}

// SetHostname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHostname(host string) {
	if u.path.isOpaque() {
		return
	}
	u.parser.basicParser(host, nil, u, stateHostname)
}

// Port implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Port() string {
	if u.port == nil {
		return ""
	}
	return *u.port
}

// SetPort implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPort(port string) {
	if u.host == nil || *u.host == "" || u.protocol == "file" {
		return
	}
	if port == "" {
		u.port = nil
	} else {
		u.parser.basicParser(port, nil, u, statePort)
	}
}

func (u *Url) DecodedPort() int {
	if u.decodedPort == 0 {
		return u.getDefaultPort()
	} else {
		return u.decodedPort
	}
}

// Pathname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Pathname() string {
	return u.path.String()
}

// SetPathname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPathname(path string) {
	if u.path.isOpaque() {
		return
	}
	u.path.init()
	u.parser.basicParser(path, nil, u, statePathStart)
}

// Search implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Search() string {
	if u.search == nil || len(*u.search) == 0 {
		return ""
	}
	return "?" + *u.search
}

// SetSearch implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetSearch(query string) {
	if query == "" {
		u.search = nil
		if u.searchParams != nil {
			u.searchParams.params = u.searchParams.params[:0]
		}
		if u.hash == nil && u.search == nil {
			u.path.stripTrailingSpacesIfOpaque()
		}
		return
	}
	query = strings.TrimPrefix(query, "?")
	if u.search == nil {
		u.search = new(string)
	}
	_, _ = u.parser.basicParser(query, nil, u, stateQuery)
	if u.searchParams == nil {
		u.newUrlSearchParams()
	} else {
		u.searchParams.init(*u.search)
	}
}

// SearchParams implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SearchParams() *searchParams {
	if u.searchParams == nil {
		u.newUrlSearchParams()
	}
	return u.searchParams
}

func (u *Url) Query() string {
	if u.search == nil || len(*u.search) == 0 {
		return ""
	}
	return *u.search
}

// Hash implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Hash() string {
	if u.hash == nil || len(*u.hash) == 0 {
		return ""
	}
	return "#" + *u.hash
}

// SetHash implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHash(fragment string) {
	if fragment == "" {
		u.hash = nil
		if u.hash == nil && u.search == nil {
			u.path.stripTrailingSpacesIfOpaque()
		}
		return
	}
	fragment = strings.TrimPrefix(fragment, "#")
	u.hash = new(string)
	u.parser.basicParser(fragment, nil, u, stateFragment)
}

func (u *Url) Fragment() string {
	if u.hash == nil || len(*u.hash) == 0 {
		return ""
	}
	return *u.hash
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

func (u *Url) IsIPv4() bool {
	return u.isIPv4
}

func (u *Url) IsIPv6() bool {
	return u.isIPv6
}
