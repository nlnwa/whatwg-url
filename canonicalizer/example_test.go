/*
 * Copyright 2021 National Library of Norway.
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

package canonicalizer_test

import (
	"fmt"
	"github.com/nlnwa/whatwg-url/canonicalizer"
)

func ExampleNew() {
	c := canonicalizer.New(canonicalizer.WithRemoveUserInfo(), canonicalizer.WithRemoveFragment())
	u, err := c.Parse("http://user@example.com/a//d?b#c")
	if err == nil {
		fmt.Println(u)
	}
	// Output: http://example.com/a//d?b
}

func ExampleGoogleSafeBrowsing() {
	u, err := canonicalizer.GoogleSafeBrowsing.Parse("http://user@example.com/a//d?b#c")
	if err == nil {
		fmt.Println(u)
	}
	// Output: http://user@example.com/a/d?b
}
