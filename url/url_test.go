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
	"io"
	"os"
	"strconv"
	"testing"
)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

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

func TestUrl_Getters(t *testing.T) {
	tests := []struct {
		name        string
		inputUrl    string
		href        string
		protocol    string
		scheme      string
		username    string
		password    string
		host        string
		port        string
		decodedPort int
		path        string
		search      string
		query       string
		hash        string
		fragment    string
	}{
		{"1",
			"HTTP://u:p@example.com:80/foo?q=1#bar",
			"http://u:p@example.com/foo?q=1#bar",
			"http:",
			"http",
			"u",
			"p",
			"example.com",
			"",
			80,
			"/foo",
			"?q=1",
			"q=1",
			"#bar",
			"bar",
		},
		{"2",
			"HTTP://u:p@example.com/foo?q=1#bar",
			"http://u:p@example.com/foo?q=1#bar",
			"http:",
			"http",
			"u",
			"p",
			"example.com",
			"",
			80,
			"/foo",
			"?q=1",
			"q=1",
			"#bar",
			"bar",
		},
		{"3",
			"HTTP://u:p@example.com:8080/foo?q=1#bar",
			"http://u:p@example.com:8080/foo?q=1#bar",
			"http:",
			"http",
			"u",
			"p",
			"example.com",
			"8080",
			8080,
			"/foo",
			"?q=1",
			"q=1",
			"#bar",
			"bar",
		},
		{"4",
			"HTTPs://example.com",
			"https://example.com/",
			"https:",
			"https",
			"",
			"",
			"example.com",
			"",
			443,
			"/",
			"",
			"",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := Parse(tt.inputUrl)
			if got := u.Href(false); got != tt.href {
				t.Errorf("Href() = %v, want %v", got, tt.href)
			}
			if got := u.Protocol(); got != tt.protocol {
				t.Errorf("Protocol() = %v, want %v", got, tt.protocol)
			}
			if got := u.Scheme(); got != tt.scheme {
				t.Errorf("Scheme() = %v, want %v", got, tt.scheme)
			}
			if got := u.Username(); got != tt.username {
				t.Errorf("Username() = %v, want %v", got, tt.username)
			}
			if got := u.Password(); got != tt.password {
				t.Errorf("Password() = %v, want %v", got, tt.password)
			}
			if got := u.Hostname(); got != tt.host {
				t.Errorf("Hostname() = %v, want %v", got, tt.host)
			}
			if got := u.Port(); got != tt.port {
				t.Errorf("Port() = %v, want %v", got, tt.port)
			}
			if got := u.DecodedPort(); got != tt.decodedPort {
				t.Errorf("DecodedPort() = %v, want %v", got, tt.decodedPort)
			}
			if got := u.Pathname(); got != tt.path {
				t.Errorf("Pathname() = %v, want %v", got, tt.path)
			}
			if got := u.Search(); got != tt.search {
				t.Errorf("Search() = %v, want %v", got, tt.search)
			}
			if got := u.Query(); got != tt.query {
				t.Errorf("Query() = %v, want %v", got, tt.query)
			}
			if got := u.Hash(); got != tt.hash {
				t.Errorf("Hash() = %v, want %v", got, tt.hash)
			}
			if got := u.Fragment(); got != tt.fragment {
				t.Errorf("Fragment() = %v, want %v", got, tt.fragment)
			}
		})
	}
}
