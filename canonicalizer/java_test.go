// Copyright (C) 2016-2020 National Library of Australia
// Copyright (C) 2016-2018 Internet Archive
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Adapted from urlcanon's IpAddressesTest.java into URL-level assertions.
// See README.md for the pinned source and adaptation details.

package canonicalizer_test

import (
	"testing"

	canon "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestSemanticJavaIPv6Cases(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"1:0:0::", "1::"},
		{"1:0:2::", "1:0:2::"},
		{"ffff::192.0.2.128", "ffff::c000:280"},
		{"ffff:02::3:192.0.2.128", "ffff:2::3:c000:280"},
		{"ffff::0.0.2.3", "ffff::203"},
		{"1::2", "1::2"},
		{"1:0:0::2", "1::2"},
		{"1:0:2:0:0:3:4:5", "1:0:2::3:4:5"},
		{"1:0:0:2:0:3:4:5", "1::2:0:3:4:5"},
		{"1:0:2:0:3:0:4:0", "1:0:2:0:3:0:4:0"},
		{"::ffff:192.0.2.128", "::ffff:c000:280"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			u := checkProfile(t, canon.Semantic(), "http://["+tc.input+"]/", "http://["+tc.want+"]/")
			if !u.IsIPv6() {
				t.Fatal("canonicalization lost IPv6 metadata")
			}
		})
	}
	for _, input := range []string{"bogus", "1:2:3:4:5:6:7:8:9", "1:2", "1:2:3", "1::2::3"} {
		t.Run(input, func(t *testing.T) {
			if _, err := canon.Semantic().WithParser(url.NewParser()).Parse("http://[" + input + "]/"); err == nil {
				t.Fatal("invalid IPv6 address was accepted")
			}
		})
	}
}
