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
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
)

func ExampleNew() {
	c := canonicalizer.New(canonicalizer.RemoveUserInfo, canonicalizer.RemoveFragment)
	u, err := c.Parse("http://user@example.com/a//d?b#c")
	if err == nil {
		fmt.Println(u)
	}
	// Output: http://example.com/a//d?b
}

func ExampleGoogleSafeBrowsing() {
	u, err := canonicalizer.GoogleSafeBrowsing().Parse("http://user@example.com/a//d?b#c")
	if err == nil {
		fmt.Println(u)
	}
	// Output: http://user@example.com/a/d?b
}

func ExampleProfile() {
	profile := canonicalizer.New(
		canonicalizer.RemoveUserInfo,
		canonicalizer.RemoveFragment,
		canonicalizer.SortQuery,
	).WithDefaultScheme("https")
	u, err := profile.Parse("user@example.org/path?b=2&a=1#fragment")
	if err != nil {
		panic(err)
	}
	fmt.Println(u)
	// Output: https://example.org/path?a=1&b=2
}

func ExampleIA() {
	filters := canonicalizer.DefaultSessionQueryFilters()
	delete(filters, "phpsessid")
	filters["tracking"] = func(name, _ string, _ bool) bool {
		return !strings.HasPrefix(name, "utm_")
	}
	profile := canonicalizer.IA(canonicalizer.WithHostMassage(false), canonicalizer.WithQueryFilters(filters))
	u, err := profile.Parse("http://www.example.org/Path?utm_source=x&CFID=1&PHPSESSID=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		panic(err)
	}
	fmt.Println(u)
	// Output: http://www.example.org/path?phpsessid=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
}

func ExampleFilterQuery() {
	profile := canonicalizer.New(
		canonicalizer.MassageHost,
		canonicalizer.FilterQuery(func(name, _ string, _ bool) bool {
			return name != "tracking"
		}),
		canonicalizer.SortQuery,
	)
	u, err := profile.Parse("https://www.example.org/?b=2&tracking=x&a=1")
	if err != nil {
		panic(err)
	}
	fmt.Println(u)
	// Output: https://example.org/?a=1&b=2
}
