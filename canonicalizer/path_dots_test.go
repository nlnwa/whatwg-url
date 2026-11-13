// Copyright (C) 2016 National Library of Australia
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
// Adapted from urlcanon's ResolvePathDotsTest.java at revision
// 842afd4d84089cbbef87dd3286edaf053caedb0a, using its unchanged path_dots.json.

package canonicalizer

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestURLCanonPathDots(t *testing.T) {
	file, err := os.Open("../testdata/iipc-urlcanon/path_dots.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	expect := func(want json.Token) {
		t.Helper()
		got, err := decoder.Token()
		if err != nil || got != want {
			t.Fatalf("fixture token = %v, %v; want %v", got, err, want)
		}
	}
	expect(json.Delim('{'))
	count := 0
	for decoder.More() {
		group, err := decoder.Token()
		if err != nil || group != "special" && group != "nonspecial" {
			t.Fatalf("unexpected fixture group: %v, %v", group, err)
		}
		expect(json.Delim('{'))
		// Token decoding retains repeated JSON object keys, as upstream's
		// Java streaming reader does. A map would silently drop cases.
		for decoder.More() {
			input, err := decoder.Token()
			if err != nil {
				t.Fatal(err)
			}
			var want string
			if err := decoder.Decode(&want); err != nil {
				t.Fatal(err)
			}
			count++
			t.Run(fmt.Sprintf("%s/%d", group, count), func(t *testing.T) {
				if got := resolvePathDots(input.(string), group == "special"); got != want {
					t.Fatalf("resolvePathDots(%q) = %q, want %q", input, got, want)
				}
			})
		}
		expect(json.Delim('}'))
	}
	expect(json.Delim('}'))
	if count != 932 {
		t.Fatalf("executed %d upstream cases, want all 932 including duplicates", count)
	}
}
