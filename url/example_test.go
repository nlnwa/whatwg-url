package url_test

import (
	"errors"
	"fmt"

	urlerrors "github.com/nationallibraryofnorway/whatwg-url/errors"
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
	fmt.Println(errors.Is(err, urlerrors.ErrPortInvalid))
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
		if errors.Is(verr, urlerrors.ErrInvalidURLUnit) {
			hasInvalidURLUnit = true
			break
		}
	}

	fmt.Println(hasInvalidURLUnit)
	// Output: true
}
