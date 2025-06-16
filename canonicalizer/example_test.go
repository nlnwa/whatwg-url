package canonicalizer_test

import (
	"fmt"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
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
