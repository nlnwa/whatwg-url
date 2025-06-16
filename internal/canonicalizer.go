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
	gerrors "errors"

	"github.com/nationallibraryofnorway/whatwg-url/errors"
)

func NewCanonicalizer(parser *Parser) *Canonicalizer {
	return &Canonicalizer{
		Parser:                        parser,
		SortQuery:                     NoSort,
		RepeatedQueryPercentEncodeSet: RepeatedQueryPercentEncodeSet,
	}
}

type Canonicalizer struct {
	DefaultScheme string

	*Parser
	RepeatedQueryPercentEncodeSet *PercentEncodeSet

	SortQuery QuerySort

	RemoveUserInfo            bool
	RemovePort                bool
	RemoveFragment            bool
	RepeatedPercentDecoding   bool
	PercentEncodedSpaceToPlus bool
}

func (c *Canonicalizer) Parse(rawUrl string) (*Url, error) {
	u, err := c.parse(rawUrl)
	if err != nil {
		return nil, err
	}
	return c.canonicalize(u)
}

func (c *Canonicalizer) ParseRef(rawUrl, ref string) (*Url, error) {
	b, err := c.parse(rawUrl)
	if err != nil {
		return nil, err
	}

	u, err := b.parser.parse(ref, b, nil, NoState)
	if err != nil {
		return nil, err
	}

	return c.canonicalize(u)
}

func (c *Canonicalizer) parse(rawUrl string) (*Url, error) {
	u, err := c.Parser.parse(rawUrl, nil, nil, NoState)
	if err == nil {
		return u, nil
	}
	if gerrors.Is(err, errors.ErrMissingSchemeNonRelativeURL) && c.DefaultScheme != "" {
		rawUrl = c.DefaultScheme + "://" + rawUrl
		return c.Parser.parse(rawUrl, nil, nil, NoState)
	}
	return nil, err
}

func (c *Canonicalizer) canonicalize(u *Url) (*Url, error) {
	if c.RepeatedPercentDecoding {
		if u.Hostname() != "" {
			hostname := c.repeatDecodeThenEncode(u.Hostname(), HostPercentEncodeSet)
			u.host = hostname
		}
		if u.Pathname() != "" {
			pathname := c.repeatDecodeThenEncode(u.Pathname(), c.PathPercentEncodeSet)
			u.SetPathname(pathname)
		}
		if u.Search() != "" {
			u.SearchParams().Iterate(func(pair *NameValuePair) {
				pair.Name = c.repeatDecodeThenEncode(pair.Name, c.RepeatedQueryPercentEncodeSet)
				pair.Value = c.repeatDecodeThenEncode(pair.Value, c.RepeatedQueryPercentEncodeSet)
			})
		}
		if u.Hash() != "" && !c.RemoveFragment {
			u.fragment = c.repeatDecodeThenEncode(u.fragment, FragmentPercentEncodeSet)
		}
	}
	if c.RemoveUserInfo {
		u.SetUsername("")
		u.SetPassword("")
	}
	if c.RemovePort {
		u.SetPort("")
	}

	switch c.SortQuery {
	case SortKeys:
		u.SearchParams().Sort()
	case SortParameter:
		u.SearchParams().SortAbsolute()
	}
	if c.RemoveFragment {
		u.SetHash("")
	}

	return u, nil
}

type QuerySort int

const (
	// Do not sort query.
	NoSort QuerySort = iota
	// Stable sort on query parameter keys.
	SortKeys
	// Sort on entire query parameter.
	SortParameter
)
