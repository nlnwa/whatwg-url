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
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
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
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	empty := args{}
	var testNum int
	for _, tt := range tests {
		if tt != empty {
			testNum++

			t.Run(strconv.Itoa(testNum), func(t *testing.T) {
				got, err := ParseRef(tt.Base, tt.Input)
				if (err != nil) != tt.Failure {
					t.Logf("Base: %v, Input: %v, Expected: %v, GOT: %v", tt.Base, tt.Input, tt.Href, got)
					t.Errorf("ParseRef() error = %v, wantErr %v", err, tt.Failure)
					return
				}
				if err != nil && tt.Failure {
					return
				}
				if err != nil {
					t.Logf("Base: %v, Input: %v, Expected: %v, GOT: %v", tt.Base, tt.Input, tt.Href, got)
					t.Errorf("ParseRef() error = %v, wantErr %v", err, tt.Failure)
					return
				}

				if got.String() != tt.Href {
					t.Errorf("String() got = %v, want %v", got.String(), tt.Href)
				}

				if got.protocol+":" != tt.Protocol {
					fmt.Printf("%T %T %v %v\n", got.protocol, tt.Protocol, got.protocol, tt.Protocol)
					t.Errorf("Scheme got = %v, want %v", got.protocol, tt.Protocol)
				}

				if got.username != tt.Username {
					t.Errorf("User.Username() got = %v, want %v", got.username, tt.Username)
				}

				if got.password != tt.Password {
					t.Errorf("User.Password() got = %v, want %v", got.password, tt.Password)
				}

				if got.Host() != tt.Host {
					t.Errorf("Host got = '%v', want '%v'", got.Host(), tt.Host)
				}

				if got.Hostname() != tt.Hostname {
					t.Errorf("Hostname() got = %v, want %v", got.Hostname(), tt.Hostname)
				}

				if got.Port() != tt.Port {
					t.Errorf("Port() got = %v, want %v", got.port, tt.Port)
				}

				if got.Pathname() != tt.Pathname {
					t.Errorf("Path got = %v, want %v", got.Pathname(), tt.Pathname)
				}

				if got.Search() != tt.Search {
					t.Errorf("RawQuery got = %v, want %v", got.Search(), tt.Search)
				}

				if got.Hash() != tt.Hash {
					t.Errorf("Fragment got = %v, want %v", got.Hash(), tt.Hash)
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
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
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
			url, _ := Parse(tt.url)
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
			url, _ := Parse(tt.url)
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
			url, _ := Parse(tt.url)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			s.Set(tt.key, tt.value)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}
