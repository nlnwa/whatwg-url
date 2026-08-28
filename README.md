# whatwg-url

whatwg-url is a spec-compliant URL parser written in Go. See [WHATWG](https://url.spec.whatwg.org/#url-parsing) website for the specification.

## Status

Whatwg-url parser is up to date as of [16 January 2026](https://github.com/web-platform-tests/wpt/tree/b200561bc63dc0951a6df7d11fc333c71cb68e36) and passes all relevant tests from [web-platform-tests](https://github.com/web-platform-tests/wpt/tree/master/url).

## Usage

API is similar to [Chapter 6 in WHATWG URL Standard](https://url.spec.whatwg.org/#api). See [documentation](https://pkg.go.dev/github.com/nlnwa/whatwg-url) for details.

```go
import "github.com/nationallibraryofnorway/whatwg-url/url"

u, _ := url.Parse("http://example.com:80/a?b#c")
fmt.Println(u.Scheme())    // http
fmt.Println(u.Host())      // example.com
fmt.Println(u.Port())      // ""
fmt.Println(u.Pathname())  // "/a"
fmt.Println(u.Href(false)) // http://example.com/a?b#c
fmt.Println(u.Href(true))  // http://example.com/a?b
fmt.Println(u.Hash())      // "#c"
fmt.Println(u.Fragment())  // "c"
fmt.Println(u.Search())    // "?b"
fmt.Println(u.Query())     // "b"
fmt.Println(u)             // http://example.com/a?b#c
```

### Options

The default parser instance follows the WHATWG URL Standard. To adapt parsing to other needs, create a new parser instance and configure it with [options](https://pkg.go.dev/github.com/nationallibraryofnorway/whatwg-url/url#ParserOption).

Validation errors do not fail parsing unless the standard says to return
failure. Successful URLs retain their validation history by default;
`WithReportValidationErrors(false)` disables only that successful-URL history,
not failure diagnostics. `WithLaxHostParsing`, `WithAcceptInvalidCodepoints`,
and `WithPercentEncodeSinglePercentSign` are deliberate extensions: standard
validation is detected before they recover. `WithFailOnValidationError` stops
at the first validation occurrence and takes precedence over every recovery
option, regardless of option order.

Example:

```go
p := url.NewParser(
    url.WithAcceptInvalidCodepoints(),
    url.WithCollapseConsecutiveSlashes(),
)
```

### Validation errors and parse failures

A `ValidationError` represents one WHATWG validation-error occurrence. Its
`Code()` is a stable `ValidationErrorCode`, while `Detail()` is intended for
people and can include more context. A validation error is not necessarily
fatal: the default parser continues whenever the standard does.

When the standard says to return failure, `Parse`, `ParseRef`, and `BasicParse`
return a nil URL and a `*ParseError`. Every `ParseError` matches
`ErrParseFailure` and exposes all validation occurrences detected before the
failure through both `ValidationErrors()` and Go multi-error unwrapping. Some
standard failure steps have no validation-error category; those return a
`ParseError` with an empty validation list.

```go
import (
    "errors"
    "fmt"

    "github.com/nationallibraryofnorway/whatwg-url/url"
)

u, err := url.Parse(" https://example.org ")
if err == nil {
    for _, validationErr := range u.ValidationErrors() {
        fmt.Println(validationErr.Code()) // invalid-URL-unit
    }
}

u, err = url.Parse("https://example.org:7z")
if parseErr, ok := errors.AsType[*url.ParseError](err); ok {
    fmt.Println(u == nil)                             // true
    fmt.Println(errors.Is(err, url.ErrParseFailure)) // true
    fmt.Println(errors.Is(err, url.ErrPortInvalid))  // true
    for _, validationErr := range parseErr.ValidationErrors() {
        fmt.Println(validationErr.Code()) // port-invalid
    }
}
```

### Canonicalization

If you want canonicalization beyond what's described in the standard, you can use the
[Canonicalizer API](https://pkg.go.dev/github.com/nationallibraryofnorway/whatwg-url/canonicalizer). You can define your own canonicalization profile:

```go
import "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"

c := canonicalizer.New(
    canonicalizer.WithRemoveUserInfo(),
    canonicalizer.WithRemoveFragment(),
)
u, err := c.Parse("http://user@example.com/a?b#c")
```

Or use one of the predefined profiles:

```go
u, err := canonicalizer.GoogleSafeBrowsing.Parse("http://user@example.com/a?b#c")
```

Parser behavior remains UTF-8 based and does not guess alternate encodings.
If you need compatibility handling for legacy percent-encoded Latin-1 octets during canonicalization,
use canonicalizer options:

```go
c := canonicalizer.New(
    canonicalizer.WithRepeatedPercentDecoding(),
    canonicalizer.WithRepeatedPercentDecodingEncoding(charmap.ISO8859_1),
)
```

For advanced WHATWG state-machine usage (`BasicParser` and parser states), use
`github.com/nationallibraryofnorway/whatwg-url/x/whatwg`.

### Percent-encode set customization

`PercentEncodeSet` can be customized directly with `Set` and `Clear`:

```go
set := url.QueryPercentEncodeSet.Set('#', '&').Clear(';')
p := url.NewParser(url.WithQueryPercentEncodeSet(set))
```

## Web Platform Tests Compliance

Go 1.27 is required because the WPT and IDNA conformance data depend on its
Unicode 17.0 tables. The included WHATWG URL and IDNA tests run normally as
part of `go test ./...`; no tests are skipped and no extra test flags are
needed.
