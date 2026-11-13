// Copyright (C) 2017-2018 Internet Archive
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
// Adapted from Brozzler's seed-scope assertions into Go tests of Semantic
// followed by query removal and SSURT serialization. See README.md for sources.

package canonicalizer_test

import (
	"testing"

	canon "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/surt"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestSemanticBrozzlerSeedScope(t *testing.T) {
	profile := canon.Semantic().Then(canon.RewriteQueryString(
		func(_ *url.Url, _ string, _ bool) (string, bool) { return "", false },
	))
	for _, tc := range []struct{ name, input, want string }{
		{"http seed", "http://example.com", "com,example,//http:/"},
		{"https seed", "https://example.org/", "org,example,//https:/"},
		{"path seed", "http://example.com/a/", "com,example,//http:/a/"},
		{"redirect destination", "http://example.com/b/", "com,example,//http:/b/"},
		{"seed without hashtag", "http://example.org/", "org,example,//http:/"},
		{"seed with hashtag", "http://example.org/#hash", "org,example,//http:/"},
		{"documented seed normalization", "https://www.EXAMPLE.com:443/foo//bar?a=b&c=d#fdiap", "com,example,www,//https:/foo/bar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := profile.Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := surt.SSURTURL(u, true)
			if err != nil || got != tc.want {
				t.Fatalf("seed scope = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
