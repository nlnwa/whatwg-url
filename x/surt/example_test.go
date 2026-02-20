package surt_test

import (
	"fmt"

	"github.com/nationallibraryofnorway/whatwg-url/x/surt"
)

func ExampleString() {
	v, _ := surt.String("http://archive.org/goo/?a=2&b&a=1")
	fmt.Println(v)
	// Output: org,archive)/goo?a=1&a=2&b
}

func ExampleString_withOptions() {
	v, _ := surt.String(
		"http://www.example.com/",
		surt.WithIncludeScheme(true),
		surt.WithTrailingComma(true),
	)
	fmt.Println(v)
	// Output: http://com,example,)/
}

func ExampleSSURTString() {
	v, _ := surt.SSURTString("http://www.example.com/foo/bar?query#frag", true)
	fmt.Println(v)
	// Output: com,example,www,//:http:/foo/bar?query#frag
}

func ExampleUnSSURT() {
	v, _ := surt.UnSSURT("org,example,foo,//81:http@user:pass:/path?query#frag")
	fmt.Println(v)
	// Output: http://user:pass@foo.example.org:81/path?query#frag
}

func ExampleUnSURT_normalizationLimitation() {
	original := "http://www.Example.com/path/?b=2&a=1#Frag"
	s, _ := surt.String(original)
	u, _ := surt.UnSURT(s)
	fmt.Println(original)
	fmt.Println(s)
	fmt.Println(u)
	// Output: http://www.Example.com/path/?b=2&a=1#Frag
	// com,example)/path?a=1&b=2
	// http://example.com/path?a=1&b=2
}
