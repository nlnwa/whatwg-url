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
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestUrlSearchParams_IsLazyAssociatedObject(t *testing.T) {
	u, err := testParser.Parse("https://example.test/?a=1")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if u.searchParams != nil {
		t.Fatal("Parse() initialized searchParams")
	}

	u.SetSearch("?b=2")
	if u.searchParams != nil {
		t.Fatal("SetSearch() initialized searchParams")
	}

	got := u.SearchParams()
	if got == nil {
		t.Fatal("SearchParams() = nil")
	}
	if got != u.SearchParams() {
		t.Fatal("SearchParams() returned different objects")
	}
}

func TestUrlSearchParams_SetSearchUpdatesAssociatedObject(t *testing.T) {
	tests := []struct {
		name           string
		search         string
		wantParams     string
		wantHref       string
		wantQuery      string
		wantSearch     string
		wantMutatedURL string
	}{
		{
			name:           "non-empty",
			search:         "?b=2",
			wantParams:     "b=2",
			wantHref:       "https://example.test/?b=2",
			wantQuery:      "b=2",
			wantSearch:     "?b=2",
			wantMutatedURL: "https://example.test/?b=2&c=3",
		},
		{
			name:           "empty",
			search:         "",
			wantParams:     "",
			wantHref:       "https://example.test/",
			wantQuery:      "",
			wantSearch:     "",
			wantMutatedURL: "https://example.test/?c=3",
		},
		{
			name:           "empty query",
			search:         "?",
			wantParams:     "",
			wantHref:       "https://example.test/?",
			wantQuery:      "",
			wantSearch:     "",
			wantMutatedURL: "https://example.test/?c=3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := testParser.Parse("https://example.test/?a=1")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			searchParams := u.SearchParams()

			u.SetSearch(tt.search)

			if searchParams != u.SearchParams() {
				t.Fatal("SetSearch() replaced the associated SearchParams object")
			}
			if got := searchParams.String(); got != tt.wantParams {
				t.Errorf("SearchParams().String() = %q, want %q", got, tt.wantParams)
			}
			if got := u.Query(); got != tt.wantQuery {
				t.Errorf("Query() = %q, want %q", got, tt.wantQuery)
			}
			if got := u.Search(); got != tt.wantSearch {
				t.Errorf("Search() = %q, want %q", got, tt.wantSearch)
			}
			if got := u.Href(false); got != tt.wantHref {
				t.Errorf("Href() = %q, want %q", got, tt.wantHref)
			}

			searchParams.Append("c", "3")
			if got := u.Href(false); got != tt.wantMutatedURL {
				t.Errorf("Href() after Append() = %q, want %q", got, tt.wantMutatedURL)
			}
		})
	}
}

func TestUrlSearchParams_MutationsUpdateURL(t *testing.T) {
	u, err := testParser.Parse("https://example.test/?b=2&a=1&a=2")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	searchParams := u.SearchParams()

	searchParams.Set("a", "3")
	searchParams.Append("c", "4")
	searchParams.Delete("b")
	searchParams.Sort()

	if got, want := u.Query(), "a=3&c=4"; got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
	if got, want := u.Search(), "?a=3&c=4"; got != want {
		t.Errorf("Search() = %q, want %q", got, want)
	}
	if got, want := u.Href(false), "https://example.test/?a=3&c=4"; got != want {
		t.Errorf("Href() = %q, want %q", got, want)
	}
}

func TestUrlSearchParams_CloneHasIndependentAssociatedObject(t *testing.T) {
	original, err := testParser.Parse("https://example.test/?a=1")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	originalParams := original.SearchParams()

	clone := original.Clone()
	cloneParams := clone.SearchParams()
	if originalParams == cloneParams {
		t.Fatal("Clone() reused the original SearchParams object")
	}

	cloneParams.Set("a", "2")
	if got, want := clone.Href(false), "https://example.test/?a=2"; got != want {
		t.Errorf("clone Href() = %q, want %q", got, want)
	}
	if got, want := original.Href(false), "https://example.test/?a=1"; got != want {
		t.Errorf("original Href() = %q, want %q", got, want)
	}

	originalParams.Append("b", "3")
	if got, want := original.Href(false), "https://example.test/?a=1&b=3"; got != want {
		t.Errorf("original Href() after Append() = %q, want %q", got, want)
	}
	if got, want := clone.Href(false), "https://example.test/?a=2"; got != want {
		t.Errorf("clone Href() after original mutation = %q, want %q", got, want)
	}
}

func TestUrlSearchParams_Get(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		key     string
		want    string
		wantHas bool
	}{
		{"1", "http://example.com?foo=bar", "foo", "bar", true},
		{"2", "http://example.com?foo=bar&foo2=bar2", "foo", "bar", true},
		{"3", "http://example.com?foo=bar&foo=bar2", "foo", "bar", true},
		{"4", "http://example.com?foo=bar2&foo=bar", "foo", "bar2", true},
		{"5", "http://example.com?foo=bar2&foo=bar", "foo2", "", false},
		{"6", "http://example.com?foo=bar2&foo2", "foo2", "", true},
		{"7", "http://example.com/", "foo2", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := testParser.Parse(tt.url)
			s := url.SearchParams()
			if got := s.Get(tt.key); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
			if got := s.Has(tt.key); got != tt.wantHas {
				t.Errorf("Get() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestUrlSearchParams_GetAll(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		sort           bool
		want           []string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", false, []string{"bar"}, "foo=bar"},
		{"2", "http://example.com?foo=bar&foo2=bar2", "foo", false, []string{"bar"}, "foo=bar&foo2=bar2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo", false, []string{"bar"}, "foo2=bar2&foo=bar"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo", false, []string{"bar", "bar2"}, "foo=bar&foo=bar2"},
		{"5", "http://example.com?foo=bar2&foo=bar", "foo", false, []string{"bar2", "bar"}, "foo=bar2&foo=bar"},
		{"6", "http://example.com?foo=bar", "foo", true, []string{"bar"}, "foo=bar"},
		{"7", "http://example.com?foo=bar&foo2=bar2", "foo", true, []string{"bar"}, "foo=bar&foo2=bar2"},
		{"8", "http://example.com?foo2=bar2&foo=bar", "foo", true, []string{"bar"}, "foo=bar&foo2=bar2"},
		{"9", "http://example.com?foo=bar&foo=bar2", "foo", true, []string{"bar", "bar2"}, "foo=bar&foo=bar2"},
		{"10", "http://example.com?foo=bar2&foo=bar", "foo", true, []string{"bar2", "bar"}, "foo=bar2&foo=bar"},
		{"11", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo", true, []string{"bar2", "bar"}, "foo=bar2&foo=bar&xyz=aaa&xyz=aaa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := testParser.Parse(tt.url)
			s := url.SearchParams()
			if tt.sort {
				s.Sort()
			}
			if got := s.GetAll(tt.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_Append(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		value          string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", "bar2", "foo=bar&foo=bar2"},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "abc", "xyz", "foo=bar&foo+2=bar+2&abc=xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := testParser.Parse(tt.url)
			s := url.SearchParams()
			s.Append(tt.key, tt.value)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_Delete(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", ""},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "foo", "foo+2=bar+2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo", "foo2=bar2"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo", ""},
		{"5", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo", "xyz=aaa&xyz=aaa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := testParser.Parse(tt.url)
			s := url.SearchParams()
			s.Delete(tt.key)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_Set(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		value          string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", "bar2", "foo=bar2"},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "foo", "xyz", "foo=xyz&foo+2=bar+2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo", "xyz", "foo2=bar2&foo=xyz"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo", "xyz", "foo=xyz"},
		{"5", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo", "xyz", "xyz=aaa&foo=xyz&xyz=aaa"},
		{"6", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo2", "xyz", "xyz=aaa&foo=bar2&xyz=aaa&foo=bar&foo2=xyz"},
		{"7", "http://example.com?foo=bar&foo=fuzz&foo=barfuzz", "foo", "xyz", "foo=xyz"},
		{"8", "http://example.com?foo=bar&foo=fuzz&foo=barfuzz", "foo", "", "foo="},
		{"9", "http://example.com", "foo", "", "foo="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := testParser.Parse(tt.url)
			s := url.SearchParams()
			s.Set(tt.key, tt.value)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_String(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo=bar"},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "foo=bar&foo+2=bar+2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo2=bar2&foo=bar"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo=bar&foo=bar2"},
		{"5", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "xyz=aaa&foo=bar2&xyz=aaa&foo=bar"},
		{"6", "http://example.com?foo=bar&foo=fuzz&foo=barfuzz", "foo=bar&foo=fuzz&foo=barfuzz"},
		{"7", "http://example.com?foo", "foo="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := testParser.Parse(tt.url)
			s := url.SearchParams()
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("got %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

// Go ports of web-platform-tests URLSearchParams cases.
// Copyright web-platform-tests contributors. BSD-3-Clause; see
// ../testdata/wpt-searchparams/LICENSE.md and README.md for sources and coverage.

// The Go API has no standalone constructor. Initialize its form tuples directly
// so constructor-string cases do not undergo URL preprocessing (e.g. NULs).
func wptSearchParams(input string) *SearchParams {
	p := &SearchParams{url: newUrl()}
	p.values.Init(input)
	return p
}

func wptURL(t *testing.T, input string) *Url {
	t.Helper()
	u, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func wptEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type wptFormVector struct {
	Input  string
	Output [][2]string
}

func wptFormVectors(t *testing.T, file string) []wptFormVector {
	t.Helper()
	data, err := os.ReadFile("../testdata/wpt-searchparams/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var cases []wptFormVector
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("empty WPT fixture")
	}
	return cases
}

func wptFormPairs(t *testing.T, p *SearchParams, want [][2]string) {
	t.Helper()
	got := make([][2]string, len(p.values))
	for i, pair := range p.values {
		got[i] = [2]string{pair.Name, pair.Value}
	}
	if !slices.Equal(got, want) {
		t.Errorf("form pairs = %q, want %q", got, want)
	}
}

func TestWPTSearchParamsParsing(t *testing.T) {
	// urlencoded-parser.any.js: all 35 URLSearchParams input/output vectors.
	for _, tc := range wptFormVectors(t, "urlencoded-parser.json") {
		t.Run(tc.Input, func(t *testing.T) {
			wptFormPairs(t, wptSearchParams(tc.Input), tc.Output)
		})
	}
}

func TestWPTSearchParamsSort(t *testing.T) {
	// urlsearchparams-sort.any.js: standalone and associated-URL variants.
	for _, tc := range wptFormVectors(t, "urlsearchparams-sort.json") {
		t.Run(tc.Input, func(t *testing.T) {
			p := wptSearchParams(tc.Input)
			p.Sort()
			wptFormPairs(t, p, tc.Output)
			u := wptURL(t, "https://example/?"+tc.Input)
			u.SearchParams().Sort()
			wptFormPairs(t, wptSearchParams(u.Query()), tc.Output)
		})
	}
	t.Run("Sorting non-existent params removes ? from URL", func(t *testing.T) {
		u := wptURL(t, "http://example.com/?")
		u.SearchParams().Sort()
		wptEqual(t, u.String(), "http://example.com/")
		wptEqual(t, u.Search(), "")
	})
}

func TestWPTSearchParamsConstructorStrings(t *testing.T) {
	// urlsearchparams-constructor.any.js: string parsing, not JS constructors.
	for _, tc := range []struct{ name, input, key, value string }{
		{"Parse + value", "a=b+c", "a", "b c"},
		{"Parse + name", "a+b=c", "a b", "c"},
		{"Parse space value", "a=b c", "a", "b c"},
		{"Parse space name", "a b=c", "a b", "c"},
		{"Parse %20 value", "a=b%20c", "a", "b c"},
		{"Parse %20 name", "a%20b=c", "a b", "c"},
		{"Parse NUL value", "a=b\x00c", "a", "b\x00c"},
		{"Parse NUL name", "a\x00b=c", "a\x00b", "c"},
		{"Parse %00 value", "a=b%00c", "a", "b\x00c"},
		{"Parse %00 name", "a%00b=c", "a\x00b", "c"},
		{"Parse composition value", "a=b⎄", "a", "b⎄"},
		{"Parse composition name", "a⎄b=c", "a⎄b", "c"},
		{"Parse encoded composition value", "a=b%e2%8e%84", "a", "b⎄"},
		{"Parse encoded composition name", "a%e2%8e%84b=c", "a⎄b", "c"},
		{"Parse supplementary value", "a=b💩c", "a", "b💩c"},
		{"Parse supplementary name", "a💩b=c", "a💩b", "c"},
		{"Parse encoded supplementary value", "a=b%f0%9f%92%a9c", "a", "b💩c"},
		{"Parse encoded supplementary name", "a%f0%9f%92%a9b=c", "a💩b", "c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := wptSearchParams(tc.input)
			if !p.Has(tc.key) {
				t.Errorf("missing name %q", tc.key)
			}
			wptEqual(t, p.Get(tc.key), tc.value)
		})
	}
	t.Run("Parse encoded +", func(t *testing.T) {
		p := wptSearchParams("")
		p.Set("query", "+15555555555")
		wptEqual(t, p.String(), "query=%2B15555555555")
		wptEqual(t, p.Get("query"), "+15555555555")
		wptEqual(t, wptSearchParams(p.String()).Get("query"), "+15555555555")
	})
	t.Run("URLSearchParams constructor, string", func(t *testing.T) {
		p := wptSearchParams("&a&&& &&&&&a+b=& c&m%c3%b8%c3%b8")
		for _, key := range []string{"a", "a b", " ", " c", "møø"} {
			if !p.Has(key) {
				t.Errorf("missing name %q", key)
			}
		}
		if p.Has("c") {
			t.Error("unexpected trimmed name c")
		}
	})
}

func TestWPTSearchParamsAccessors(t *testing.T) {
	// urlsearchparams-{get,getall,has}.any.js. Go Get uses "" for absence;
	// pair it with Has to distinguish absent keys from empty values.
	for _, tc := range []struct {
		input, key string
		values     []string
	}{
		{"a=b&c=d", "a", []string{"b"}},
		{"a=b&c=d", "c", []string{"d"}},
		{"a=b&c=d", "e", nil},
		{"a=b&c=d&a=e", "a", []string{"b", "e"}},
		{"=b&c=d", "", []string{"b"}},
		{"a=&c=d&a=e", "a", []string{"", "e"}},
		{"first=second&third&&", "first", []string{"second"}},
		{"first=second&third&&", "third", []string{""}},
		{"first=second&third&&", "fourth", nil},
		{"null=a", "null", []string{"a"}},
	} {
		t.Run(tc.input+"/"+tc.key, func(t *testing.T) {
			p := wptSearchParams(tc.input)
			if p.Has(tc.key) != (len(tc.values) != 0) || !slices.Equal(p.GetAll(tc.key), tc.values) {
				t.Errorf("Has/GetAll(%q) = %v/%q, want %q", tc.key, p.Has(tc.key), p.GetAll(tc.key), tc.values)
			}
			want := ""
			if len(tc.values) != 0 {
				want = tc.values[0]
			}
			wptEqual(t, p.Get(tc.key), want)
		})
	}
}

func TestWPTSearchParamsAppend(t *testing.T) {
	// urlsearchparams-append.any.js. JS null/numbers are supplied as strings.
	for _, tc := range []struct{ name, key, value, want string }{
		{"Append same name", "a", "b", "a=b&a=b"},
		{"Append empty strings", "", "", "=&="},
		{"Append null", "null", "null", "null=null&null=null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := wptSearchParams("")
			p.Append(tc.key, tc.value)
			p.Append(tc.key, tc.value)
			wptEqual(t, p.String(), tc.want)
		})
	}
	t.Run("Append multiple", func(t *testing.T) {
		p := wptSearchParams("")
		p.Append("first", "1")
		p.Append("second", "2")
		p.Append("third", "")
		p.Append("first", "10")
		p.Append("first", "10")
		wptEqual(t, p.Get("first"), "1")
		wptEqual(t, p.Get("second"), "2")
		wptEqual(t, p.Get("third"), "")
		if !p.Has("first") || !p.Has("third") {
			t.Error("appended keys missing")
		}
	})
}

func TestWPTSearchParamsSet(t *testing.T) {
	// urlsearchparams-set.any.js and getAll() multiples.
	p := wptSearchParams("a=b&c=d&a=e")
	p.Set("a", "B")
	wptEqual(t, p.String(), "a=B&c=d")
	p.Set("e", "f")
	wptEqual(t, p.String(), "a=B&c=d&e=f")
	p = wptSearchParams("a=1&a=2&a=3&a")
	if !slices.Equal(p.GetAll("a"), []string{"1", "2", "3", ""}) {
		t.Error("getAll() lost duplicate values")
	}
	p.Set("first", "4")
	wptEqual(t, p.Get("a"), "1")
	p.Set("a", "one")
	wptEqual(t, p.Get("a"), "one")
	if !p.Has("a") || !slices.Equal(p.GetAll("a"), []string{"one"}) {
		t.Error("Set did not replace duplicate values")
	}
}

func TestWPTSearchParamsDelete(t *testing.T) {
	// urlsearchparams-delete.any.js: single-argument deletion.
	for _, tc := range []struct{ input, key, want string }{
		{"a=b&c=d", "a", "c=d"},
		{"a=a&b=b&a=a&c=c", "a", "b=b&c=c"},
		{"a=a&=&b=b&c=c", "", "a=a&b=b&c=c"},
		{"a=a&null=null&b=b", "null", "a=a&b=b"},
		{"a=a&undefined=undefined&b=b", "undefined", "a=a&b=b"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			p := wptSearchParams(tc.input)
			p.Delete(tc.key)
			wptEqual(t, p.String(), tc.want)
		})
	}
	t.Run("Deleting appended multiple and has following delete", func(t *testing.T) {
		p := wptSearchParams("a=b&c=d&&")
		p.Append("first", "1")
		p.Append("first", "2")
		p.Delete("first")
		if p.Has("first") || !p.Has("a") || !p.Has("c") || p.Has("d") {
			t.Error("unexpected names after deleting appended duplicates")
		}
	})
	for _, input := range []string{"http://example.com/?param1&param2", "http://example.com/?"} {
		u := wptURL(t, input)
		u.SearchParams().Delete("param1")
		u.SearchParams().Delete("param2")
		wptEqual(t, u.String(), "http://example.com/")
		wptEqual(t, u.Search(), "")
	}
	for _, fragment := range []string{"", "#test"} {
		u := wptURL(t, "data:space    ?test"+fragment)
		u.SearchParams().Delete("test")
		wptEqual(t, u.Search(), "")
		wptEqual(t, u.Pathname(), "space   %20")
		wptEqual(t, u.String(), "data:space   %20"+fragment)
	}
}

func TestWPTSearchParamsStringifier(t *testing.T) {
	// urlsearchparams-stringifier.any.js: escaping on both sides of '='.
	for _, tc := range []struct{ name, value, encoded string }{
		{"space", "b c", "b+c"},
		{"+", "b+c", "b%2Bc"},
		{"=", "=", "%3D"},
		{"&", "&", "%26"},
		{"*-._", "*-._", "*-._"},
		{"%", "b%c", "b%25c"},
		{"NUL", "b\x00c", "b%00c"},
		{"supplementary", "b💩c", "b%F0%9F%92%A9c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := wptSearchParams("")
			p.Append("a", tc.value)
			wptEqual(t, p.String(), "a="+tc.encoded)
			p.Delete("a")
			p.Append(tc.value, "c")
			wptEqual(t, p.String(), tc.encoded+"=c")
		})
	}
	for _, tc := range []struct{ input, want string }{
		{"a=b&c=d&&e&&", "a=b&c=d&e="},
		{"a = b &a=b&c=d%20", "a+=+b+&a=b&c=d+"},
		{"a=&a=b", "a=&a=b"},
		{"id=0&value=%", "id=0&value=%25"},
		{"b=%2sf%2a", "b=%252sf*"},
		{"b=%2%2af%2a", "b=%252*f*"},
		{"b=%%2a", "b=%25*"},
	} {
		wptEqual(t, wptSearchParams(tc.input).String(), tc.want)
	}
	t.Run("Serialize empty name and value", func(t *testing.T) {
		p := wptSearchParams("")
		for _, tc := range []struct{ name, value, want string }{
			{"a", "", "a="}, {"a", "", "a=&a="}, {"", "b", "a=&a=&=b"},
			{"", "", "a=&a=&=b&="}, {"", "", "a=&a=&=b&=&="},
		} {
			p.Append(tc.name, tc.value)
			wptEqual(t, p.String(), tc.want)
		}
	})
	t.Run("URLSearchParams must not do newline normalization", func(t *testing.T) {
		u := wptURL(t, "http://www.example.com/")
		p := u.SearchParams()
		p.Append("a\nb", "c\rd")
		p.Append("e\n\rf", "g\r\nh")
		wptEqual(t, p.String(), "a%0Ab=c%0Dd&e%0A%0Df=g%0D%0Ah")
	})
}

func TestWPTSearchParamsURL(t *testing.T) {
	// url-searchparams.any.js and stringifier's associated-URL case.
	u := wptURL(t, "http://example.org/file?a=b&c=d")
	p := u.SearchParams()
	if p != u.SearchParams() {
		t.Fatal("SearchParams object identity changed")
	}
	wptEqual(t, p.String(), "a=b&c=d")
	for _, search := range []string{"e=f&g=h", "?e=f&g=h"} {
		u.SetSearch(search)
		wptEqual(t, u.Search(), "?e=f&g=h")
		wptEqual(t, p.String(), "e=f&g=h")
	}
	p.Append("i", " j ")
	wptEqual(t, u.Search(), "?e=f&g=h&i=+j+")
	wptEqual(t, p.Get("i"), " j ")
	p.Set("e", "updated")
	wptEqual(t, u.Search(), "?e=updated&g=h&i=+j+")
	u.SetSearch("")
	wptEqual(t, p.String(), "")
	wptEqual(t, u.Search(), "")
	u = wptURL(t, "http://example.org/file??a=b&c=d")
	wptEqual(t, u.Search(), "??a=b&c=d")
	wptEqual(t, u.SearchParams().String(), "%3Fa=b&c=d")
	u = wptURL(t, "http://www.example.com/?a=b,c")
	p = u.SearchParams()
	wptEqual(t, p.String(), "a=b%2Cc")
	wptEqual(t, u.String(), "http://www.example.com/?a=b,c")
	p.Append("x", "y")
	wptEqual(t, u.String(), "http://www.example.com/?a=b%2Cc&x=y")
	wptEqual(t, p.String(), "a=b%2Cc&x=y")
}

func TestWPTSearchParamsIteration(t *testing.T) {
	// urlsearchparams-foreach.any.js: basic traversal only. Iterate is a Go
	// mutation API, not the JS forEach/for-of live-iterator interface.
	p := wptSearchParams("a=1&b=2&c=3")
	var keys, values []string
	p.Iterate(func(pair *NameValuePair) {
		keys = append(keys, pair.Name)
		values = append(values, pair.Value)
	})
	if !slices.Equal(keys, []string{"a", "b", "c"}) || !slices.Equal(values, []string{"1", "2", "3"}) {
		t.Errorf("iteration = %q/%q", keys, values)
	}
	wptSearchParams("").Iterate(func(*NameValuePair) { t.Error("visited an empty list") })
}

func TestSearchParamsNonUTF8Decoding(t *testing.T) {
	// https://url.spec.whatwg.org/#concept-urlencoded-parser
	for _, tc := range []struct {
		name, query, key string
		values           []string
	}{
		{"legacy_byte", "q=%FC", "q", []string{"\ufffd"}},
		{"truncated_sequence", "q=%E2%82", "q", []string{"\ufffd"}},
		{"truncated_four_byte_sequence", "q=%F0%9F%92", "q", []string{"\ufffd"}},
		{"adjacent_invalid_bytes", "q=%FF%FE", "q", []string{"\ufffd\ufffd"}},
		{"overlong_sequence", "q=%E0%80%80", "q", []string{"\ufffd\ufffd\ufffd"}},
		{"surrogate", "q=%ED%A0%80", "q", []string{"\ufffd\ufffd\ufffd"}},
		{"above_unicode_limit", "q=%F4%90%80%80", "q", []string{"\ufffd\ufffd\ufffd\ufffd"}},
		{"ASCII_after_incomplete_sequence", "q=%E2%82A", "q", []string{"\ufffdA"}},
		{"invalid_then_valid", "q=%E2%82%C3%BC", "q", []string{"\ufffdü"}},
		{"decoded_name", "%FC=value", "\ufffd", []string{"value"}},
		{"equal_decoded_names", "%FF=first&%FE=second", "\ufffd", []string{"first", "second"}},
		{"valid_UTF8", "q=%C3%BC", "q", []string{"ü"}},
		{"existing_replacement", "q=%EF%BF%BD", "q", []string{"\ufffd"}},
		{"BOM_preserved", "q=%EF%BB%BF", "q", []string{"\ufeff"}},
		{"BOM_in_name_preserved", "%EF%BB%BF=value", "\ufeff", []string{"value"}},
		{"percent_decoded_once", "q=%25FC", "q", []string{"%FC"}},
		{"plus_and_encoded_plus", "q=a+b%2Bc", "q", []string{"a b+c"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := "https://example.org/?" + tc.query
			u, err := Parse(input)
			if err != nil {
				t.Fatal(err)
			}
			params := u.SearchParams()
			if !params.Has(tc.key) || params.Get(tc.key) != tc.values[0] || !slices.Equal(params.GetAll(tc.key), tc.values) {
				t.Errorf("decoded key %q: Has=%v Get=%q GetAll=%q, want %q", tc.key, params.Has(tc.key), params.Get(tc.key), params.GetAll(tc.key), tc.values)
			}
			if u.String() != input {
				t.Errorf("reading form data changed URL to %q", u.String())
			}
		})
	}
}

func TestSearchParamsNonUTF8Synchronization(t *testing.T) {
	u, err := Parse("https://example.org/?q=initial")
	if err != nil {
		t.Fatal(err)
	}
	p := u.SearchParams()
	u.SetSearch("q=%E2%82&%FF=first&%FE=second")
	if u.SearchParams() != p || p.Get("q") != "\ufffd" || !slices.Equal(p.GetAll("\ufffd"), []string{"first", "second"}) {
		t.Fatalf("SetSearch did not refresh decoded form fields: %q", p.String())
	}
	want := "q=%EF%BF%BD&%EF%BF%BD=first&%EF%BF%BD=second"
	if got := p.String(); got != want {
		t.Errorf("form serialization = %q, want %q", got, want)
	}
	if got := u.Query(); got != "q=%E2%82&%FF=first&%FE=second" {
		t.Errorf("read-only form access changed query to %q", got)
	}
	p.Append("probe", "x")
	if got := u.Query(); got != want+"&probe=x" {
		t.Errorf("mutation serialized query = %q, want %q", got, want+"&probe=x")
	}
}

func TestSearchParamsFormEscaping(t *testing.T) {
	// https://url.spec.whatwg.org/#concept-urlencoded-serializer
	// Append already-decoded, valid strings to isolate encoding from parsing.
	for _, tc := range []struct{ name, value, want string }{
		{"plus", "+", "q=%2B"},
		{"ampersand", "&", "q=%26"},
		{"percent", "%", "q=%25"},
		{"equals", "=", "q=%3D"},
		{"literal_escape", "%FC", "q=%25FC"},
		{"space", " ", "q=+"},
		{"controls", "\x00\t\x7f", "q=%00%09%7F"},
		{"form_punctuation", "!'()~", "q=%21%27%28%29%7E"},
		{"component_punctuation", "$,:;/?@[]\\^`{|}", "q=%24%2C%3A%3B%2F%3F%40%5B%5D%5C%5E%60%7B%7C%7D"},
		{"safe_characters", "AZaz09*-._", "q=AZaz09*-._"},
		{"valid_UTF8", "ü", "q=%C3%BC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Parse("https://example.org/")
			if err != nil {
				t.Fatal(err)
			}
			params := u.SearchParams()
			params.Append("q", tc.value)
			if got := params.String(); got != tc.want {
				t.Errorf("form serialization = %q, want %q", got, tc.want)
			}
			if got := u.Query(); got != tc.want {
				t.Errorf("associated URL query = %q, want %q", got, tc.want)
			}
			// Names use the same form escaping rules; delimiter characters in
			// a name must not create additional fields or change its value.
			u.SetSearch("")
			params.Append(tc.value, "value")
			wantName := strings.TrimPrefix(tc.want, "q=") + "=value"
			if got := u.Query(); got != wantName {
				t.Errorf("escaped name query = %q, want %q", got, wantName)
			}
			again, err := Parse(u.String())
			if err != nil {
				t.Fatal(err)
			}
			if got := again.SearchParams().Get(tc.value); got != "value" {
				t.Errorf("reparsed value for name %q = %q, want value", tc.value, got)
			}
		})
	}
}

func TestSearchParamsUTF16Sorting(t *testing.T) {
	// https://url.spec.whatwg.org/#dom-urlsearchparams-sort
	// Literal expectations cover UTF-16 ordering, prefixes, and stable decoded names.
	for _, tc := range []struct{ name, query, want string }{
		{"stable_equal_names", "b=last&a=first&a=second", "a=first&a=second&b=last"},
		{"UTF16_code_units", "%EF%BF%BF=1&%F0%90%80%80=2", "%F0%90%80%80=2&%EF%BF%BF=1"},
		{"BMP_boundary", "%EE%80%80=upper&%F0%90%80%80=supplementary&%ED%9F%BF=lower", "%ED%9F%BF=lower&%F0%90%80%80=supplementary&%EE%80%80=upper"},
		{"surrogate_pair_order", "%F0%90%90%80=3&%F0%90%80%81=2&%F0%90%80%80=1", "%F0%90%80%80=1&%F0%90%80%81=2&%F0%90%90%80=3"},
		{"empty_name_and_prefixes", "ab=long&a=short&=empty", "=empty&a=short&ab=long"},
		{"supplementary_prefix", "%F0%90%80%80a=long&%F0%90%80%80=short", "%F0%90%80%80=short&%F0%90%80%80a=long"},
		{"stable_supplementary_names", "%EF%BF%BF=x&%F0%90%80%80=z&%F0%90%80%80=a", "%F0%90%80%80=z&%F0%90%80%80=a&%EF%BF%BF=x"},
		{"equal_decoded_names", "%61=z&a=a", "a=z&a=a"},
		{"equal_replacement_names", "%FF=z&%FE=a", "%EF%BF%BD=z&%EF%BF%BD=a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Parse("https://example.org/?" + tc.query)
			if err != nil {
				t.Fatal(err)
			}
			u.SearchParams().Sort()
			if got := u.Query(); got != tc.want {
				t.Errorf("sorted query = %q, want %q", got, tc.want)
			}
			u.SearchParams().Sort()
			if got := u.Query(); got != tc.want {
				t.Errorf("sorting again = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSearchParamsSortAbsoluteUTF16(t *testing.T) {
	for _, tc := range []struct{ name, query, want string }{
		{"names", "%EF%BF%BF=a&%F0%90%80%80=z", "%F0%90%80%80=z&%EF%BF%BF=a"},
		{"values", "q=%EF%BF%BF&q=%F0%90%80%80", "q=%F0%90%80%80&q=%EF%BF%BF"},
		{"names_before_values", "%EF%BF%BF=a&%F0%90%80%80=z&%F0%90%80%80=a", "%F0%90%80%80=a&%F0%90%80%80=z&%EF%BF%BF=a"},
		{"prefixes_and_duplicates", "b=&a=ab&a=a&a=a&a=", "a=&a=a&a=a&a=ab&b="},
		{"decoded_names", "%FF=z&%FE=a", "%EF%BF%BD=a&%EF%BF%BD=z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Parse("https://example.org/?" + tc.query)
			if err != nil {
				t.Fatal(err)
			}
			u.SearchParams().SortAbsolute()
			if got := u.Query(); got != tc.want {
				t.Errorf("absolute sort = %q, want %q", got, tc.want)
			}
		})
	}
}
