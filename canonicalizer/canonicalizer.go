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

package canonicalizer

import (
	"github.com/nlnwa/whatwg-url/errors"
	"github.com/nlnwa/whatwg-url/url"
	"strings"
)

func New(opts ...url.ParserOption) url.Parser {
	p := &profile{
		Parser:    url.NewParser(opts...),
		sortQuery: NoSort,
	}
	for _, opt := range opts {
		if o, ok := opt.(canonParserOption); ok {
			o.applyProfile(p)
		}
	}
	return p
}

type profile struct {
	url.Parser
	removeUserInfo          bool
	removePort              bool
	removeFragment          bool
	sortQuery               querySort
	repeatedPercentDecoding bool
	defaultScheme           string
}

func (p *profile) Parse(rawUrl string) (*url.Url, error) {
	u, err := p.Parser.Parse(rawUrl)
	if err != nil {
		if errors.Code(err) == errors.FailRelativeUrlWithNoBase && p.defaultScheme != "" {
			rawUrl = p.defaultScheme + "://" + rawUrl
			u, err = p.Parser.Parse(rawUrl)
		}
		if err != nil {
			return nil, err
		}
	}

	return p.canonicalize(u)
}

func (p *profile) ParseRef(rawUrl, ref string) (*url.Url, error) {
	b, err := p.Parser.Parse(rawUrl)
	if err != nil {
		if errors.Code(err) == errors.FailRelativeUrlWithNoBase && p.defaultScheme != "" {
			rawUrl = p.defaultScheme + "://" + rawUrl
			b, err = p.Parser.Parse(rawUrl)
		}
		if err != nil {
			return nil, err
		}
	}

	u, err := b.Parse(ref)
	if err != nil {
		return nil, err
	}

	return p.canonicalize(u)
}

func (p *profile) canonicalize(u *url.Url) (*url.Url, error) {
	if p.repeatedPercentDecoding {
		if u.Hostname() != "" {
			u.SetHostname(decodeEncode(p.Parser, u.Hostname()))
		}
		if u.Pathname() != "" {
			u.SetPathname(decodeEncode(p.Parser, u.Pathname()))
		}
		if u.Search() != "" {
			u.SearchParams().Iterate(func(pair *url.NameValuePair) {
				pair.Name = decodeEncode(p.Parser, pair.Name)
				pair.Value = decodeEncode(p.Parser, pair.Value)
			})
		}
		if u.Hash() != "" {
			u.SetHash(decodeEncode(p.Parser, strings.TrimPrefix(u.Hash(), "#")))
		}
	}

	if p.removePort {
		u.SetPort("")
	}
	if p.removeUserInfo {
		u.SetUsername("")
		u.SetPassword("")
	}
	if p.removeFragment {
		u.SetHash("")
	}

	switch p.sortQuery {
	case SortKeys:
		u.SearchParams().Sort()
	case SortParameter:
		u.SearchParams().SortAbsolute()
	}

	return u, nil
}

var GoogleSafeBrowsingPercentEncodeSet = url.NewPercentEncodeSet(33, '#', '%')

func decodeEncode(parser url.Parser, s string) string {
	r := parser.PercentEncodeString(repeatedDecode(parser, s), GoogleSafeBrowsingPercentEncodeSet)
	return r
}

// repeatedDecode repeatedly percent-unescape a string until it has no more percent-escapes
func repeatedDecode(parser url.Parser, s string) string {
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
