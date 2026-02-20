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

package internal

import (
	"strings"
	"unicode/utf8"

	"github.com/nationallibraryofnorway/whatwg-url/errors"
)

func NewParser() *Parser {
	return &Parser{
		PathPercentEncodeSet:            PathPercentEncodeSet,
		SpecialQueryPercentEncodeSet:    SpecialQueryPercentEncodeSet,
		QueryPercentEncodeSet:           QueryPercentEncodeSet,
		SpecialFragmentPercentEncodeSet: FragmentPercentEncodeSet,
		FragmentPercentEncodeSet:        FragmentPercentEncodeSet,
		SpecialSchemes: map[string]uint16{
			"ftp":   21,
			"file":  0,
			"http":  80,
			"https": 443,
			"ws":    80,
			"wss":   443,
		},
	}
}

// Parser configures a url parser. Parser are set by the ParserOption
// values passed to NewParser.
type Parser struct {
	SpecialSchemes map[string]uint16

	PreParseHostFunc  func(url *Url, host string) string
	PostParseHostFunc func(url *Url, host string) string

	PathPercentEncodeSet            *PercentEncodeSet
	QueryPercentEncodeSet           *PercentEncodeSet
	FragmentPercentEncodeSet        *PercentEncodeSet
	SpecialQueryPercentEncodeSet    *PercentEncodeSet
	SpecialFragmentPercentEncodeSet *PercentEncodeSet

	ReportValidationErrors              bool
	FailOnValidationError               bool
	LaxHostParsing                      bool
	CollapseConsecutiveSlashes          bool
	AcceptInvalidCodepoints             bool
	PercentEncodeSinglePercentSign      bool
	AllowSettingPathForNonBaseUrl       bool
	SkipWindowsDriveLetterNormalization bool
	SkipTrailingSlashNormalization      bool
	SkipEqualsForEmptySearchParamsValue bool
	HasCustomSpecialSchemes             bool
}

func (p *Parser) Parse(rawUrl string) (*Url, error) {
	return p.parse(rawUrl, nil, nil, NoState)
}

func (p *Parser) ParseRef(rawUrl, ref string) (*Url, error) {
	b, err := p.parse(rawUrl, nil, nil, NoState)
	if err != nil {
		return nil, err
	}
	return b.parser.parse(ref, b, nil, NoState)
}

type State int

const (
	NoState State = iota
	StateSchemeStart
	StateScheme
	StateNoScheme
	StateOpaquePath
	StateSpecialRelativeOrAuthority
	StateSpecialAuthoritySlashes
	StateSpecialAuthorityIgnoreSlashes
	StatePathOrAuthority
	StateAuthority
	StateAuthorityStart
	StateHost
	StateHostname
	StateFile
	StateFileHost
	StateFileSlash
	StatePort
	StatePath
	StatePathStart
	StateQuery
	StateFragment
	StateRelative
	StateRelativeSlash
)

func (p *Parser) BasicParser(urlOrRef string, base *Url, url *Url, stateOverride State) (*Url, error) {
	return p.parse(urlOrRef, base, url, stateOverride)
}

// BasicParser implements WHATWG basic URL parser (https://url.spec.whatwg.org/#concept-basic-url-parser)
// In most cases, when possible, prefer using the higher level Parse method.
func (p *Parser) parse(urlOrRef string, base *Url, url *Url, stateOverride State) (*Url, error) {
	var changed bool

	// If url is not given:
	if url == nil {
		// Set url to a new URL.
		url = &Url{path: &path{}}

		// If input contains any leading or trailing C0 control or space,
		// Remove any leading and trailing C0 control or space from input.
		if urlOrRef, changed = trimC0OrSpace(urlOrRef); changed {
			// invalid-URL-unit validation error.
			if err := p.handleError(url, errors.ErrInvalidURLUnit); err != nil {
				return nil, err
			}
		}
	}

	url.parser = p

	// If input contains any ASCII tab or newline,
	// Remove all ASCII tab or newline from input.
	if urlOrRef, changed = removeASCIITabNewline(urlOrRef); changed {
		// invalid-URL-unit validation error.
		if err := p.handleError(url, errors.ErrInvalidURLUnit); err != nil {
			return nil, err
		}
	}

	// Let state be state override if given, or scheme start state otherwise.
	state := StateSchemeStart
	stateOverridden := stateOverride > NoState
	if stateOverridden {
		state = stateOverride
	}

	// Set encoding to the result of getting an output encoding from encoding.

	// Let buffer be the empty string.
	var buf []byte

	// Let atSignSeen, insideBrackets, and passwordTokenSeen be false.
	atFlag := false
	bracketFlag := false
	passwordTokenSeen := -1

	// Let pointer be a pointer for input.
	var r rune
	var b byte

	const eofRune = rune(-1)

	pos := 0
	size := 0
	strLen := len(urlOrRef)

	var port uint32
	var sawPortDigits bool
	var portOverflow bool

	invalidUtf8 := false
	authorityStart := -1

	for {
		pos += size
		invalidUtf8 = false
		if pos >= strLen {
			r = eofRune
			size = 0
		} else {
			b = urlOrRef[pos]
			if b < utf8.RuneSelf {
				r = rune(b)
				size = 1
			} else {
				r, size = utf8.DecodeRuneInString(urlOrRef[pos:])
				if r == utf8.RuneError && size == 1 {
					r = rune(b)
					invalidUtf8 = true
				}
			}
		}

	reconsume: // reuse previous r and b

		switch state {

		case StateSchemeStart:

			// If c is an ASCII alpha,
			if isASCIIAlpha(r) {
				// append c,
				if 'A' <= b && b <= 'Z' {
					// lowercased,
					b |= 0x20
				}
				// to buffer,
				buf = append(buf, b)

				// and set state to scheme state.
				state = StateScheme
				continue
			}

			// Otherwise, if state override is not given,
			if !stateOverridden {
				// set state to no scheme state
				state = StateNoScheme
				// and decrease pointer by 1
				// reconsume = true
				// continue
				goto reconsume
			}

			// Otherwise, return failure.
			return nil, p.handleErrorAndFail(url, errors.ErrInvalidURLUnit)
			// Note: This indication of failure is used exclusively by the Location object’s protocol setter.

		case StateScheme:

			// If c is an ASCII alphanumeric, U+002B (+), U+002D (-), or U+002E (.),
			if isASCIIAlphanumericPlus(r) {
				// append c, lowercased, to buffer.
				if 'A' <= b && b <= 'Z' {
					b |= 0x20 // to lower case
				}
				buf = append(buf, b)
				continue
			}
			// Otherwise, if c is U+003A (:), then:
			if r == ':' {
				scheme := string(buf)
				buf = buf[:0]

				// candidate scheme classification (do not mutate url yet)
				candDefaultPort, candSpecial := p.classifyScheme(scheme)

				// cache old specialness before any mutation
				oldSpecial := url.isSpecial

				// If state override is given, then:
				if stateOverridden {
					// If url’s scheme is a special scheme and buffer is not a special scheme, then return.
					if oldSpecial && !candSpecial {
						return url, nil
					}
					// If url’s scheme is not a special scheme and buffer is a special scheme, then return.
					if !oldSpecial && candSpecial {
						return url, nil
					}
					// If url includes credentials or has a non-null port, and buffer is "file", then return.
					if (url.username != "" || url.password != "" || url.hasPort) && scheme == "file" {
						return url, nil
					}
					// If url’s scheme is "file" and its host is an empty host or null, then return.
					if url.scheme == "file" && (!url.hasHost || url.host == "") {
						return url, nil
					}
				}

				// Set url’s scheme to buffer.
				url.scheme = scheme
				url.isSpecial = candSpecial
				url.defaultPort = candDefaultPort

				// If state override is given, then:
				if stateOverridden {
					// If url’s port is url’s scheme’s default port, then set url’s port to null.
					url.normalizePort()
					// Return.
					return url, nil
				}

				// If url’s scheme is "file", then:
				if url.scheme == "file" {
					// If remaining does not start with "//", special-scheme-missing-following-solidus validation error.
					nextIdx := pos + size
					if nextIdx+1 >= len(urlOrRef) || urlOrRef[nextIdx] != '/' || urlOrRef[nextIdx+1] != '/' {
						if err := p.handleError(url, errors.ErrSpecialSchemeMissingFollowingSolidus); err != nil {
							return nil, err
						}
					}
					// Set state to file state.
					state = StateFile
					continue
				}
				// Otherwise, if url is special, base is non-null, and base’s scheme is url’s scheme:
				if url.isSpecial && base != nil && base.scheme == url.scheme {
					// Assert: base is special (and therefore does not have an opaque path).
					// Set state to special relative or authority state.
					state = StateSpecialRelativeOrAuthority
					continue
				}
				// Otherwise, if url is special,
				if url.isSpecial {
					// set state to special authority slashes state.
					state = StateSpecialAuthoritySlashes
					continue
				}
				// Otherwise, if remaining starts with an U+002F (/),
				nextIdx := pos + size
				if nextIdx < len(urlOrRef) && urlOrRef[nextIdx] == '/' {
					// set state to path or authority state
					state = StatePathOrAuthority
					// and increase pointer by 1.
					pos = nextIdx
					// _ = input.advanceBytes(1)
					continue
				}
				// Otherwise, set url’s path to the empty string and set state to opaque path state.
				url.path.setOpaque("")
				state = StateOpaquePath
				continue
			}
			// Otherwise, if state override is not given,
			if !stateOverridden {
				// set buffer to the empty string,
				buf = buf[:0]
				// state to no scheme state,
				state = StateNoScheme
				// and start over (from the first code point in input).
				pos = 0
				size = 0
				continue
			}
			// Otherwise, return failure.
			return nil, p.handleErrorAndFail(url, errors.ErrInvalidURLUnit)

		case StateNoScheme:
			// If base is null, or base has an opaque path and c is not U+0023 (#),
			if base == nil || (base.path.isOpaque() && r != '#') {
				// missing-scheme-non-relative-URL validation error, return failure.
				if err := p.handleError(url, errors.ErrMissingSchemeNonRelativeURL); err != nil {
					return nil, err
				}
				continue
			}
			// Otherwise, if base has an opaque path and c is U+0023 (#),
			if base.path.isOpaque() && r == '#' {
				// set url’s scheme to base’s scheme,
				url.scheme = base.scheme
				url.isSpecial = base.isSpecial
				// url’s path to base’s path,
				url.path = base.path
				// url’s query to base’s query,
				url.query = base.query
				url.hasQuery = base.hasQuery
				// url’s fragment to the empty string,
				url.fragment = ""
				url.hasFragment = true
				// and set state to fragment state.
				state = StateFragment
				continue
			}
			// Otherwise, if base’s scheme is not "file",
			if base.scheme != "file" {
				// set state to relative state
				state = StateRelative
				// and decrease pointer by 1.
				goto reconsume
			}
			// Otherwise, set state to file state
			state = StateFile
			// and decrease pointer by 1.
			goto reconsume

		case StateSpecialRelativeOrAuthority:
			// If c is U+002F (/)
			if r == '/' {
				// and remaining starts with U+002F (/),
				nextIdx := pos + size
				if nextIdx < len(urlOrRef) && urlOrRef[nextIdx] == '/' {
					// then set state to special authority ignore slashes state
					state = StateSpecialAuthorityIgnoreSlashes
					// and increase pointer by 1.
					// _ = input.advanceBytes(1)
					pos = nextIdx
					continue
				}
			}
			// Otherwise, special-scheme-missing-following-solidus validation error,
			if err := p.handleError(url, errors.ErrSpecialSchemeMissingFollowingSolidus); err != nil {
				return nil, err
			}
			// set state to relative state
			state = StateRelative
			// and decrease pointer by 1.
			goto reconsume

		case StatePathOrAuthority:
			// If c is U+002F (/),
			if r == '/' {
				// then set state to authority state.
				state = StateAuthorityStart
				continue
			}
			// Otherwise, set state to path state,
			state = StatePath
			// and decrease pointer by 1.
			goto reconsume

		case StateRelative:
			// Assert: base’s scheme is not "file".

			// Set url’s scheme to base’s scheme.
			url.scheme = base.scheme
			url.isSpecial = base.isSpecial

			// If c is U+002F (/),
			if r == '/' {
				// then set state to relative slash state.
				state = StateRelativeSlash
				continue
			}
			// Otherwise, if url is special and c is U+005C (\),
			if url.isSpecial && r == '\\' {
				// invalid-reverse-solidus validation error,
				if err := p.handleError(url, errors.ErrInvalidReverseSolidus); err != nil {
					return nil, err
				}
				// set state to relative slash state.
				state = StateRelativeSlash
				continue
			}
			// Otherwise:

			// Set url’s username to base’s username,
			url.username = base.username
			// url’s password to base’s password,
			url.password = base.password
			// url’s host to base’s host
			url.host = base.host
			url.hasHost = base.hasHost
			// url’s port to base’s port,
			url.hasPort = base.hasPort
			url.port = base.port
			url.defaultPort = base.defaultPort
			// url’s path to a clone of base’s path,
			url.path = base.path.clone()
			// and url’s query to base’s query.
			url.query = base.query
			url.hasQuery = base.hasQuery

			// If c is U+003F (?),
			if r == '?' {
				// then set url’s query to the empty string,
				url.query = ""
				url.hasQuery = true
				// and state to query state.
				state = StateQuery
				continue
			}
			// Otherwise, if c is U+0023 (#),
			if r == '#' {
				// set url’s fragment to the empty string
				url.fragment = ""
				url.hasFragment = true
				// and state to fragment state.
				state = StateFragment
				continue
			}
			// Otherwise, if c is not the EOF code point:
			if r != eofRune {
				// Set url’s query to null.
				url.query = ""
				url.hasQuery = false
				// Shorten url’s path.
				url.path.shortenPath(url.scheme)
				// Set state to path state
				state = StatePath
				// and decrease pointer by 1.
				goto reconsume
			}

		case StateRelativeSlash:
			// If url is special and c is U+002F (/) or U+005C (\), then:
			if url.isSpecial && (r == '/' || r == '\\') {
				// If c is U+005C (\),
				if r == '\\' {
					// invalid-reverse-solidus validation error.
					if err := p.handleError(url, errors.ErrInvalidReverseSolidus); err != nil {
						return nil, err
					}
				}
				// Set state to special authority ignore slashes state.
				state = StateSpecialAuthorityIgnoreSlashes
				continue
			}
			// Otherwise, if c is U+002F (/),
			if r == '/' {
				// then set state to authority state.
				state = StateAuthorityStart
				continue
			}
			// Otherwise, set url’s username to base’s username,
			url.username = base.username
			// url’s password to base’s password,
			url.password = base.password
			// url’s host to base’s host,
			url.host = base.host
			url.hasHost = base.hasHost
			// url’s port to base’s port,
			url.hasPort = base.hasPort
			url.port = base.port
			url.defaultPort = base.defaultPort
			// state to path state,
			state = StatePath
			// and then, decrease pointer by 1.
			goto reconsume

		case StateSpecialAuthoritySlashes:
			// If c is U+002F (/)
			if r == '/' {
				// and remaining starts with U+002F (/),
				nextIdx := pos + size
				if nextIdx < len(urlOrRef) && urlOrRef[nextIdx] == '/' {
					// then set state to special authority ignore slashes state
					state = StateSpecialAuthorityIgnoreSlashes
					// and increase pointer by 1.
					// _ = input.advanceBytes(1)
					pos = nextIdx
					continue
				}
			}
			// Otherwise, special-scheme-missing-following-solidus validation error,
			if err := p.handleError(url, errors.ErrSpecialSchemeMissingFollowingSolidus); err != nil {
				return nil, err
			}
			// set state to special authority ignore slashes state
			state = StateSpecialAuthorityIgnoreSlashes
			// and decrease pointer by 1.
			goto reconsume

		case StateSpecialAuthorityIgnoreSlashes:

			// If c is neither U+002F (/) nor U+005C (\),
			if r != '/' && r != '\\' {
				// then set state to authority state
				state = StateAuthorityStart
				// and decrease pointer by 1.
				goto reconsume
			}
			// Otherwise, special-scheme-missing-following-solidus validation error.
			if err := p.handleError(url, errors.ErrSpecialSchemeMissingFollowingSolidus); err != nil {
				return nil, err
			}

		case StateAuthorityStart:
			authorityStart = pos
			state = StateAuthority
			fallthrough
		case StateAuthority:
			// If c is U+0040 (@), then:
			if r == '@' {
				// Invalid-credentials validation error.
				if err := p.handleError(url, errors.ErrInvalidCredentials); err != nil {
					return nil, err
				}
				// Let authorityChunk be the substring from authorityStart up to this '@'
				authority := urlOrRef[authorityStart:pos]

				// If atSignSeen is true,
				if atFlag {
					// then prepend "%40" to buffer.
					buf = append(buf, "%40"...)
				}

				// For each code point in authority: percent-encode into userinfo
				buf = appendUserinfoSubstring(buf, authority, &passwordTokenSeen)

				// Set atSignSeen to true.
				atFlag = true

				authorityStart = pos + 1 // position after last '@'

				// Set buffer to the empty string.
			} else
			// Otherwise, if one of the following is true:
			// - c is the EOF code point, U+002F (/), U+003F (?), or U+0023 (#)
			// - url is special and c is U+005C (\)
			if (r == eofRune || r == '/' || r == '?' || r == '#') ||
				(url.isSpecial && r == '\\') {
				// If atSignSeen is true and buffer is the empty string,
				if atFlag && authorityStart == pos {
					// host-missing validation error, return failure.
					if err := p.handleError(url, errors.ErrHostMissing); err != nil {
						return nil, err
					}
				}
				// Decrease pointer by buffer’s code point length + 1,
				pos = authorityStart - 1
				size = 1

				// set buffer to the empty string,
				if passwordTokenSeen >= 0 {
					url.username = string(buf[:passwordTokenSeen])
					url.password = string(buf[passwordTokenSeen:])
					passwordTokenSeen = -1
				} else if len(buf) > 0 {
					url.username = string(buf)
				}
				buf = buf[:0]

				// and set state to host state.
				state = StateHost
			}
			// Otherwise, append c to buffer.
			continue

		case StateHost:
			fallthrough
		case StateHostname:
			// If state override is given and url’s scheme is "file",
			if stateOverridden && url.scheme == "file" {
				// then decrease pointer by 1
				// and set state to file host state.
				state = StateFileHost
				goto reconsume
			}
			// Otherwise, if c is U+003A (:) and insideBrackets is false:
			if r == ':' && !bracketFlag {
				// If buffer is the empty string,
				if len(buf) == 0 {
					// host-missing validation error, return failure.
					if err := p.handleError(url, errors.ErrHostMissing); err != nil {
						return nil, err
					}
				}
				// If state override is given and state override is hostname state,
				if stateOverride == StateHostname {
					// then return failure.
					return nil, p.handleErrorAndFail(url, errors.ErrInvalidURLUnit)
				}
				// Let host be the result of host parsing buffer with url is not special.
				host, err := p.parseHost(url, buf, !url.isSpecial)
				if err != nil {
					// If host is failure, then return failure.
					return url, err
				}
				// Set url’s host to host,
				url.host = host
				url.hasHost = true
				// buffer to the empty string,
				buf = buf[:0]
				// and state to port state.
				state = StatePort
				continue
			}
			// Otherwise, if one of the following is true:
			// - c is the EOF code point, U+002F (/), U+003F (?), or U+0023 (#)
			// - url is special and c is U+005C (\)
			if r == eofRune || r == '/' || r == '?' || r == '#' || (url.isSpecial && r == '\\') {
				// then decrease pointer by 1, and:

				// If url is special and buffer is the empty string,
				if url.isSpecial && len(buf) == 0 {
					// host-missing validation error, return failure.
					if err := p.handleError(url, errors.ErrHostMissing); err != nil {
						return nil, err
					}
					goto reconsume
				}
				// Otherwise, if state override is given, buffer is the empty string,
				// and either url includes credentials or url’s port is non-null,
				if stateOverridden && len(buf) == 0 && (url.username != "" || url.password != "" || url.hasPort) {
					// then return failure.
					return url, nil
				}
				// Let host be the result of host parsing buffer with url is not special.
				host, err := p.parseHost(url, buf, !url.isSpecial)
				if err != nil {
					// If host is failure, then return failure.
					return url, err
				}
				// Set url’s host to host,
				url.host = host
				url.hasHost = true
				// buffer to the empty string,
				buf = buf[:0]
				// and state to path start state.
				state = StatePathStart

				// If state override is given, then return.
				if stateOverridden {
					return url, nil
				}

				goto reconsume
			}

			// Otherwise:
			switch r {
			case '[':
				// If c is U+005B ([), then set insideBrackets to true.
				bracketFlag = true
			case ']':
				// If c is U+005D (]), then set insideBrackets to false.
				bracketFlag = false
			}

			// Append c to buffer.
			if r < utf8.RuneSelf || (invalidUtf8 && p.AcceptInvalidCodepoints) {
				buf = append(buf, b)
			} else {
				buf = utf8.AppendRune(buf, r)
			}
			continue

		case StatePort:
			// If c is an ASCII digit,
			if '0' <= r && r <= '9' { // drop ASCIIDigit bitset
				// append c to buffer.
				sawPortDigits = true
				if !portOverflow {
					port = port*10 + uint32(r-'0')
					if port > 65535 {
						portOverflow = true
					}
				}
				continue
			}
			// Otherwise, if one of the following is true:
			// - c is the EOF code point, U+002F (/), U+003F (?), or U+0023 (#);
			// - url is special and c is U+005C (\); or
			// - state override is given
			if (r == eofRune || r == '/' || r == '?' || r == '#') || (url.isSpecial && r == '\\') || stateOverridden {
				// If buffer is not the empty string:
				if sawPortDigits {
					if portOverflow {
						// If port is not a 16-bit unsigned integer, port-out-of-range validation error, return failure.
						if err := p.handleError(url, errors.ErrPortOutOfRange); err != nil {
							return nil, err
						}
					} else {
						// Let port be the mathematical integer value that is represented by buffer in radix-10 using ASCII digits for digits with values 0 through 9.
						p16 := uint16(port)

						// Set url’s port to null, if port is url’s scheme’s default port; otherwise to port.
						url.hasPort = true
						url.port = p16
						url.normalizePort()
					}

					// Set buffer to the empty string.
					port = 0
					sawPortDigits = false
					portOverflow = false

					// If state override is given, then return.
					if stateOverridden {
						return url, nil
					}
				}
				// If state override is given,
				if stateOverridden {
					// then return failure.
					if err := p.handleError(url, errors.ErrPortMissing); err != nil {
						return nil, err
					}
				}
				// Set state to path start state
				state = StatePathStart
				// and decrease pointer by 1.
				goto reconsume
			}

			// Otherwise, port-invalid validation error, return failure.
			if err := p.handleError(url, errors.ErrPortInvalid); err != nil {
				return nil, err
			}

		case StateFile:
			// Set url’s scheme to "file".
			url.scheme = "file"
			url.isSpecial = true
			// Set url’s host to the empty string.
			url.host = ""
			url.hasHost = true

			// If c is U+002F (/) or U+005C (\), then:
			if r == '/' || r == '\\' {
				// If c is U+005C (\),
				if r == '\\' {
					// invalid-reverse-solidus validation error.
					if err := p.handleError(url, errors.ErrInvalidReverseSolidus); err != nil {
						return nil, err
					}
				}
				// Set state to file slash state.
				state = StateFileSlash
				continue
			}

			// Otherwise, if base is non-null and base’s scheme is "file":
			if base != nil && base.scheme == "file" {
				// Set url’s host to base’s host,
				url.host = base.host
				url.hasHost = base.hasHost
				// url’s path to a clone of base’s path,
				url.path = base.path.clone()
				// and url’s query to base’s query.
				url.query = base.query
				url.hasQuery = base.hasQuery

				//If c is U+003F (?),
				if r == '?' {
					// then set url’s query to the empty string
					url.query = ""
					// and state to query state.
					state = StateQuery
					continue
				}
				// Otherwise, if c is U+0023 (#),
				if r == '#' {
					// set url’s fragment to the empty string
					url.fragment = ""
					url.hasFragment = true
					// and state to fragment state.
					state = StateFragment
					continue
				}
				// Otherwise, if c is not the EOF code point:
				if r != eofRune {
					// Set url’s query to null.
					url.query = ""
					url.hasQuery = false
					// If the code point substring from pointer to the end of input does not start with a Windows drive letter,
					if !startsWithAWindowsDriveLetter(urlOrRef[pos:]) {
						// then shorten url’s path.
						url.path.shortenPath(url.scheme)

					} else { // Otherwise:
						// File-invalid-Windows-drive-letter validation error.
						if err := p.handleError(url, errors.ErrFileInvalidWindowsDriveLetter); err != nil {
							return nil, err
						}
						// Set url’s path to « ».
						url.path.init()
					}
					// Set state to path state
					state = StatePath
					// and decrease pointer by 1.
					goto reconsume
				}
				break // break in case of EOF
			}

			// Otherwise,
			// set state to path state,
			state = StatePath
			// and decrease pointer by 1.
			goto reconsume

		case StateFileSlash:
			// If c is U+002F (/) or U+005C (\), then:
			if r == '/' || r == '\\' {
				// If c is U+005C (\),
				if r == '\\' {
					// invalid-reverse-solidus validation error.
					if err := p.handleError(url, errors.ErrInvalidReverseSolidus); err != nil {
						return nil, err
					}
				}
				// Set state to file host state.
				state = StateFileHost
				continue
			}

			// Otherwise:

			// If base is non-null and base’s scheme is "file", then:
			if base != nil && base.scheme == "file" {
				// Set url’s host to base’s host.
				url.host = base.host
				url.hasHost = base.hasHost
				// If the code point substring from pointer to the end of input does not start with a Windows drive letter
				// and base’s path[0] is a normalized Windows drive letter,
				if !startsWithAWindowsDriveLetter(urlOrRef[pos:]) && base.path != nil && isNormalizedWindowsDriveLetter(base.path.p[0]) {
					// then append base’s path[0] to url’s path.
					url.path.addSegment(base.path.p[0])
					// Note: This is a (platform-independent) Windows drive letter quirk. Both url’s and base’s host are null under these conditions and therefore not copied
				}
			}

			// Set state to path state,
			state = StatePath
			// and decrease pointer by 1.
			goto reconsume

		case StateFileHost:
			// If c is the EOF code point, U+002F (/), U+005C (\), U+003F (?), or U+0023 (#),
			if r == eofRune || r == '/' || r == '\\' || r == '?' || r == '#' {
				// then decrease pointer by 1 and then:

				// If state override is not given and buffer is a Windows drive letter,
				if !stateOverridden && isWindowsDriveLetter(string(buf)) {
					// file-invalid-Windows-drive-letter-host validation error,
					if err := p.handleError(url, errors.ErrFileInvalidWindowsDriveLetterHost); err != nil {
						return nil, err
					}
					// set state to path state.
					state = StatePath
					goto reconsume
				}
				// Otherwise, if buffer is the empty string, then:
				if len(buf) == 0 {
					// Set url’s host to the empty string.
					url.host = ""
					url.hasHost = true
					// If state override is given, then return.
					if stateOverridden {
						return nil, nil
					}
					// Set state to path start state.
					state = StatePathStart
					goto reconsume
				}
				// Otherwise, run these steps:

				// Let host be the result of host parsing buffer with url is not special.
				host, err := p.parseHost(url, buf, !url.isSpecial)
				if err != nil {
					// If host is failure, then return failure.
					return url, err
				}
				// If host is "localhost",
				if host == "localhost" {
					// then set host to the empty string.
					host = ""
				}
				// Set url’s host to host.
				url.host = host
				url.hasHost = true
				// If state override is given, then return.
				if stateOverridden {
					return url, nil
				}
				// Set buffer to the empty string
				buf = buf[:0]
				// and state to path start state.
				state = StatePathStart
				goto reconsume
			}
			// Otherwise, append c to buffer.
			if r < utf8.RuneSelf {
				buf = append(buf, b)
			} else {
				buf = utf8.AppendRune(buf, r)
			}

		case StatePathStart:
			// If url is special, then:
			if url.isSpecial && !p.SkipTrailingSlashNormalization {
				// If c is U+005C (\),
				if r == '\\' {
					// invalid-reverse-solidus validation error.
					if err := p.handleError(url, errors.ErrInvalidReverseSolidus); err != nil {
						return nil, err
					}
				}
				// Set state to path state.
				state = StatePath

				// If c is neither U+002F (/) nor U+005C (\),
				if r != '/' && r != '\\' {
					// then decrease pointer by 1.
					goto reconsume
				}
				continue
			}
			// Otherwise, if state override is not given and c is U+003F (?),
			if !stateOverridden && r == '?' {
				// set url’s query to the empty string
				url.query = ""
				url.hasQuery = false
				// and state to query state.
				state = StateQuery
				continue
			}
			// Otherwise, if state override is not given and c is U+0023 (#),
			if !stateOverridden && r == '#' {
				// set url’s fragment to the empty string
				url.fragment = ""
				url.hasFragment = true
				// and state to fragment state.
				state = StateFragment
				continue
			}
			// Otherwise, if c is not the EOF code point:
			if r != eofRune {
				// Set state to path state.
				state = StatePath
				// If c is not U+002F (/),
				if r != '/' {
					// then decrease pointer by 1.
					goto reconsume
				}
				continue
			}
			// Otherwise, if state override is given and url’s host is null,
			if stateOverridden && !url.hasHost {
				// append the empty string to url’s path.
				url.path.addSegment("")
			}

		case StatePath:

			// If one of the following is true:
			// - c is the EOF code point or U+002F (/)
			if (r == eofRune || r == '/') ||
				// - url is special and c is U+005C (\)
				(url.isSpecial && r == '\\') ||
				// - state override is not given and c is U+003F (?) or U+0023 (#)
				(!stateOverridden && (r == '?' || r == '#')) {

				// If url is special and c is U+005C (\)
				if url.isSpecial && r == '\\' {
					// , invalid-reverse-solidus validation error.
					if err := p.handleError(url, errors.ErrInvalidReverseSolidus); err != nil {
						return nil, err
					}
				}

				// If buffer is a double-dot URL path segment, then:
				if isDoubleDotPathSegmentBytes(buf) {
					// Shorten url’s path.
					url.path.shortenPath(url.scheme)
					// If neither c is U+002F (/), nor url is special and c is U+005C (\),
					if r != '/' && !(url.isSpecial && r == '\\') {
						// append the empty string to url’s path.
						url.path.addSegment("")
						// Note: This means that for input /usr/.. the result is / and not a lack of a path.
					}

				} else if
				// Otherwise, if buffer is a single-dot URL path segment
				isSingleDotPathSegmentBytes(buf) &&
					// and if neither c is U+002F (/), nor url is special and c is U+005C (\),
					r != '/' && !(url.isSpecial && r == '\\') {
					// append the empty string to url’s path.
					url.path.addSegment("")
				} else if
				// Otherwise, if buffer is not a single-dot URL path segment, then:
				!isSingleDotPathSegmentBytes(buf) {
					// If url’s scheme is "file", url’s path is empty, and
					if url.scheme == "file" && url.path.isEmpty() &&
						// buffer is a Windows drive letter,
						isWindowsDriveLetterASCIIBytes(buf) && !p.SkipWindowsDriveLetterNormalization {
						// then replace the second code point in buffer with U+003A (:).
						buf[1] = ':'
						// Note: This is a (platform-independent) Windows drive letter quirk.
					}

					pathSegment := string(buf)

					// Append buffer to url’s path.
					if !p.CollapseConsecutiveSlashes || !url.isSpecial || url.path.isEmpty() || len(url.path.p[len(url.path.p)-1]) > 0 {
						url.path.addSegment(pathSegment)
					} else {
						url.path.p[len(url.path.p)-1] = pathSegment
					}
				}

				// Set buffer to the empty string.
				buf = buf[:0]

				// If c is U+003F (?),
				if r == '?' {
					// then set url’s query to the empty string
					url.query = ""
					url.hasQuery = true
					// and state to query state.
					state = StateQuery
					continue
				}
				// If c is U+0023 (#),
				if r == '#' {
					// then set url’s fragment to the empty string
					url.fragment = ""
					url.hasFragment = true
					// and state to fragment state.
					state = StateFragment
					continue
				}

				break
			}
			// Otherwise, run these steps:

			// If c is not a URL code point and not U+0025 (%),
			if !isURLCodePoint(r) && r != '%' {
				// invalid-URL-unit validation error.
				if err := p.handleError(url, errors.ErrInvalidURLUnit); err != nil {
					return nil, err
				}
			}
			// If c is U+0025 (%)
			if r == '%' {
				// and remaining does not start with two ASCII hex digits,
				invalidPercentEncoding, d := remainingIsInvalidPercentEncoded(urlOrRef[pos:])
				if invalidPercentEncoding {
					// invalid-URL-unit validation error.
					if err := p.handleErrorWithDescription(url, errors.ErrInvalidURLUnit, d); err != nil {
						return nil, err
					}
					if p.PercentEncodeSinglePercentSign {
						buf = append(buf, percentPercentEncoded...)
						break // to avoid being appended again below
					}
				}
			}
			// UTF-8 percent-encode c using the path percent-encode set and append the result to buffer.
			buf = appendPercentEncoded(buf, r, p.PathPercentEncodeSet)

		case StateOpaquePath:
			// If c is U+003F (?),
			if r == '?' {
				// then set url’s query to the empty string
				url.query = ""
				url.hasQuery = false
				// and state to query state.
				state = StateQuery
			} else
			// Otherwise, if c is U+0023 (#),
			if r == '#' {
				// then set url’s fragment to the empty string
				url.fragment = ""
				url.hasFragment = true
				// and state to fragment state.
				state = StateFragment
			} else
			// Otherwise, if c is U+0020 SPACE:
			if r == ' ' {
				// If remaining starts with U+003F (?) or U+003F (#),
				nextIdx := pos + size
				if nextIdx < len(urlOrRef) && (urlOrRef[nextIdx] == '?' || urlOrRef[nextIdx] == '#') {
					// then append "%20" to url’s path.
					buf = append(buf, spacePercentEncoded...)
				} else {
					// Otherwise, append U+0020 SPACE to url’s path.
					buf = append(buf, ' ')
				}
				continue
			} else
			// Otherwise, if c is not the EOF code point:
			if r != eofRune {
				// If c is not a URL code point and not U+0025 (%),
				if !isURLCodePoint(r) && r != '%' {
					// invalid-URL-unit validation error.
					if err := p.handleError(url, errors.ErrInvalidURLUnit); err != nil {
						return nil, err
					}
				}
				// If c is U+0025 (%)
				if r == '%' {
					// and remaining does not start with two ASCII hex digits,
					invalidPercentEncoding, d := remainingIsInvalidPercentEncoded(urlOrRef[pos:])
					if invalidPercentEncoding {
						// invalid-URL-unit validation error.
						if err := p.handleErrorWithDescription(url, errors.ErrInvalidURLUnit, d); err != nil {
							return nil, err
						}
						if p.PercentEncodeSinglePercentSign {
							buf = append(buf, percentPercentEncoded...)
							continue
						}
					}
				}

				// UTF-8 percent-encode c using the C0 control percent-encode set
				// and append the result to url’s path.
				buf = append(buf, percentEncode(r, C0PercentEncodeSet)...)
				continue
			}
			if len(buf) > 0 {
				url.path.p[0] = string(buf)
				buf = buf[:0]
			}

		case StateQuery:
			// If encoding is not UTF-8 and one of the following is true:
			// - url is not special
			// - url’s scheme is "ws" or "wss"
			// then set encoding to UTF-8.

			// If one of the following is true:
			// - state override is not given and c is U+0023 (#)
			// - c is the EOF code point
			if (!stateOverridden && r == '#') || r == eofRune {
				// Let queryPercentEncodeSet be the special-query percent-encodeset if url is special;
				// otherwise the query percent-encode set.

				// !! percent encoding is done per code point below

				// Percent-encode after encoding, with encoding, buffer, and queryPercentEncodeSet,
				// and append the result to url’s query.
				// !! percent encoding is done per code point below
				url.query = string(buf)
				url.hasQuery = true

				// Note: This operation cannot be invoked code-point-for-code-point due to the stateful ISO-2022-JP encoder.

				// Set buffer to the empty string.
				buf = buf[:0]

				// If c is U+0023 (#),
				if r == '#' {
					// then set url’s fragment to the empty string
					url.fragment = ""
					url.hasFragment = true
					// and state to fragment state.
					state = StateFragment
					continue
				}
			} else if
			// Otherwise, if c is not the EOF code point:
			r != eofRune {
				// If c is not a URL code point and not U+0025 (%),
				if !isURLCodePoint(r) && r != '%' {
					// invalid-URL-unit validation error.
					if err := p.handleError(url, errors.ErrInvalidURLUnit); err != nil {
						return nil, err
					}
				}

				// If c is U+0025 (%)
				if r == '%' {
					// and remaining does not start with two ASCII hex digits,
					invalidPercentEncoding, d := remainingIsInvalidPercentEncoded(urlOrRef[pos:])
					if invalidPercentEncoding {
						// invalid-URL-unit validation error.
						if err := p.handleErrorWithDescription(url, errors.ErrInvalidURLUnit, d); err != nil {
							return nil, err
						}
						if p.PercentEncodeSinglePercentSign {
							buf = append(buf, percentPercentEncoded...)
							continue // to not append again below
						}
					}
				}
				// !! percent encode per code point (done for the whole buffer in spec)
				// Append c to buffer.
				if url.isSpecial {
					buf = appendPercentEncoded(buf, r, p.SpecialQueryPercentEncodeSet)
				} else {
					buf = appendPercentEncoded(buf, r, p.QueryPercentEncodeSet)
				}
			}

		case StateFragment:
			// If c is not the EOF code point, then:
			if r != eofRune {
				// If c is not a URL code point and not U+0025 (%),
				if !isURLCodePoint(r) && r != '%' {
					// invalid-URL-unit validation error.
					if err := p.handleError(url, errors.ErrInvalidURLUnit); err != nil {
						return nil, err
					}
				}
				// If c is U+0025 (%)
				if r == '%' {
					// and remaining does not start with two ASCII hex digits,
					invalidPercentEncoding, d := remainingIsInvalidPercentEncoded(urlOrRef[pos:])
					if invalidPercentEncoding {
						// invalid-URL-unit validation error.
						if err := p.handleErrorWithDescription(url, errors.ErrInvalidURLUnit, d); err != nil {
							return nil, err
						}
						if p.PercentEncodeSinglePercentSign {
							buf = append(buf, percentPercentEncoded...)
							continue // to avoid being appended again below
						}
					}
				}

				// UTF-8 percent-encode c using the fragment percent-encode set and append the result to url’s fragment.
				if url.isSpecial {
					buf = appendPercentEncoded(buf, r, p.SpecialFragmentPercentEncodeSet)
				} else {
					buf = appendPercentEncoded(buf, r, p.FragmentPercentEncodeSet)
				}
				continue
			}

			url.fragment = string(buf)
			url.hasFragment = true
			buf = buf[:0]
		}

		if r == eofRune {
			break
		}
	}

	return url, nil
}

func (p *Parser) NewUrl() *Url {
	u := Url{}
	u.parser = p

	u.path = &path{}
	u.path.init()

	return &u
}

// appendUserinfoSubstring takes a UTF-8 substring as userinfo input
// and append each rune into consumeUserinfoCodePoint.
// Important behavior:
// - ':' switches from username to password once (first ':' only).
// - '@' occurring as *data* inside userinfo MUST become "%40" (lossless).
// - All other code points are UTF-8 percent-encoded using UserInfoPercentEncodeSet.
func appendUserinfoSubstring(
	buf []byte,
	s string,
	passwordTokenSeen *int,
) []byte {
	// For each codePoint in buffer:
	for _, r := range s {
		// If codePoint is U+003A (:) and passwordTokenSeen is false,
		if r == ':' && *passwordTokenSeen == -1 {
			// then set passwordTokenSeen to true
			*passwordTokenSeen = len(buf)
			// and continue.
			continue
		}
		// If codePoint is '@' as data, encode it explicitly.
		// This matches the spec behavior where earlier '@' are reclassified as data by prepending "%40".
		if r == '@' {
			buf = append(buf, "%40"...)
		}
		// Let encodedCodePoints be the result of running UTF-8 percent-encode codePoint using the userinfo percent-encode set.
		enc := percentEncode(r, UserInfoPercentEncodeSet)

		buf = append(buf, enc...)
	}
	return buf
}

func (p *Parser) classifyScheme(scheme string) (uint16, bool) {
	if p.HasCustomSpecialSchemes {
		p, ok := p.SpecialSchemes[scheme]
		return p, ok
	}
	return classifyDefaultSpecial(scheme)
}

func isSingleDotPathSegmentBytes(b []byte) bool {
	// "."
	if len(b) == 1 && b[0] == '.' {
		return true
	}
	// "%2e" / "%2E"
	return len(b) == 3 &&
		b[0] == '%' &&
		b[1] == '2' &&
		(b[2] == 'e' || b[2] == 'E')
}

func isDoubleDotPathSegmentBytes(b []byte) bool {
	// ".."
	if len(b) == 2 && b[0] == '.' && b[1] == '.' {
		return true
	}
	// ".%2e"
	if len(b) == 4 &&
		b[0] == '.' &&
		b[1] == '%' &&
		b[2] == '2' &&
		(b[3] == 'e' || b[3] == 'E') {
		return true
	}
	// "%2e."
	if len(b) == 4 &&
		b[0] == '%' &&
		b[1] == '2' &&
		(b[2] == 'e' || b[2] == 'E') &&
		b[3] == '.' {
		return true
	}
	// "%2e%2e"
	if len(b) == 6 &&
		b[0] == '%' &&
		b[1] == '2' &&
		(b[2] == 'e' || b[2] == 'E') &&
		b[3] == '%' &&
		b[4] == '2' &&
		(b[5] == 'e' || b[5] == 'E') {
		return true
	}
	return false
}

func startsWithAWindowsDriveLetter(s string) bool {
	if len(s) >= 2 && isWindowsDriveLetter(s[0:2]) &&
		(len(s) == 2 || s[2] == '/' || s[2] == '\\' || s[2] == '?' || s[2] == '#') {
		return true
	}

	return false
}

func isWindowsDriveLetter(s string) bool {
	return len(s) == 2 && isASCIIAlpha(s[0]) && (s[1] == ':' || s[1] == '|')
}

func isWindowsDriveLetterASCIIBytes(b []byte) bool {
	return len(b) == 2 && isASCIIAlpha(b[0]) && (b[1] == ':' || b[1] == '|')
}

func isNormalizedWindowsDriveLetter(s string) bool {
	return len(s) == 2 && isASCIIAlpha(s[0]) && s[1] == ':'
}

func trimC0OrSpace(s string) (string, bool) {
	if s == "" {
		return s, false
	}

	start := 0
	for start < len(s) && s[start] <= 0x20 {
		start++
	}
	end := len(s)
	for end > start && s[end-1] <= 0x20 {
		end--
	}

	if start == 0 && end == len(s) {
		return s, false
	}
	return s[start:end], true
}

func removeASCIITabNewline(s string) (string, bool) {
	first := strings.IndexAny(s, "\t\n\r")
	if first == -1 {
		return s, false
	}

	var b strings.Builder
	b.Grow(len(s) - 1) // Pre-allocate the maximum possible size
	b.WriteString(s[:first])

	for i := first + 1; i < len(s); i++ {
		c := s[i]
		if c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		b.WriteByte(c)
	}

	return b.String(), true
}
