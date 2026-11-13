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

// Url represents a URL.
type Url struct {
	validationErrors []*ValidationError

	scheme   string
	username string
	password string
	host     string
	query    string
	fragment string

	path         *path
	searchParams *SearchParams
	parser       *parser

	port        uint16
	defaultPort uint16

	hasHost     bool
	hasQuery    bool
	hasFragment bool
	isIPv4      bool
	isIPv6      bool
	isSpecial   bool
	hasPort     bool
}

func newUrl() *Url {
	return &Url{
		path: &path{},
	}
}

func (u *Url) Parse(ref string) (*Url, error) {
	return u.parser.BasicParse(ref, u, nil, NoState)
}

// Href implements WHATWG url api (https://url.spec.whatwg.org/#api)
// If excludeFragment is true, the fragment component will be excluded from the output.
func (u *Url) Href(excludeFragment bool) string {
	var out strings.Builder
	out.Grow(u.hrefLen(excludeFragment))

	// Let output be url’s scheme
	out.WriteString(u.scheme)
	// and U+003A (:) concatenated.
	out.WriteByte(':')
	// If url’s host is non-null:
	if u.hasHost {
		out.WriteString("//")
		// If url includes credentials, then:
		if u.username != "" || u.password != "" {
			// Append url’s username to output.
			if u.username != "" {
				out.WriteString(u.username)
			}
			// If url’s password is not the empty string,
			if u.password != "" {
				// then append U+003A (:),
				out.WriteByte(':')
				// followed by url’s password, to output.
				out.WriteString(u.password)
			}
			// Append U+0040 (@) to output.
			out.WriteByte('@')
		}
		// Append url’s host, serialized, to output.
		out.WriteString(u.host)
		// If url’s port is non-null,
		if u.hasPort {
			// append U+003A (:)
			out.WriteByte(':')
			// followed by url’s port, serialized, to output.
			out.WriteString(u.Port())
		}
	}
	// If url’s host is null, url does not have an opaque path, url’s path’s size is greater than 1, and url’s path[0] is the empty string,
	if !u.hasHost && u.path.needsDisambiguation() {
		// then append U+002F (/) followed by U+002E (.) to output.
		out.WriteString("/.")
		// Note: This prevents web+demo:/.//not-a-host/ or web+demo:/path/..//not-a-host/,
		// when parsed and then serialized, from ending up as web+demo://not-a-host/
		// (they end up as web+demo:/.//not-a-host/).
	}

	// Append the result of URL path serializing url to output.
	out.WriteString(u.path.String())

	// If url’s query is non-null,

	if u.hasQuery {
		// append U+003F (?),
		out.WriteByte('?')
		// followed by url’s query, to output.
		out.WriteString(u.query)
	}
	// If exclude fragment is false and url’s fragment is non-null,
	if !excludeFragment && u.hasFragment {
		// then append U+0023 (#),
		out.WriteByte('#')
		// followed by url’s fragment, to output.
		out.WriteString(u.fragment)
	}
	// Return output.
	return out.String()
}

// Protocol implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Protocol() string {
	// return this’s URL’s scheme, followed by U+003A (:).
	return u.scheme + ":"
}

// SetProtocol implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetProtocol(scheme string) {
	// basic URL parse the given value,
	if !strings.HasSuffix(scheme, ":") {
		// followed by U+003A (:),
		scheme = scheme + ":"
	}
	// with this’s URL as url and scheme start state as state override.
	_, _ = u.parser.BasicParse(scheme, nil, u, StateSchemeStart)
}

func (u *Url) Scheme() string {
	return u.scheme
}

// Username implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Username() string {
	// return this’s URL’s username.
	return u.username
}

// SetUsername implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetUsername(username string) {
	// If this’s URL cannot have a username/password/port, then return.
	if !u.hasHost || u.host == "" || u.scheme == "file" {
		return
	}
	if username == "" {
		u.username = ""
		return
	}
	// Set the username given this’s URL and the given value.
	u.username = PercentEncode(username, UserInfoPercentEncodeSet)
}

// Password implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Password() string {
	// return this’s URL’s password.
	return u.password
}

// SetPassword implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPassword(password string) {
	// If this’s URL cannot have a username/password/port,
	if !u.hasHost || u.host == "" || u.scheme == "file" {
		// then return.
		return
	}
	if password == "" {
		u.password = ""
		return
	}
	// Set the password given this’s URL and the given value.
	u.password = PercentEncode(password, UserInfoPercentEncodeSet)
}

// Host implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Host() string {
	// If url’s host is null,
	if !u.hasHost {
		// then return the empty string.
		return ""
	}
	// If url’s port is null,
	if !u.hasPort {
		// return url’s host, serialized.
		return u.host
	}
	// Return url’s host, serialized, followed by U+003A (:) and url’s port, serialized.
	return u.host + ":" + u.Port()
}

// SetHost implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHost(host string) {
	// If this’s URL has an opaque path,
	if u.path.isOpaque() {
		// then return.
		return
	}
	// Basic URL parse the given value with this’s URL as url and host state as state override.
	_, _ = u.parser.BasicParse(host, nil, u, StateHost)
}

// Hostname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Hostname() string {
	// If this’s URL’s host is null,
	if !u.hasHost {
		// then return the empty string.
		return ""
	}
	// Return this’s URL’s host, serialized.
	return u.host
}

// SetHostname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHostname(host string) {
	// If this’s URL has an opaque path,
	if u.path.isOpaque() {
		// then return.
		return
	}
	// Basic URL parse the given value with this’s URL as url and hostname state as state override.
	_, _ = u.parser.BasicParse(host, nil, u, StateHostname)
}

// Port implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Port() string {
	// If this’s URL’s port is null,
	if !u.hasPort {
		// then return the empty string.
		return ""
	}
	// Return this’s URL’s port, serialized.
	return uitoa16(u.port)
}

func uitoa16(v uint16) string {
	// max 5 digits
	var buf [5]byte
	i := len(buf)
	x := uint32(v)
	for {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
		if x == 0 {
			break
		}
	}
	return string(buf[i:])
}

// SetPort implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPort(port string) {
	// If this’s URL cannot have a username/password/port, then return.
	if !u.hasHost || u.host == "" || u.scheme == "file" {
		return
	}
	// If the given value is the empty string,
	if port == "" {
		// then set this’s URL’s port to null.
		u.hasPort = false
		u.port = 0
		return
	}
	// Otherwise, basic URL parse the given value with this’s URL as url
	// and port state as state override.
	_, _ = u.parser.BasicParse(port, nil, u, StatePort)
}

func (u *Url) DecodedPort() uint16 {
	if u.hasPort {
		return u.port
	}
	if u.isSpecial {
		return u.defaultPort
	}
	return 0
}

// Pathname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Pathname() string {
	// return the result of URL path serializing this’s URL.
	return u.path.String()
}

// SetPathname implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetPathname(path string) {
	// If this’s URL has an opaque path,
	if u.path.isOpaque() {
		// then return
		return
	}
	// Empty this’s URL’s path.
	u.path.init()
	// Basic URL parse the given value with this’s URL as url
	// and path start state as state override.
	_, _ = u.parser.BasicParse(path, nil, u, StatePathStart)
}

// OpaquePath tells if the path is opaque (https://url.spec.whatwg.org/#url-opaque-path)
func (u *Url) OpaquePath() bool {
	return u.path.opaque
}

// Search implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Search() string {
	// If this’s URL’s query is either null or the empty string,
	if !u.hasQuery || len(u.query) == 0 {
		// then return the empty string.
		return ""
	}
	// Return U+003F (?), followed by this’s URL’s query.
	return "?" + u.query
}

// SetSearch implements WHATWG url api (https://url.spec.whatwg.org/#api).
// It parses the supplied query as a URL component and applies the configured
// URL query percent-encoding rules. For form-data semantics use SearchParams;
// to rewrite serialized query terms without decoding use RewriteQuery.
func (u *Url) SetSearch(query string) {
	// Let url be this’s URL.

	// If the given value is the empty string,
	if query == "" {
		// then set url’s query to null,
		u.query = ""
		u.hasQuery = false
		if !u.hasFragment {
			u.path.stripTrailingSpacesIfOpaque()
		}
		u.refreshSearchParams()
		// and return.
		return
	}
	// Let input be the given value with a single leading U+003F (?) removed, if any.
	if query[0] == '?' {
		query = query[1:]
	}
	// Set url’s query to the empty string.
	u.query = ""
	u.hasQuery = false
	// Basic URL parse input with url as url and query state as state override.
	_, _ = u.parser.BasicParse(query, nil, u, StateQuery)
	if query == "" && !u.hasFragment {
		u.path.stripTrailingSpacesIfOpaque()
	}
	u.refreshSearchParams()
}

func (u *Url) refreshSearchParams() {
	if u.searchParams == nil {
		return
	}
	u.searchParams.values.Init(u.query)
}

// SearchParams implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SearchParams() *SearchParams {
	// return this’s query object.
	if u.searchParams == nil {
		var v params
		if u.hasQuery {
			v.Init(u.query)
		}
		u.searchParams = &SearchParams{url: u, values: v}
	}
	return u.searchParams
}

// Query returns the serialized query without a leading '?'. Use ParseQuery
// to obtain lossless terms or SearchParams for decoded form data.
func (u *Url) Query() string {
	if !u.hasQuery || len(u.query) == 0 {
		return ""
	}
	return u.query
}

// Hash implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) Hash() string {
	// If this’s URL’s fragment is either null or the empty string,
	if !u.hasFragment || len(u.fragment) == 0 {
		// then return the empty string.
		return ""
	}
	// Return U+0023 (#), followed by this’s URL’s fragment.
	return "#" + u.fragment
}

// SetHash implements WHATWG url api (https://url.spec.whatwg.org/#api)
func (u *Url) SetHash(fragment string) {
	// If the given value is the empty string,
	if fragment == "" {
		// then set this’s URL’s fragment to null
		u.fragment = ""
		u.hasFragment = false

		// Out-of-spec: Strip potential trailing path spaces if the URL is opaque
		if !u.hasQuery {
			u.path.stripTrailingSpacesIfOpaque()
		}
		// and return.
		return
	}
	// Let input be the given value with a single leading U+0023 (#) removed, if any.
	if fragment[0] == '#' {
		fragment = fragment[1:]
	}
	// Set this’s URL’s fragment to the empty string.
	u.fragment = ""
	u.hasFragment = true
	// Basic URL parse input with this’s URL as url and fragment state as state override.
	_, _ = u.parser.BasicParse(fragment, nil, u, StateFragment)
}

func (u *Url) Fragment() string {
	if !u.hasFragment || len(u.fragment) == 0 {
		return ""
	}
	return u.fragment
}

func (u *Url) String() string {
	return u.Href(false)
}

// ValidationErrors returns a copy of the slice of validation errors associated
// with this URL. The entries refer to the recorded occurrences.
func (u *Url) ValidationErrors() []*ValidationError {
	return append([]*ValidationError(nil), u.validationErrors...)
}

func (u *Url) IsIPv4() bool {
	return u.isIPv4
}

func (u *Url) IsIPv6() bool {
	return u.isIPv6
}

// Clone returns a deep copy of the URL.
func (u *Url) Clone() *Url {
	return &Url{
		validationErrors: append([]*ValidationError(nil), u.validationErrors...),
		scheme:           u.scheme,
		username:         u.username,
		password:         u.password,
		host:             u.host,
		hasHost:          u.hasHost,
		port:             u.port,
		defaultPort:      u.defaultPort,
		hasPort:          u.hasPort,
		path:             u.path.clone(),
		query:            u.query,
		hasQuery:         u.hasQuery,
		fragment:         u.fragment,
		hasFragment:      u.hasFragment,
		searchParams:     nil,
		parser:           u.parser,
		isIPv4:           u.isIPv4,
		isIPv6:           u.isIPv6,
		isSpecial:        u.isSpecial,
	}
}

func classifyDefaultSpecial(scheme string) (port uint16, isSpecial bool) {
	switch scheme {
	case "ftp":
		return 21, true
	case "http", "ws":
		return 80, true
	case "https", "wss":
		return 443, true
	case "file":
		return 0, true
	default:
		return 0, false
	}
}

func (u *Url) IsSpecialScheme() bool {
	return u.isSpecial
}

func (u *Url) normalizePort() {
	if !u.isSpecial || !u.hasPort {
		return
	}
	if u.port == u.defaultPort {
		u.hasPort = false
		u.port = 0
	}
}

func (u *Url) hrefLen(excludeFragment bool) int {
	n := len(u.scheme) + 1 // scheme + ':'

	if u.hasHost {
		n += 2 // "//"
		if u.username != "" || u.password != "" {
			n += len(u.username)
			if u.password != "" {
				n += 1 + len(u.password) // ':' + password
			}
			n += 1 // '@'
		}
		n += len(u.host)
		if u.hasPort {
			n += 1 + portLen(u.port)
		}
	}

	if !u.hasHost && u.path.needsDisambiguation() {
		n += 2 // "/."
	}

	n += u.path.serializedLen()

	if u.hasQuery {
		n += 1 + len(u.query)
	}
	if !excludeFragment && u.hasFragment {
		n += 1 + len(u.fragment)
	}
	return n
}

func portLen(p uint16) int {
	switch {
	case p < 10:
		return 1
	case p < 100:
		return 2
	case p < 1000:
		return 3
	case p < 10000:
		return 4
	default:
		return 5
	}
}
