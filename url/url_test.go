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
	"io/ioutil"
	"os"
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
					t.Errorf("ParseRef(%v, %v) error = %v, wantErr %v", tt.Base, tt.Input, err, tt.Failure)
					return
				}
				if err != nil && tt.Failure {
					return
				}
				if err != nil {
					t.Logf("Base: %v, Input: %v, Expected: %v, GOT: %v", tt.Base, tt.Input, tt.Href, got)
					t.Errorf("ParseRef(%v, %v) error = %v, wantErr %v", tt.Base, tt.Input, err, tt.Failure)
					return
				}

				if got.String() != tt.Href {
					t.Logf("Base: '%v', Input: '%v'", tt.Base, tt.Input)
					t.Errorf("String() got = '%v', want '%v'", got.String(), tt.Href)
				}

				if got.protocol+":" != tt.Protocol {
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

				reparsed, err := Parse(got.String())
				if err != nil {
					t.Errorf("Parse() error = %v", err)
					return
				}
				if got.String() != reparsed.String() {
					t.Errorf("Reparsing expected same result got = %v, want %v", reparsed.String(), got.String())
				}
			})
		}
	}
}

type setterTest struct {
	Comment   string
	Href      string
	New_value string
	Expected  map[string]string
}

type setterTestCollection struct {
	Protocol []setterTest
	Username []setterTest
	Password []setterTest
	Host     []setterTest
	Hostname []setterTest
	Port     []setterTest
	Pathname []setterTest
	Search   []setterTest
	Hash     []setterTest
}

func TestUrl_SetProtocol(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Protocol {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetProtocol(tt.New_value)

			if err != nil {
				t.Errorf("Input: '%v', New value: '%v', Parse() error = '%v'", tt.Href, tt.New_value, err)
				return
			}

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetUsername(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Username {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetUsername(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetPassword(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Password {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetPassword(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetHost(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Host {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetHost(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetHostname(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Hostname {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetHostname(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetPort(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Port {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetPort(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetPathname(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Pathname {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetPathname(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetSearch(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Search {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetSearch(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func TestUrl_SetHash(t *testing.T) {
	var tests setterTestCollection

	jsonFile, err := os.Open("../testdata/setters_tests.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	var testNum int
	for _, tt := range tests.Hash {
		testNum++

		t.Run(strconv.Itoa(testNum)+":"+tt.Comment, func(t *testing.T) {
			got, err := Parse(tt.Href)
			got.SetHash(tt.New_value)

			validateExpected(got, err, tt, t)
		})
	}
}

func validateExpected(got *Url, err error, expected setterTest, t *testing.T) {
	if err != nil {
		t.Errorf("Input: '%v', New value: '%v', Parse() error = '%v'", expected.Href, expected.New_value, err)
		return
	}

	if href, ok := expected.Expected["href"]; ok {
		if got.String() != href {
			t.Errorf("Input: '%v', New value: '%v', String() got = '%v', want '%v'",
				expected.Href, expected.New_value, got.String(), href)
		}
	}

	if protocol, ok := expected.Expected["protocol"]; ok {
		if got.Protocol() != protocol {
			t.Errorf("Scheme got = '%v', want '%v'", got.Protocol(), protocol)
		}
	}

	if username, ok := expected.Expected["username"]; ok {
		if got.Username() != username {
			t.Errorf("Username got = '%v', want '%v'", got.Username(), username)
		}
	}

	if password, ok := expected.Expected["password"]; ok {
		if got.Password() != password {
			t.Errorf("Password got = '%v', want '%v'", got.Password(), password)
		}
	}

	if host, ok := expected.Expected["host"]; ok {
		if got.Host() != host {
			t.Errorf("Host() got = '%v', want '%v'", got.Host(), host)
		}
	}

	if hostname, ok := expected.Expected["hostname"]; ok {
		if got.Hostname() != hostname {
			t.Errorf("Hostname() got = '%v', want '%v'", got.Hostname(), hostname)
		}
	}

	if port, ok := expected.Expected["port"]; ok {
		if got.Port() != port {
			t.Errorf("Port() got = '%v', want '%v'", got.Port(), port)
		}
	}

	if pathname, ok := expected.Expected["pathname"]; ok {
		if got.Pathname() != pathname {
			t.Errorf("Pathname() got = '%v', want '%v'", got.Pathname(), pathname)
		}
	}

	if search, ok := expected.Expected["search"]; ok {
		if got.Search() != search {
			t.Errorf("Search() got = '%v', want '%v'", got.Search(), search)
		}
	}

	if hash, ok := expected.Expected["hash"]; ok {
		if got.Hash() != hash {
			t.Errorf("Hash() got = '%v', want '%v'", got.Hash(), hash)
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
