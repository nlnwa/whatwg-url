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
	"iter"
	"sort"
	"strings"

	"golang.org/x/text/encoding/unicode"
)

type NameValuePair struct {
	Name, Value string
}

type params []NameValuePair

// https://url.spec.whatwg.org/#concept-urlencoded-parser
func (s *params) Init(query string) {
	*s = (*s)[:0]
	decoder := unicode.UTF8.NewDecoder()
	for q := range strings.SplitSeq(query, "&") {
		if q == "" {
			continue
		}

		name, value, found := strings.Cut(q, "=")

		name = strings.ReplaceAll(name, "+", " ")
		name = percentDecode(name, nil)

		if found {
			value = strings.ReplaceAll(value, "+", " ")
			value = percentDecode(value, nil)
		}

		// UTF-8 decode without BOM handling preserves a leading U+FEFF.
		// This decoder replaces malformed sequences with U+FFFD and cannot fail.
		// https://encoding.spec.whatwg.org/#utf-8-decode-without-bom
		name, _ = decoder.String(name)
		value, _ = decoder.String(value)

		*s = append(*s, NameValuePair{Name: name, Value: value})
	}
}

func (s *params) Encode(ps *PercentEncodeSet) string {
	var b strings.Builder

	for i, p := range *s {
		if i > 0 {
			b.WriteByte('&')
		}

		for _, r := range p.Name {
			if r == ' ' {
				b.WriteByte('+')
				continue
			}
			b.Write(percentEncode(r, ps))

		}

		b.WriteByte('=')

		for _, r := range p.Value {
			if r == ' ' {
				b.WriteByte('+')
				continue
			}
			b.Write(percentEncode(r, ps))
		}
	}

	return b.String()
}

func (s *params) String() string {
	return s.Encode(FormPercentEncodeSet)
}

// Append appends a new name/value pair to the search parameters.
func (s *params) Append(name, value string) {
	*s = append(*s, NameValuePair{Name: name, Value: value})
}

// Delete deletes the given search parameter, and its associated value(s), from the search parameters.
func (s *params) Delete(name string) {
	out := (*s)[:0]
	for _, nvp := range *s {
		if nvp.Name != name {
			out = append(out, nvp)
		}
	}
	*s = out
}

// Filter removes name/value pairs from the search parameters for which the given function returns false.
func (s *params) Filter(keep func(name, value string) bool) {
	out := (*s)[:0]
	for _, p := range *s {
		if keep(p.Name, p.Value) {
			out = append(out, p)
		}
	}
	*s = out
}

// Get returns the first value associated with the given search parameter name.
func (s *params) Get(name string) string {
	for _, nvp := range *s {
		if nvp.Name == name {
			return nvp.Value
		}
	}
	return ""
}

// GetAll returns all the values associated with the given search parameter name.
func (s *params) GetAll(name string) []string {
	var result []string
	for _, nvp := range *s {
		if nvp.Name == name {
			result = append(result, nvp.Value)
		}
	}
	return result
}

// Has returns true if the search parameters contains a parameter with the given name.
func (s *params) Has(name string) bool {
	for _, nvp := range *s {
		if nvp.Name == name {
			return true
		}
	}
	return false
}

// Set sets the value associated with name to value. It replaces any existing values associated with name.
func (s *params) Set(name, value string) {
	isSet := false
	params := (*s)[:0]
	for _, nvp := range *s {
		if nvp.Name == name {
			if isSet {
				continue
			}
			nvp.Value = value
			isSet = true
		}
		params = append(params, nvp)
	}
	if !isSet {
		*s = append(params, NameValuePair{Name: name, Value: value})
	} else {
		*s = params
	}
}

// Sort sorts the search parameters by name.
func (s *params) Sort() {
	sort.SliceStable(*s, func(i, j int) bool {
		return (*s)[i].Name < (*s)[j].Name
	})
}

// SortAbsolute sorts the search parameters by name and value.
func (s *params) SortAbsolute() {
	sort.SliceStable(*s, func(i, j int) bool {
		if (*s)[i].Name != (*s)[j].Name {
			return (*s)[i].Name < (*s)[j].Name
		}
		return (*s)[i].Value < (*s)[j].Value
	})
}

func (v *params) Values() iter.Seq[*NameValuePair] {
	return func(yield func(*NameValuePair) bool) {
		for i := range *v {
			if !yield(&(*v)[i]) {
				return
			}
		}
	}
}

// SearchParams represents a query as application/x-www-form-urlencoded data.
// Parsing converts '+' to space, percent-decodes and UTF-8 decodes names and
// values (replacing malformed UTF-8 with U+FFFD), drops empty terms,
// and loses the distinction between a bare term ("flag") and an empty value
// ("flag="). Every mutation serializes the query as form data.
//
// Use SearchParams when the query represents form data. Use RawQuery when its
// serialized spelling matters.
type SearchParams struct {
	url    *Url
	values params

	deferSync bool
	dirty     bool
}

func (p *SearchParams) Get(name string) string      { return p.values.Get(name) }
func (p *SearchParams) GetAll(name string) []string { return p.values.GetAll(name) }
func (p *SearchParams) Has(name string) bool        { return p.values.Has(name) }

func (p *SearchParams) Append(name, value string) {
	p.values.Append(name, value)
	p.markDirty()
}

func (p *SearchParams) Delete(name string) {
	p.values.Delete(name)
	p.markDirty()
}

func (p *SearchParams) Filter(keep func(name, value string) bool) {
	p.values.Filter(keep)
	p.markDirty()
}

func (p *SearchParams) Set(name, value string) {
	p.values.Set(name, value)
	p.markDirty()
}

// Sort stably sorts the decoded form parameters by name and synchronizes the
// URL using application/x-www-form-urlencoded serialization.
func (p *SearchParams) Sort() {
	p.values.Sort()
	p.markDirty()
}

// SortAbsolute stably sorts the decoded form parameters by name and value and
// synchronizes the URL using application/x-www-form-urlencoded serialization.
func (p *SearchParams) SortAbsolute() {
	p.values.SortAbsolute()
	p.markDirty()
}

func (p *SearchParams) Iterate(f func(pair *NameValuePair)) {
	for i := range p.values {
		f(&p.values[i])
	}
	p.markDirty()
}

func (p *SearchParams) String() string {
	return p.values.String()
}

func (p *SearchParams) markDirty() {
	if p.deferSync {
		p.dirty = true
		return
	}
	p.sync()
}

func (p *SearchParams) sync() {
	q := p.values.String()
	if q == "" {
		p.url.query = ""
		p.url.hasQuery = false
		if !p.url.hasFragment {
			p.url.path.stripTrailingSpacesIfOpaque()
		}
	} else {
		p.url.query = q
		p.url.hasQuery = true
	}
}
