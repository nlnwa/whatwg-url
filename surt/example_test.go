package surt_test

import (
	"fmt"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/surt"
)

func ExampleString() {
	// The key places reversed hostname labels before the path and query.
	value, _ := surt.String("http://archive.org/goo/?a=2&b&a=1")
	fmt.Println(value)
	// Output: org,archive)/goo?a=1&a=2&b
}

func ExampleString_withOptions() {
	value, _ := surt.String(
		"http://www.example.com/",
		surt.WithIncludeScheme(true),
		surt.WithTrailingComma(true),
	)
	fmt.Println(value)
	// Output: http://(com,example,)/
}

func ExampleString_withParser() {
	profile := canonicalizer.IA(canonicalizer.WithHostMassage(false), canonicalizer.WithQueryFilters(nil))
	value, err := surt.String("http://www.example.org/Path?CFID=1", surt.WithParser(profile))
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: org,example,www)/path?cfid=1
}

func ExampleURL() {
	u, err := canonicalizer.IA().Parse("http://www.example.org/Path?b=2&a=1")
	if err != nil {
		panic(err)
	}
	value, err := surt.URL(u)
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: org,example)/path?a=1&b=2
}

func ExampleUnSURT() {
	value, _ := surt.UnSURT("https://(com,example:80,)/path")
	fmt.Println(value)
	// Output: https://example.com:80/path
}

func ExampleSSURTString() {
	// SSURT puts scheme metadata between the reversed hostname and path.
	value, _ := surt.SSURTString("http://www.example.com/path?query#frag", true)
	fmt.Println(value)
	// Output: com,example,www,//http:/path?query#frag
}
