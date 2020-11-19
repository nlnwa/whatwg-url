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
	"github.com/nlnwa/whatwg-url/errors"
	"github.com/willf/bitset"
	u2 "net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func NewParser(opts ...ParserOption) Parser {
	p := &parser{opts: defaultParserOptions()}
	for _, opt := range opts {
		opt.apply(&p.opts)
	}
	return p
}

type Parser interface {
	Parse(rawUrl string) (*Url, error)
	ParseRef(rawUrl, ref string) (*Url, error)
}

type parser struct {
	opts parserOptions
}

func (p *parser) Parse(rawUrl string) (*Url, error) {
	return p.basicParser(rawUrl, nil, nil, noState)
}

func (p *parser) ParseRef(rawUrl, ref string) (*Url, error) {
	b, err := p.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	return p.basicParser(ref, b, nil, noState)
}

func (u *Url) Parse(ref string) (*Url, error) {
	return u.parser.basicParser(ref, u, nil, noState)
}

var defaultParser = NewParser()

func Parse(rawUrl string) (*Url, error) {
	return defaultParser.Parse(rawUrl)
}

func ParseRef(rawUrl, ref string) (*Url, error) {
	return defaultParser.ParseRef(rawUrl, ref)
}

type state int

const (
	noState state = iota
	stateSchemeStart
	stateScheme
	stateNoScheme
	stateCannotBeABaseUrl
	stateSpecialRelativeOrAuthority
	stateSpecialAuthoritySlashes
	stateSpecialAuthorityIgnoreSlashes
	statePathOrAuthority
	stateAuthority
	stateHost
	stateHostname
	stateFile
	stateFileHost
	stateFileSlash
	statePort
	statePath
	statePathStart
	stateQuery
	stateFragment
	stateRelative
	stateRelativeSlash
)

func (p *parser) basicParser(urlOrRef string, base *Url, url *Url, stateOverride state) (*Url, error) {
	stateOverridden := stateOverride > noState
	if url == nil {
		url = &Url{inputUrl: urlOrRef}
		if i, changed := trim(url.inputUrl, C0OrSpacePercentEncodeSet); changed {
			if err := p.handleError(url, errors.IllegalLeadingOrTrailingChar); err != nil {
				return nil, err
			}
			url.inputUrl = i
		}
	} else {
		url.inputUrl = urlOrRef
	}
	url.parser = p

	if i, changed := remove(url.inputUrl, ASCIITabOrNewline); changed {
		if err := p.handleError(url, errors.IllegalTabOrNewline); err != nil {
			return nil, err
		}
		url.inputUrl = i
	}

	input := newInputString(url.inputUrl)
	var state state
	if stateOverridden {
		state = stateOverride
	} else {
		state = stateSchemeStart
	}

	var buffer strings.Builder
	atFlag := false
	bracketFlag := false
	passwordTokenSeenFlag := false

	for {
		r := input.nextCodePoint()

		switch state {
		case stateSchemeStart:
			if ASCIIAlpha.Test(uint(r)) {
				buffer.WriteRune(unicode.ToLower(r))
				state = stateScheme
			} else if !stateOverridden {
				state = stateNoScheme
				input.rewindLast()
			} else {
				return p.handleFailure(url, errors.FailIllegalCodePoint, nil)
			}
		case stateScheme:
			tr := ASCIIAlphanumeric.Clone().Set(0x2b).Set(0x2d).Set(0x2e)
			if tr.Test(uint(r)) {
				buffer.WriteRune(unicode.ToLower(r))
			} else if r == ':' {
				if stateOverridden {
					//If url’s scheme is a special scheme and buffer is not a special scheme, then return.
					if url.isSpecialScheme(url.protocol) && !url.isSpecialScheme(buffer.String()) {
						return url, nil
					}
					//If url’s scheme is not a special scheme and buffer is a special scheme, then return.
					if !url.isSpecialScheme(url.protocol) && url.isSpecialScheme(buffer.String()) {
						return url, nil
					}
					//If url includes credentials or has a non-null port, and buffer is "file", then return.
					if (url.username != "" || url.password != "" || url.port != nil) && buffer.String() == "file" {
						return url, nil
					}
					//If url’s scheme is "file" and its host is an empty host or null, then return.
					if url.protocol == "file" && *url.host == "" {
						return url, nil
					}
				}
				url.protocol = buffer.String()
				if stateOverridden {
					url.cleanDefaultPort()
					return url, nil
				}
				buffer.Reset()
				if url.protocol == "file" {
					if !input.remainingStartsWith("//") {
						if err := p.handleError(url, errors.IllegalSlashes); err != nil {
							return nil, err
						}
					}
					state = stateFile
				} else if url.IsSpecialScheme() && base != nil && base.protocol == url.protocol {
					state = stateSpecialRelativeOrAuthority
					base.cannotBeABaseUrl = false
				} else if url.IsSpecialScheme() {
					state = stateSpecialAuthoritySlashes
				} else if input.remainingStartsWith("/") {
					state = statePathOrAuthority
					input.nextCodePoint()
				} else {
					url.cannotBeABaseUrl = true
					state = stateCannotBeABaseUrl
				}
			} else if !stateOverridden {
				buffer.Reset()
				state = stateNoScheme
				input.reset()
			} else {
				return p.handleFailure(url, errors.FailIllegalScheme, nil)
			}
		case stateNoScheme:
			if (base == nil || base.cannotBeABaseUrl) && r != '#' {
				return p.handleFailure(url, errors.FailRelativeUrlWithNoBase, nil)
			} else if base != nil && base.cannotBeABaseUrl && r == '#' {
				url.protocol = base.protocol
				url.path = base.path // TODO: Ensure copy????
				url.search = base.search
				url.hash = new(string)
				url.cannotBeABaseUrl = true
				state = stateFragment
			} else if base != nil && base.protocol != "file" {
				state = stateRelative
				input.rewindLast()
			} else {
				state = stateFile
				input.rewindLast()
			}
		case stateSpecialRelativeOrAuthority:
			if r == '/' && input.remainingStartsWith("/") {
				state = stateSpecialAuthorityIgnoreSlashes
				input.nextCodePoint()
			} else {
				if err := p.handleError(url, errors.IllegalSlashes); err != nil {
					return nil, err
				}
				state = stateRelative
				input.rewindLast()
			}
		case statePathOrAuthority:
			if r == '/' {
				state = stateAuthority
			} else {
				state = statePath
				input.rewindLast()
			}
		case stateRelative:
			url.protocol = base.protocol
			if r == '/' {
				state = stateRelativeSlash
			} else if url.isSpecialSchemeAndBackslash(r) {
				if err := p.handleError(url, errors.IllegalSlashes); err != nil {
					return nil, err
				}
				state = stateRelativeSlash
			} else {
				url.username = base.username
				url.password = base.password
				url.host = base.host
				url.port = base.port
				url.decodedPort = base.decodedPort
				url.path = base.path // TODO: Ensure copy????
				url.search = base.search
				if r == '?' {
					url.search = new(string)
					state = stateQuery
				} else if r == '#' {
					url.hash = new(string)
					state = stateFragment
				} else if !input.eof {
					url.search = nil
					if len(url.path) > 0 {
						url.path = url.path[0 : len(url.path)-1]
					}
					state = statePath
					input.rewindLast()
				}
			}
		case stateRelativeSlash:
			if url.IsSpecialScheme() && (r == '/' || r == '\\') {
				if r == '\\' {
					if err := p.handleError(url, errors.IllegalSlashes); err != nil {
						return nil, err
					}
				}
				state = stateSpecialAuthorityIgnoreSlashes
			} else if r == '/' {
				state = stateAuthority
			} else {
				url.username = base.username
				url.password = base.password
				url.host = base.host
				url.port = base.port
				state = statePath
				input.rewindLast()
			}
		case stateSpecialAuthoritySlashes:
			if r == '/' && input.remainingStartsWith("/") {
				state = stateSpecialAuthorityIgnoreSlashes
				input.nextCodePoint()
			} else {
				if err := p.handleError(url, errors.IllegalSlashes); err != nil {
					return nil, err
				}
				state = stateSpecialAuthorityIgnoreSlashes
				input.rewindLast()
			}
		case stateSpecialAuthorityIgnoreSlashes:
			if r != '/' && r != '\\' {
				state = stateAuthority
				input.rewindLast()
			} else {
				if err := p.handleError(url, errors.IllegalSlashes); err != nil {
					return nil, err
				}
			}
		case stateAuthority:
			if r == '@' {
				if err := p.handleError(url, errors.AtInAuthority); err != nil {
					return nil, err
				}
				if atFlag {
					// Prepend %40 to buffer
					tmp := buffer.String()
					buffer.Reset()
					buffer.WriteString("%40")
					buffer.WriteString(tmp)
				}
				atFlag = true
				bb := newInputString(buffer.String())
				c := bb.nextCodePoint()
				for !bb.eof {
					if c == ':' && !passwordTokenSeenFlag {
						passwordTokenSeenFlag = true
						c = bb.nextCodePoint()
						continue
					}
					encodedCodePoints := p.percentEncodeRune(c, UserInfoPercentEncodeSet)
					if passwordTokenSeenFlag {
						url.password += encodedCodePoints
					} else {
						url.username += encodedCodePoints
					}
					c = bb.nextCodePoint()
				}
				buffer.Reset()
			} else if (input.eof || r == '/' || r == '?' || r == '#') || url.isSpecialSchemeAndBackslash(r) {
				if atFlag && buffer.Len() == 0 {
					return p.handleFailure(url, errors.FailMissingHost, nil)
				}
				input.rewind(len([]rune(buffer.String())) + 1)
				buffer.Reset()
				state = stateHost
			} else {
				buffer.WriteRune(r)
			}
		case stateHost:
			fallthrough
		case stateHostname:
			if stateOverridden && url.protocol == "file" {
				input.rewindLast()
				state = stateFileHost
			} else if r == ':' && !bracketFlag {
				if buffer.Len() == 0 {
					return p.handleFailure(url, errors.FailMissingHost, nil)
				}
				host, err := p.parseHost(url, p, buffer.String(), !url.IsSpecialScheme())
				if err != nil {
					return p.handleFailure(url, errors.FailIllegalHost, err)
				}
				url.host = &host
				buffer.Reset()
				state = statePort

				if stateOverride == stateHostname {
					return url, nil
				}
			} else if (input.eof || r == '/' || r == '?' || r == '#') || url.isSpecialSchemeAndBackslash(r) {
				input.rewindLast()
				if url.IsSpecialScheme() && buffer.Len() == 0 {
					return p.handleFailure(url, errors.FailMissingHost, nil)
				} else if stateOverridden && buffer.Len() == 0 && (url.username != "" || url.password != "" || url.port != nil) {
					return p.handleFailure(url, errors.FailMissingHost, nil)
				} else {
					host, err := p.parseHost(url, p, buffer.String(), !url.IsSpecialScheme())
					if err != nil {
						return p.handleFailure(url, errors.FailIllegalHost, err)
					}
					url.host = &host
					buffer.Reset()
					state = statePathStart
					if stateOverridden {
						return url, nil
					}
				}
			} else {
				if r == '[' {
					bracketFlag = true
				} else if r == ']' {
					bracketFlag = false
				}
				if input.currentIsInvalid() && p.opts.acceptInvalidCodepoints {
					buffer.WriteString(string([]byte{input.getCurrentAsByte()}))
				} else {
					buffer.WriteRune(r)
				}
			}
		case statePort:
			if ASCIIDigit.Test(uint(r)) {
				buffer.WriteRune(r)
			} else if (input.eof || r == '/' || r == '?' || r == '#') || url.isSpecialSchemeAndBackslash(r) || stateOverridden {
				if buffer.Len() > 0 {
					port, err := strconv.Atoi(buffer.String())
					if err != nil {
						return p.handleFailure(url, errors.FailIllegalPort, nil)
					}
					if port > 65535 {
						return p.handleFailure(url, errors.FailIllegalPort, nil)
					}
					portString := strconv.Itoa(port)
					url.decodedPort = port
					url.port = &portString
					url.cleanDefaultPort()
					buffer.Reset()
				}
				if stateOverridden {
					return url, nil
				}
				state = statePathStart
				input.rewindLast()
			} else {
				return p.handleFailure(url, errors.FailIllegalPort, nil)
			}
		case stateFile:
			url.protocol = "file"
			url.host = new(string)
			if r == '/' || r == '\\' {
				if r == '\\' {
					if err := p.handleError(url, errors.IllegalSlashes); err != nil {
						return nil, err
					}
				}
				state = stateFileSlash
			} else if base != nil && base.protocol == "file" {
				url.host = base.host
				url.path = base.path // TODO: Ensure copy????
				url.search = base.search
				if r == '?' {
					url.search = new(string)
					state = stateQuery
				} else if r == '#' {
					url.hash = new(string)
					state = stateFragment
				} else if !input.eof {
					url.search = nil
					if !startsWithAWindowsDriveLetter(input.remainingFromPointer()) {
						shortenPath(url)
					} else {
						if err := p.handleError(url, errors.BadWindowsDriveLetter); err != nil {
							return nil, err
						}
						url.path = []string{}
					}
					state = statePath
					input.rewindLast()

				}
			} else {
				state = statePath
				input.rewindLast()
			}
		case stateFileSlash:
			if r == '/' || r == '\\' {
				if r == '\\' {
					if err := p.handleError(url, errors.IllegalSlashes); err != nil {
						return nil, err
					}
				}
				state = stateFileHost
			} else {
				if base != nil && base.protocol == "file" {
					url.host = base.host
					if !startsWithAWindowsDriveLetter(input.remainingFromPointer()) && base.path != nil && isNormalizedWindowsDriveLetter(base.path[0]) {
						// This is a (platform-independent) Windows drive letter quirk. Both url’s and base’s host are null under these conditions and therefore not copied
						url.path = append(url.path, base.path[0])
					}
				}
				state = statePath
				input.rewindLast()
			}
		case stateFileHost:
			if input.eof || r == '/' || r == '\\' || r == '?' || r == '#' {
				input.rewindLast()
				if !stateOverridden && isWindowsDriveLetter(buffer.String()) {
					if err := p.handleError(url, errors.BadWindowsDriveLetter); err != nil {
						return nil, err
					}
					state = statePath
				} else if buffer.Len() == 0 {
					url.host = new(string)
					if stateOverridden {
						return nil, nil
					}
					state = statePathStart
				} else {
					host, err := p.parseHost(url, p, buffer.String(), !url.IsSpecialScheme())
					if err != nil {
						return p.handleFailure(url, errors.FailIllegalHost, err)
					}
					if host == "localhost" {
						host = ""
					}
					url.host = &host
					if stateOverridden {
						return url, nil
					}
					buffer.Reset()
					state = statePathStart
				}
			} else {
				buffer.WriteRune(r)
			}
		case statePathStart:
			if url.IsSpecialScheme() && !p.opts.skipTrailingSlashNormalization {
				if r == '\\' {
					if err := p.handleError(url, errors.IllegalSlashes); err != nil {
						return nil, err
					}
				}
				state = statePath
				if r != '/' && r != '\\' {
					input.rewindLast()
				}
			} else if !stateOverridden && r == '?' {
				url.search = new(string)
				state = stateQuery
			} else if !stateOverridden && r == '#' {
				url.hash = new(string)
				state = stateFragment
			} else if !input.eof {
				state = statePath
				if r != '/' {
					input.rewindLast()
				}
			}
		case statePath:
			if (input.eof || r == '/') ||
				url.isSpecialSchemeAndBackslash(r) ||
				(!stateOverridden && (r == '?' || r == '#')) {

				if url.isSpecialSchemeAndBackslash(r) {
					if err := p.handleError(url, errors.IllegalSlashes); err != nil {
						return nil, err
					}
				}
				if isDoubleDotPathSegment(buffer.String()) {
					shortenPath(url)

					if r != '/' && !url.isSpecialSchemeAndBackslash(r) {
						url.path = append(url.path, "")
					}
				} else if isSingleDotPathSegment(buffer.String()) && r != '/' && !url.isSpecialSchemeAndBackslash(r) {
					url.path = append(url.path, "")
				} else if !isSingleDotPathSegment(buffer.String()) {
					if url.protocol == "file" && len(url.path) == 0 && isWindowsDriveLetter(buffer.String()) {
						// replace second code point in buffer with U+003A (:).
						// This is a (platform-independent) Windows drive letter quirk.
						if !p.opts.skipWindowsDriveLetterNormalization {
							b := buffer.String()
							buffer.Reset()
							buffer.WriteString(b[0:1] + ":" + b[2:])
						}
					}
					if !p.opts.collapseConsecutiveSlashes || !url.IsSpecialScheme() || len(url.path) == 0 || len(url.path[len(url.path)-1]) > 0 {
						url.path = append(url.path, buffer.String())
					} else {
						url.path[len(url.path)-1] = buffer.String()
					}
				}
				buffer.Reset()
				if r == '?' {
					url.search = new(string)
					state = stateQuery
				} else if r == '#' {
					url.hash = new(string)
					state = stateFragment
				}
			} else {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.IllegalCodePoint); err != nil {
						return nil, err
					}
				}
				invalidPercentEncoding := input.remainingIsInvalidPercentEncoded()
				if invalidPercentEncoding {
					if err := p.handleError(url, errors.InvalidPercentEncoding); err != nil {
						return nil, err
					}
				}
				if invalidPercentEncoding {
					buffer.WriteString(p.percentEncodeInvalidRune(r, p.opts.pathPercentEncodeSet))
				} else {
					buffer.WriteString(p.percentEncodeRune(r, p.opts.pathPercentEncodeSet))
				}
			}
		case stateCannotBeABaseUrl:
			if r == '?' {
				url.search = new(string)
				state = stateQuery
			} else if r == '#' {
				url.hash = new(string)
				state = stateFragment
			} else {
				if !input.eof && !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.IllegalCodePoint); err != nil {
						return nil, err
					}
				}
				invalidPercentEncoding := input.remainingIsInvalidPercentEncoded()
				if invalidPercentEncoding {
					if err := p.handleError(url, errors.InvalidPercentEncoding); err != nil {
						return nil, err
					}
				}
				if !input.eof {
					if len(url.path) == 0 {
						url.path = append(url.path, "")
					}
					if invalidPercentEncoding {
						url.path[0] += p.percentEncodeInvalidRune(r, C0PercentEncodeSet)
					} else {
						url.path[0] += p.percentEncodeRune(r, C0PercentEncodeSet)
					}
				}

			}
		case stateQuery:
			if !stateOverridden && r == '#' {
				url.hash = new(string)
				state = stateFragment
			} else if !input.eof {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.IllegalCodePoint); err != nil {
						return nil, err
					}
				}
				if input.remainingIsInvalidPercentEncoded() {
					if err := p.handleError(url, errors.InvalidPercentEncoding); err != nil {
						return nil, err
					}
				}
				encodeSet := p.opts.queryPercentEncodeSet
				if url.isSpecialScheme(url.protocol) {
					encodeSet = p.opts.specialQueryPercentEncodeSet
				}
				*url.search += p.percentEncodeRune(r, encodeSet)
			}
		case stateFragment:
			if !input.eof {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.IllegalCodePoint); err != nil {
						return nil, err
					}
				}
				if input.remainingIsInvalidPercentEncoded() {
					if err := p.handleError(url, errors.InvalidPercentEncoding); err != nil {
						return nil, err
					}
				}
				encodeSet := p.opts.fragmentPercentEncodeSet
				if url.isSpecialScheme(url.protocol) {
					encodeSet = p.opts.specialFragmentPercentEncodeSet
				}
				*url.hash += p.percentEncodeRune(r, encodeSet)
			}
		}

		if input.eof {
			break
		}
	}

	return url, nil
}

func (p *parser) percentEncodeInvalidRune(r rune, tr *PercentEncodeSet) string {
	if p.opts.percentEncodeSinglePercentSign {
		return p.percentEncodeRune(r, tr.Set(0x25))
	}
	return p.percentEncodeRune(r, tr)
}

func (p *parser) percentEncodeRune(r rune, tr *PercentEncodeSet) string {
	if tr != nil && !tr.RuneShouldBeEncoded(r) {
		return string(r)
	}

	var bytes = make([]byte, 4)
	var n int
	if p.opts.encodingOverride != nil {
		b, _ := p.opts.encodingOverride.EncodeRune(r)
		bytes[0] = b
		n = 1
	} else {
		n = utf8.EncodeRune(bytes, r)
	}

	percentEncoded := make([]byte, 4*3)
	j := 0
	for i := 0; i < n; i++ {
		c := bytes[i]
		percentEncoded[j] = '%'
		percentEncoded[j+1] = "0123456789ABCDEF"[c>>4]
		percentEncoded[j+2] = "0123456789ABCDEF"[c&15]
		j += 3
	}
	return string(percentEncoded[:j])
}

func (p *parser) PercentEncodeString(s string, tr *PercentEncodeSet) string {
	buffer := &strings.Builder{}
	runes := []rune(s)
	for i, r := range runes {
		if r == '%' {
			if len(runes) < (i+3) ||
				(!ASCIIHexDigit.Test(uint(runes[i+1])) || !ASCIIHexDigit.Test(uint(runes[i+2]))) {
				if p.opts.percentEncodeSinglePercentSign {
					buffer.WriteString(p.percentEncodeRune(r, tr.Set(0x25)))
					continue
				}
			}
		}
		buffer.WriteString(p.percentEncodeRune(r, tr))
	}
	return buffer.String()
}

func (p *parser) DecodePercentEncoded(s string) string {
	sb := strings.Builder{}
	bytes := []byte(s)
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != '%' {
			sb.WriteByte(bytes[i])
		} else if len(bytes) < (i+3) ||
			(!ASCIIHexDigit.Test(uint(bytes[i+1])) || !ASCIIHexDigit.Test(uint(bytes[i+2]))) {
			sb.WriteByte(bytes[i])
		} else {
			b, e := u2.PathUnescape(string(bytes[i : i+3]))
			if e != nil {
				return sb.String()
			}
			if p.opts.encodingOverride != nil {
				r := p.opts.encodingOverride.DecodeByte(b[0])
				sb.WriteRune(r)
			} else {
				sb.WriteString(b)
			}
			i += 2
		}
	}
	return sb.String()
}

func isSingleDotPathSegment(s string) bool {
	if s == "." {
		return true
	}
	s = strings.ToLower(s)
	if s == "%2e" {
		return true
	}
	return false
}

func isDoubleDotPathSegment(s string) bool {
	if s == ".." {
		return true
	}
	s = strings.ToLower(s)
	if s == ".%2e" || s == "%2e." || s == "%2e%2e" {
		return true
	}
	return false
}

func shortenPath(u *Url) {
	if len(u.path) == 0 {
		return
	}
	if u.protocol == "file" && len(u.path) == 1 && isNormalizedWindowsDriveLetter(u.path[0]) {
		return
	}
	if len(u.path) == 1 {
		u.path = nil
	} else {
		u.path = u.path[0 : len(u.path)-1]
	}
}

func startsWithAWindowsDriveLetter(s string) bool {
	if len(s) >= 2 && isWindowsDriveLetter(s[0:2]) &&
		(len(s) == 2 || s[2] == '/' || s[2] == '\\' || s[2] == '?' || s[2] == '#') {
		return true
	}

	return false
}

func isWindowsDriveLetter(s string) bool {
	if len(s) == 2 && ASCIIAlpha.Test(uint(s[0])) &&
		(s[1] == ':' || s[1] == '|') {
		return true
	}
	return false
}

func isNormalizedWindowsDriveLetter(s string) bool {
	if len(s) == 2 && ASCIIAlpha.Test(uint(s[0])) &&
		(s[1] == ':') {
		return true
	}
	return false
}

func trimPrefix(s string, tr *PercentEncodeSet) (string, bool) {
	if s == "" {
		return s, false
	}
	for i, c := range s {
		if tr.RuneNotInSet(c) {
			return s[i:], i > 0
		}
	}
	return "", true
}

func trimPostfix(s string, tr *PercentEncodeSet) (string, bool) {
	if s == "" {
		return s, false
	}
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if tr.RuneNotInSet(int32(c)) {
			return s[:i+1], i < (len(s) - 1)
		}
	}
	return "", true
}

func trim(s string, tr *PercentEncodeSet) (string, bool) {
	var c1, c2 bool
	s, c1 = trimPrefix(s, tr)
	s, c2 = trimPostfix(s, tr)
	return s, c1 || c2
}

func remove(s string, tr *bitset.BitSet) (string, bool) {
	if s == "" {
		return s, false
	}
	changed := false
	var r []byte
	for _, c := range []byte(s) {
		if tr.Test(uint(c)) {
			changed = true
		} else {
			r = append(r, byte(c))
		}
	}
	return string(r), changed
}

func (u *Url) IsSpecialScheme() bool {
	return u.isSpecialScheme(u.protocol)
}

func (u *Url) isSpecialScheme(s string) bool {
	_, ok := u.getSpecialScheme(s)
	return ok
}

func (u *Url) getSpecialScheme(s string) (string, bool) {
	dp, ok := u.parser.opts.specialSchemes[s]
	return dp, ok
}

func (u *Url) isSpecialSchemeAndBackslash(r rune) bool {
	ok := u.IsSpecialScheme()
	return ok && r == '\\'
}

func (u *Url) cleanDefaultPort() {
	if dp, ok := u.getSpecialScheme(u.protocol); ok && (u.port == nil || dp == *u.port) {
		u.port = nil
	}
}

func (u *Url) getDefaultPort() int {
	if dp, ok := u.getSpecialScheme(u.protocol); ok {
		if p, err := strconv.Atoi(dp); err == nil {
			return p
		}
	}
	return 0
}
