package url_test

import (
	"errors"
	"fmt"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func ExampleNewParser() {
	p := url.NewParser(url.WithAcceptInvalidCodepoints(), url.WithCollapseConsecutiveSlashes())
	u, err := p.Parse("http://example*.com/a//d?b#c")
	if err == nil {
		fmt.Println(u)
	}
	// Output: http://example*.com/a/d?b#c
}

func ExampleUrl_Scheme() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Scheme())
	// Output: http
}

func ExampleUrl_Host() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Host())

	// Output: example.com
}

func ExampleUrl_Port() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Port())
	// Output:
}

func ExampleUrl_Pathname() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Pathname())
	// Output: /a
}

func ExampleUrl_Href() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Href(false)) // http://example.com/a?b#c
	fmt.Println(u.Href(true))  // http://example.com/a?b
	// Output: http://example.com/a?b#c
	// http://example.com/a?b
}

func ExampleUrl_Hash() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Hash())
	// Output: #c
}

func ExampleUrl_Fragment() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Fragment())
	// Output: c
}

func ExampleUrl_Search() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Search())
	// Output: ?b
}

func ExampleUrl_Query() {
	u, _ := url.Parse("http://example.com:80/a?b#c")
	fmt.Println(u.Query())
	// Output: b
}

func ExampleParse_errorsIs() {
	_, err := url.Parse("https://example.org:7z")
	fmt.Println(errors.Is(err, url.ErrPortInvalid))
	// Output: true
}

func ExampleUrl_ValidationErrors() {
	u, err := url.Parse(" https://example.org ")
	if err != nil {
		fmt.Println(false)
		return
	}

	hasInvalidURLUnit := false
	for _, verr := range u.ValidationErrors() {
		if errors.Is(verr, url.ErrInvalidURLUnit) {
			hasInvalidURLUnit = true
			break
		}
	}

	fmt.Println(hasInvalidURLUnit)
	// Output: true
}

func ExamplePercentEncodeSet_Set() {
	set := url.QueryPercentEncodeSet.Set('#', '?')
	encoded := url.PercentEncode("a#b?c", set)
	fmt.Println(encoded)
	// Output: a%23b%3Fc
}

func ExamplePercentEncodeSet_Bytes() {
	fmt.Println(url.QueryPercentEncodeSet.Bytes())

	// Output:
	// [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 34 35 60 62]
}

func ExampleWithQueryPercentEncodeSet() {
	set := url.QueryPercentEncodeSet.Set('&').Clear(';')
	p := url.NewParser(url.WithQueryPercentEncodeSet(set))
	u, _ := p.Parse("foo://example.com/?a&b")
	fmt.Println(u.String())
	// Output: foo://example.com/?a%26b
}

// func ExampleNewCanonicalizer() {
// 	c := url.NewCanonicalizer(url.WithRemoveUserInfo(), url.WithRemoveFragment())
// 	u, err := c.Parse("http://user@example.com/a//d?b#c")
// 	if err == nil {
// 		fmt.Println(u)
// 	}
// 	// Output: http://example.com/a//d?b
// }

// func ExampleGoogleSafeBrowsing() {
// 	u, err := url.GoogleSafeBrowsing.Parse("http://user@example.com/a//d?b#c")
// 	if err == nil {
// 		fmt.Println(u)
// 	}
// 	// Output: http://user@example.com/a/d?b
// }

// func ExampleWithRepeatedPercentDecodingEncoding() {
// 	strict := url.NewCanonicalizer(url.WithRepeatedPercentDecoding())
// 	legacy := url.NewCanonicalizer(
// 		url.WithRepeatedPercentDecoding(),
// 		url.WithRepeatedPercentDecodingEncoding(charmap.ISO8859_1),
// 	)

// 	u1, _ := strict.Parse("http://example.com/%E5")
// 	u2, _ := legacy.Parse("http://example.com/%E5")

// 	fmt.Println(u1)
// 	fmt.Println(u2)
// 	// Output: http://example.com/%E5
// 	// http://example.com/%C3%A5
// }
