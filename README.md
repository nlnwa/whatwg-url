# whatwg-url
Whatwg-url is a spec-compliant URL parser written in Go. See [WHATWG](https://url.spec.whatwg.org/#url-parsing) website
for the specification.

## Status
Whatwg-url parser is up to date as of [24 May 2023](https://url.spec.whatwg.org/commit-snapshots/eee49fdf4f99d59f717cbeb0bce29fda930196d4/)
and passes all relevant tests from [web-platform-tests](https://github.com/web-platform-tests/wpt/tree/master/url)

## Usage
API is similar to [Chapter 6 in WHATWG URL Standard](https://url.spec.whatwg.org/#api). See [documentation](https://pkg.go.dev/github.com/nlnwa/whatwg-url) for details.

```go
import "github.com/nlnwa/whatwg-url/url"

url, _ := url.Parse("http://example.com:80/a?b#c")
fmt.Println(url.Scheme())    // http
fmt.Println(url.Host())      // example.com
fmt.Println(url.Port())      // ""
fmt.Println(url.Pathname())  // "/a"
fmt.Println(url.Href(false)) // http://example.com/a?b#c
fmt.Println(url.Href(true))  // http://example.com/a?b
fmt.Println(url.Hash())      // "#c"
fmt.Println(url.Fragment())  // "c"
fmt.Println(url.Search())    // "?b"
fmt.Println(url.Query())     // "b"
fmt.Println(url)             // http://example.com/a?b#c
```

### Options
The default parser instance follows the WHATWG URL Standard. To adapt parsing to other needs, create a new parser
instance and configure it with [options](https://pkg.go.dev/github.com/nlnwa/whatwg-url/url#ParserOption).

example:

```go
p := url.NewParser(url.WithAcceptInvalidCodepoints(), url.WithCollapseConsecutiveSlashes())
```

### Canonicalization
If you want canonicalization beyond what's described in the standard, you can use the 
[Canonicalizer API](https://pkg.go.dev/github.com/nlnwa/whatwg-url/canonicalizer).
You can define your own canonicalization profile:

```go
c := canonicalizer.New(canonicalizer.WithRemoveUserInfo(), canonicalizer.WithRemoveFragment())
url, err := c.Parse("http://user@example.com/a?b#c")
```

Or use one of the predefined profiles:

```go
url, err := canonicalizer.GoogleSafeBrowsing.Parse("http://user@example.com/a?b#c")
```
