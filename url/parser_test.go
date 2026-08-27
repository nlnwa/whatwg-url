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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

var testParser = newDefaultParser()

var benchmarkURLSink *Url
var benchmarkPathStringSink string

const urlTestData = "../testdata/urltestdata.json"

type urlTest struct {
	Input    string
	Base     string
	Href     string
	Origin   string
	Protocol string
	Username string
	Password string
	Host     string
	Hostname string
	Port     string
	Pathname string
	Search   string
	Hash     string
	Failure  bool
}

func loadURLTestData() ([]urlTest, error) {
	var tests []urlTest

	jsonFile, err := os.Open(urlTestData)
	if err != nil {
		return nil, err
	}
	defer func() { _ = jsonFile.Close() }()
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

	var empty urlTest
	n := 0
	for _, val := range tests {
		if val == empty {
			continue
		}
		tests[n] = val
		n++
	}
	return tests[:n], nil
}

func TestParse(t *testing.T) {
	tests, err := loadURLTestData()
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	for testNum, tt := range tests {
		t.Run(strconv.Itoa(testNum), func(t *testing.T) {
			var got *Url
			var err error
			var gotHref string

			if tt.Base == "" {
				got, err = testParser.Parse(tt.Input)
				if err != nil && tt.Failure {
					return
				}
				if err == nil && tt.Failure {
					t.Fatalf("Parse(%q) = %q, expected an error", tt.Input, got)
				}

				if err != nil {
					t.Fatalf("Parse(%q) = %q, unexpected error: %v", tt.Input, got, err)
				}
				gotHref = got.String()
				if gotHref != tt.Href {
					t.Errorf("String(%s) = %q, want %q", tt.Input, gotHref, tt.Href)
				}
			} else {
				got, err = testParser.ParseRef(tt.Base, tt.Input)
				if err != nil && tt.Failure {
					return
				}
				if err == nil && tt.Failure {
					t.Fatalf("ParseRef(%q, %q) = %q, expected an error", tt.Base, tt.Input, got)
				}
				if err != nil {
					t.Fatalf("ParseRef(%q, %q) = %q, unexpected error: %v", tt.Base, tt.Input, got, err)
				}
				gotHref = got.String()
				if gotHref != tt.Href {
					t.Errorf("String(%s, %s) = %q, want %q", tt.Base, tt.Input, gotHref, tt.Href)
				}
			}

			if got.Scheme()+":" != tt.Protocol {
				t.Errorf("Scheme() = %q, want %q", got.Scheme(), tt.Protocol)
			}

			if got.Username() != tt.Username {
				t.Errorf("User.Username() got = '%v', want '%v'", got.Username(), tt.Username)
			}

			if got.Password() != tt.Password {
				t.Errorf("User.Password() got = '%v', want '%v'", got.Password(), tt.Password)
			}

			if got.Host() != tt.Host {
				t.Errorf("Host got = '%v', want '%v'", got.Host(), tt.Host)
			}

			if got.Hostname() != tt.Hostname {
				t.Errorf("Hostname() got = '%v', want '%v'", got.Hostname(), tt.Hostname)
			}

			if got.Port() != tt.Port {
				t.Errorf("Port() got = '%v', want '%v'", got.Port(), tt.Port)
			}

			if got.Pathname() != tt.Pathname {
				t.Errorf("Path got = '%v', want '%v'", got.Pathname(), tt.Pathname)
			}

			if got.Search() != tt.Search {
				t.Errorf("RawQuery got = '%v', want '%v'", got.Search(), tt.Search)
			}

			if got.Hash() != tt.Hash {
				t.Errorf("Fragment got = '%v', want '%v'", got.Hash(), tt.Hash)
			}

			t.Log(tt.Base, tt.Input, "got", gotHref, "expected", tt.Href)
			reparsed, err := testParser.Parse(gotHref)
			if err != nil {
				t.Errorf("Parse() error = '%v'", err)
				return
			}
			expectedHref := reparsed.String()
			if gotHref != expectedHref {
				t.Errorf("Reparsing expected same href got = '%v', want '%v'", expectedHref, gotHref)
			}
		})
	}
}

func BenchmarkSilly(b *testing.B) {
	for b.Loop() {
		_, err := testParser.Parse("https://www.Googlé.com")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	tests, err := loadURLTestData()
	if err != nil {
		b.Fatalf("Failed to load test data: %v", err)
	}

	for testNum, tt := range tests {
		if tt.Base != "" {
			continue
		}
		b.Run(strconv.Itoa(testNum), func(b *testing.B) {
			for b.Loop() {
				got, err := testParser.Parse(tt.Input)
				if err == nil {
					_ = got.String()
				}
			}
		})
	}
}

func BenchmarkParseRef(b *testing.B) {
	tests, err := loadURLTestData()
	if err != nil {
		b.Fatalf("Failed to load test data: %v", err)
	}

	for testNum, tt := range tests {
		if tt.Base == "" {
			continue
		}
		b.Run(strconv.Itoa(testNum), func(b *testing.B) {
			for b.Loop() {
				got, err := testParser.ParseRef(tt.Base, tt.Input)
				if err == nil {
					_ = got.String()
				}
			}
		})
	}
}

func BenchmarkPathSegments(b *testing.B) {
	for _, segments := range []int{0, 1, 2, 4, 8, 16, 32} {
		input := "https://example.com" + strings.Repeat("/segment", segments)
		name := strconv.Itoa(segments)

		b.Run(name+"/parse", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				u, err := testParser.Parse(input)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkURLSink = u
			}
		})

		b.Run(name+"/parse-and-serialize", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				u, err := testParser.Parse(input)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkPathStringSink = u.Href(false)
			}
		})
	}
}

func BenchmarkIssue6(b *testing.B) {
	// https://github.com/nlnwa/whatwg-url/issues/6
	for i := 10; i <= 20; i++ {
		n := 1 << i
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var buf strings.Builder
			buf.Grow(n + 32)
			buf.WriteString("data:text/javascript,")
			for j := 0; j <= n; j++ {
				buf.WriteString("A")
			}
			input := buf.String()
			b.ResetTimer()
			for b.Loop() {
				_, _ = testParser.Parse(input)
			}
		})
	}
}

func BenchmarkIssue8(b *testing.B) {
	// https://github.com/nlnwa/whatwg-url/issues/8
	for i := 10; i <= 20; i++ {
		n := 1 << i
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var buf strings.Builder
			buf.Grow(n + 32)
			buf.WriteString("http://example.com/?foo=bar")
			for j := 0; j <= n; j++ {
				buf.WriteString("&foo=bar")
			}
			b.ResetTimer()
			for b.Loop() {
				_, _ = testParser.Parse(buf.String())
			}
		})
	}
}

func TestDecodedPort(t *testing.T) {
	u, err := testParser.Parse("http://example.com:8080")
	if err != nil {
		t.Errorf("Parse() error = '%v'", err)
		return
	}

	if u.port != 8080 {
		t.Errorf("decodedPort got = '%v', want '%v'", u.port, 8080)
	}

	relative, err := testParser.ParseRef(u.String(), "/")
	if err != nil {
		t.Errorf("ParseRef() error = '%v'", err)
		return
	}

	if u.port != relative.port {
		t.Errorf("decodedPort got = '%v', want '%v'", relative.port, u.port)
	}
}

func TestParseShouldNotModifyBaseUrl(t *testing.T) {
	rawUrl := "http://example.org/ns/foo?adc=ghi#frag"
	base, err := testParser.Parse(rawUrl)
	if err != nil {
		t.Fatal(err)
	}

	_, err = base.Parse("bar?abc=def#fragment")
	if err != nil {
		t.Fatal(err)
	}

	if base.String() != rawUrl {
		t.Errorf("base URL = %v, want %v", base.String(), rawUrl)
	}
}
