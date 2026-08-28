# whatwg-url

whatwg-url is a spec-compliant URL parser written in Go. See [WHATWG](https://url.spec.whatwg.org/#url-parsing) website for the specification.

## Status

Whatwg-url parser is up to date as of [16 January 2026](https://github.com/web-platform-tests/wpt/tree/b200561bc63dc0951a6df7d11fc333c71cb68e36) and passes all relevant tests from [web-platform-tests](https://github.com/web-platform-tests/wpt/tree/master/url).

The library requires Go 1.27 or later. Go 1.27 provides the Unicode 17.0
tables needed to pass the complete bundled WPT URL and IDNA conformance test
suites.

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

The default `Parse` function and `NewParser` implement WHATWG parsing. Opt-in,
policy-neutral primitives are available for applications that need additional
canonicalization:

```go
u, _ := url.Parse("https://example.org/?q=a+b&flag&space=%20")
u.RewriteQuery(func(_ *url.Url, raw string, present bool) (string, bool) {
    query := url.ParseRawQuery(raw)
    query.SortAbsolute()
    return query.String(), present
})
fmt.Println(u.Query()) // flag&q=a+b&space=%20
```

These APIs rewrite already-serialized components without reparsing or
re-encoding them. They are experimental extensions, not WHATWG behavior.
Callers are responsible for returning valid serialized component values.

Use `SearchParams` when the query represents form data. Use `RawQuery` when its
serialized spelling matters. `SearchParams` uses
`application/x-www-form-urlencoded` semantics: it decodes percent escapes,
turns `+` into a space, discards empty terms, treats `flag` like `flag=`, and
form-serializes after mutation. `RawQuery` preserves those distinctions.

#### Migration from the removed canonicalizer API

- Replace WHATWG canonicalizer profiles with `Parse` or `NewParser`.
- Remove credentials, ports, and fragments with `SetUsername("")`,
  `SetPassword("")`, `SetPort("")`, and `SetHash("")` (or
  `RewriteFragment` when absent versus empty matters).
- Use `SearchParams` for form-query operations.
- Use `ParseRawQuery` inside `RewriteQuery` for lossless query sorting or
  filtering.
- Use `PercentDecodeRepeated` followed by `PercentEncode` for repeated escape
  normalization.
- Implement default-scheme behavior as a caller-owned parse fallback.
- Replace component transformers with `RewriteHostname`, `RewritePathname`,
  `RewriteQuery`, and `RewriteFragment`.

For example, repeated Safe Browsing-style path escape normalization can be
composed without a Safe Browsing package or profile:

```go
escapeSet := url.NewPercentEncodeSet(0x20, ' ', '#', '%')
u.RewritePathname(func(_ *url.Url, path string) string {
    return url.PercentEncode(url.PercentDecodeRepeated(path), escapeSet)
})
```

This composition follows the repeated decode/one-pass escape shape used by
[Google Safe Browsing](https://github.com/google/safebrowsing/blob/master/urls.go),
but it deliberately does not define that product's full legacy parsing policy.

### Experimental SURT support

Package `github.com/nationallibraryofnorway/whatwg-url/surt` builds
Internet Archive-style Sort-friendly URI Reordering Transform keys from the
public `url` APIs. SURT is not a web standard, and this package is experimental.
Its `UnSURT` reverse API is an extension. SSURT support follows the
[IIPC format description](https://github.com/iipc/urlcanon/blob/master/ssurt.rst),
with an experimental Go API and reverse helper. The historical SURT format is
described in the
[Heritrix glossary](http://crawler.archive.org/articles/user_manual/glossary.html#surt).

The relevant language-neutral vectors from IIPC urlcanon are vendored under
`testdata/iipc-urlcanon` and run as part of `go test ./...`.

### Percent-encode set customization

`PercentEncodeSet` can be customized directly with `Set` and `Clear`:

```go
set := url.QueryPercentEncodeSet.Set('#', '&').Clear(';')
p := url.NewParser(url.WithQueryPercentEncodeSet(set))
```
