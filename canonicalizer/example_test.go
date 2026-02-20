package canonicalizer_test

import (
	"fmt"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"golang.org/x/text/encoding/charmap"
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

func ExampleWithRepeatedPercentDecodingEncoding() {
	strict := canonicalizer.New(canonicalizer.WithRepeatedPercentDecoding())
	legacy := canonicalizer.New(
		canonicalizer.WithRepeatedPercentDecoding(),
		canonicalizer.WithRepeatedPercentDecodingEncoding(charmap.ISO8859_1),
	)

	u1, _ := strict.Parse("http://example.com/%E5")
	u2, _ := legacy.Parse("http://example.com/%E5")

	fmt.Println(u1)
	fmt.Println(u2)
	// Output: http://example.com/%E5
	// http://example.com/%C3%A5
}
