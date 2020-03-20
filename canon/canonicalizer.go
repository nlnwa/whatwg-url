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

func New(opts ...url.ParserOption) Profile {
	p := &profile{
		parser:    url.NewParser(opts...),
		sortQuery: NoSort,
	}
	for _, opt := range opts {
		if o, ok := opt.(CanonParserOption); ok {
			o.applyProfile(p)
		}
	}
	return p
}

type Profile interface {
	Canonicalize(s string) string
}

type profile struct {
	parser                  url.Parser
	removeUserInfo          bool
	removePort              bool
	removeFragment          bool
	sortQuery               querySort
	repeatedPercentDecoding bool
	defaultScheme           string
}

func (p *profile) Canonicalize(s string) string {
	u, err := p.parser.Parse(s)
	if err != nil {
		if errors.Code(err) == errors.FailRelativeUrlWithNoBase && p.defaultScheme != "" {
			s = p.defaultScheme + "://" + s
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

	if p.removePort {
		u.SetPort("")
	}
	if p.removeUserInfo {
		u.SetUsername("")
		u.SetPassword("")
	}

	switch p.sortQuery {
	case SortKeys:
		u.SearchParams().Sort()
	case SortParameter:
		u.SearchParams().SortAbsolute()
	}

	return u.Href(p.removeFragment)
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

type CanonParserOption interface {
	applyProfile(*profile)
}

// funcCanonParserOption wraps a function that canonicalizes url into an
// implementation of the CanonParserOption interface.
type funcCanonParserOption struct {
	url.EmptyParserOption
	f func(*profile)
}

func (cpo *funcCanonParserOption) applyProfile(p *profile) {
	cpo.f(p)
}

// WithRemoveUserInfo removes username and password from url.
//
// This API is EXPERIMENTAL.
func WithRemoveUserInfo() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.removeUserInfo = true
		},
	}
}

// WithRemovePort always removes port from url. Default is to remove port if port is default port for scheme.
//
// This API is EXPERIMENTAL.
func WithRemovePort() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.removePort = true
		},
	}
}

// WithRemoveFragment removes the fragment part of the url.
//
// This API is EXPERIMENTAL.
func WithRemoveFragment() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.removeFragment = true
		},
	}
}

// WithRepeatedPercentDecoding.
//
// This API is EXPERIMENTAL.
func WithRepeatedPercentDecoding() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.repeatedPercentDecoding = true
		},
	}
}

// WithDefaultScheme sets a scheme to add if url is missing scheme.
//
// This API is EXPERIMENTAL.
func WithDefaultScheme(scheme string) url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.defaultScheme = scheme
		},
	}
}

// WithSortQuery sets sort type for query parameters.
// if query should be sorted: 0 = no sort, 1 = sort keys, but leave repeated keys in same order, 2 = sort key,value
//
// This API is EXPERIMENTAL.
func WithSortQuery(sortType querySort) url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.sortQuery = sortType
		},
	}
}

type querySort int

const (
	// Do not sort query.
	NoSort querySort = iota
	// Stable sort on query parameter keys.
	SortKeys
	// Sort on entire query parameter.
	SortParameter
)
