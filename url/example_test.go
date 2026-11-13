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

func ExampleQuery_losslessCanonicalization() {
	u, _ := url.Parse("https://example.org/?q=a+b&flag&space=%20")
	u.RewriteQuery(func(_ *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
		query.SortAbsolute()
		return query, hasQuery
	})
	fmt.Println(u.Query())
	// Output: flag&q=a+b&space=%20
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
	fmt.Printf("%v\n", url.QueryPercentEncodeSet.Bytes())

	// Output:
	// [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 34 35 60 62]
}

func ExampleWithQueryPercentEncodeSet() {
	set := url.QueryPercentEncodeSet.Set('&').Clear(';')
	p := url.NewParser(url.WithQueryPercentEncodeSet(set))
	u, _ := p.Parse("foo://example.com/?a&b")
	fmt.Println(u.String())
	// Output: foo://example.com/?a%26b
}

func ExampleParse_preserveNonUTF8Bytes() {
	u, err := url.Parse("https://example.org/%FC?q=%FC")
	if err != nil {
		panic(err)
	}
	// Already percent-encoded path/query bytes need no encoding hint.
	fmt.Println(u)
	reparsed, err := url.Parse(u.String())
	if err != nil {
		panic(err)
	}
	fmt.Println("reparse unchanged:", reparsed.String() == u.String())
	// Output:
	// https://example.org/%FC?q=%FC
	// reparse unchanged: true
}

func ExampleSearchParams_nonUTF8FormSemantics() {
	u, err := url.Parse("https://example.org/?q=%FC")
	if err != nil {
		panic(err)
	}
	params := u.SearchParams()
	fmt.Printf("%U\n", []rune(params.Get("q"))[0])
	fmt.Println(u) // Reading the form view does not rewrite the URL.
	params.Append("probe", "x")
	fmt.Println(u) // Mutation serializes all form tuples, including replacement.
	// Output:
	// U+FFFD
	// https://example.org/?q=%FC
	// https://example.org/?q=%EF%BF%BD&probe=x
}
