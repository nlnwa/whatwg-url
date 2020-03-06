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

package canon

import (
	"fmt"
	"github.com/nlnwa/whatwg-url/errors"
	"github.com/nlnwa/whatwg-url/url"
	"strings"
)

type Profile struct {
	parser                  *url.Parser
	RemoveUserInfo          bool
	RemovePort              bool
	RemoveFragment          bool
	SortQuery               bool
	SortQueryKeys           bool
	repeatedPercentDecoding bool
	DefaultScheme           string
}

func (p *Profile) Canonicalize(s string) string {
	u, err := p.parser.Parse(s)
	if err != nil {
		if errors.Code(err) == errors.FailRelativeUrlWithNoBase && p.DefaultScheme != "" {
			s = p.DefaultScheme + "://" + s
			u, err = p.parser.Parse(s)
		}
		if err != nil {
			fmt.Printf("Parse() error = %v", err)
			return s
		}
	}

	if p.repeatedPercentDecoding {
		if u.Hostname() != "" {
			u.SetHostname(decodeEncode(p.parser, u.Hostname()))
		}
		if u.Pathname() != "" {
			u.SetPathname(decodeEncode(p.parser, u.Pathname()))
		}
		if u.Search() != "" {
			u.SearchParams().Iterate(func(pair *url.NameValuePair) {
				pair.Name = decodeEncode(p.parser, pair.Name)
				pair.Value = decodeEncode(p.parser, pair.Value)
			})
		}
		if u.Hash() != "" {
			u.SetHash(decodeEncode(p.parser, strings.TrimPrefix(u.Hash(), "#")))
		}
	}

	if p.RemovePort {
		u.SetPort("")
	}
	if p.RemoveUserInfo {
		u.SetUsername("")
		u.SetPassword("")
	}

	if p.SortQueryKeys {
		u.SearchParams().Sort()
	} else if p.SortQuery {
		u.SearchParams().SortAbsolute()
	}

	return u.Href(p.RemoveFragment)
}

var GoogleSafeBrowsingPercentEncodeSet = url.NewPercentEncodeSet(33, '#', '%')

func decodeEncode(parser *url.Parser, s string) string {
	r := parser.PercentEncodeString(repeatedDecode(parser, s), GoogleSafeBrowsingPercentEncodeSet)
	return r
}

// repeatedDecode repeatedly percent-unescape a string until it has no more percent-escapes
func repeatedDecode(parser *url.Parser, s string) string {
	var r string
	for {
		r = parser.DecodePercentEncoded(s)
		if s == r {
			break
		}
		s = r
	}
	return s
}
