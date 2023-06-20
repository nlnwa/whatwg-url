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

func TestParse(t *testing.T) {
	type args struct {
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

	var tests []args

	jsonFile, err := os.Open("../testdata/urltestdata.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

	empty := args{}
	var testNum int
	for _, tt := range tests {
		if tt != empty {
			testNum++

			t.Run(strconv.Itoa(testNum), func(t *testing.T) {
				got, err := ParseRef(tt.Base, tt.Input)
				if (err != nil) != tt.Failure {
					t.Errorf("ParseRef(%v, %v) = '%v', error = '%v', wantErr %v", tt.Base, tt.Input, got, err, tt.Failure)
					return
				}
				if err != nil && tt.Failure {
					return
				}
				if err != nil {
					t.Logf("Base: '%v', Input: '%v', Expected: '%v', GOT: '%v'", tt.Base, tt.Input, tt.Href, got)
					t.Errorf("ParseRef(%v, %v) error = '%v', wantErr %v", tt.Base, tt.Input, err, tt.Failure)
					return
				}

				if got.String() != tt.Href {
					t.Logf("Base: '%v', Input: '%v'", tt.Base, tt.Input)
					t.Errorf("String() got = '%v', want '%v'", got.String(), tt.Href)
				}

				if got.scheme+":" != tt.Protocol {
					t.Errorf("Scheme got = '%v', want '%v'", got.scheme, tt.Protocol)
				}

				if got.username != tt.Username {
					t.Errorf("User.Username() got = '%v', want '%v'", got.username, tt.Username)
				}

				if got.password != tt.Password {
					t.Errorf("User.Password() got = '%v', want '%v'", got.password, tt.Password)
				}

				if got.Host() != tt.Host {
					t.Errorf("Host got = '%v', want '%v'", got.Host(), tt.Host)
				}

				if got.Hostname() != tt.Hostname {
					t.Errorf("Hostname() got = '%v', want '%v'", got.Hostname(), tt.Hostname)
				}

				if got.Port() != tt.Port {
					t.Errorf("Port() got = '%v', want '%v'", got.port, tt.Port)
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

				reparsed, err := Parse(got.String())
				if err != nil {
					t.Errorf("Parse() error = '%v'", err)
					return
				}
				if got.String() != reparsed.String() {
					t.Errorf("Reparsing expected same result got = '%v', want '%v'", reparsed.String(), got.String())
				}
			})
		}
	}
}

var result string

func BenchmarkParse(b *testing.B) {
	type args struct {
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

	var tests []args

	jsonFile, err := os.Open("../testdata/urltestdata.json")
	if err != nil {
		b.Error(err)
	}
	defer jsonFile.Close()
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

	empty := args{}
	for testNum, tt := range tests {
		if tt != empty {
			b.Run(strconv.Itoa(testNum), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					got, err := ParseRef(tt.Base, tt.Input)
					if err == nil {
						result = got.String()
					}
				}
			})
		}
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
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Parse(buf.String())
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
			for i := 0; i < b.N; i++ {
				_, _ = Parse(buf.String())
			}
		})
	}
}
