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

func ExampleSearchParams_lossyFormSemantics() {
	u, _ := url.Parse("https://example.org/?q=a+b&flag&space=%20")
	u.SearchParams().Sort()
	fmt.Println(u.Query())
	// Output: flag=&q=a+b&space=+
}

func ExampleRawQuery_losslessCanonicalization() {
	u, _ := url.Parse("https://example.org/?q=a+b&flag&space=%20")
	u.RewriteQuery(func(_ *url.Url, raw string, present bool) (string, bool) {
		query := url.ParseRawQuery(raw)
		query.SortAbsolute()
		return query.String(), present
	})
	fmt.Println(u.Query())
	// Output: flag&q=a+b&space=%20
}

// This example mirrors the composition used by Safe Browsing-style URL
// canonicalizers without defining a Safe Browsing profile or lookup API.
func ExampleUrl_safeBrowsingStyleCanonicalization() {
	u, _ := url.Parse("http://user:pass@host:8080/%25%32%35#fragment")

	// Safe Browsing's escape policy encodes controls, non-ASCII bytes, spaces,
	// '#', and '%' after recursively decoding percent escapes.
	safeBrowsingEscapes := url.NewPercentEncodeSet(0x20, ' ', '#', '%')
	u.RewritePathname(func(_ *url.Url, path string) string {
		return url.PercentEncode(url.PercentDecodeRepeated(path), safeBrowsingEscapes)
	})
	u.SetUsername("")
	u.SetPassword("")
	u.SetPort("")
	u.RewriteFragment(func(_ *url.Url, _ string, _ bool) (string, bool) {
		return "", false
	})

	fmt.Println(u)
	// Output: http://host/%25
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
