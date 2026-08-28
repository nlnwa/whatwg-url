package surt_test

import (
	"fmt"

	"github.com/nationallibraryofnorway/whatwg-url/surt"
)

func ExampleString() {
	// SURT is an Internet Archive convention, not a web standard. The surt
	// package is experimental.
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

func ExampleUnSURT() {
	value, _ := surt.UnSURT("https://(com,example:80,)/path")
	fmt.Println(value)
	// Output: https://example.com:80/path
}

func ExampleSSURTString() {
	// SSURT is an experimental, non-IA extension.
	value, _ := surt.SSURTString("http://www.example.com/path?query#frag", true)
	fmt.Println(value)
	// Output: com,example,www,//http:/path?query#frag
}
