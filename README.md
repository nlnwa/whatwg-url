# whatwg-url

whatwg-url is a spec-compliant URL parser written in Go. See [WHATWG](https://url.spec.whatwg.org/#url-parsing) website for the specification.

> **Module path:** Starting with `v0.8.0`, use
> `github.com/nationallibraryofnorway/whatwg-url` instead of
> `github.com/nlnwa/whatwg-url`. See [Upgrading from v0.6.2](#upgrading-from-v062)
> for migration instructions..

## Status

Whatwg-url parser is up to date as of [16 January 2026](https://github.com/web-platform-tests/wpt/tree/b200561bc63dc0951a6df7d11fc333c71cb68e36) and passes all relevant tests from [web-platform-tests](https://github.com/web-platform-tests/wpt/tree/master/url).

## Upgrading from v0.6.2

Starting with `v0.8.0`, the module path is
`github.com/nationallibraryofnorway/whatwg-url`. Consumers of `v0.6.2` must
migrate from `github.com/nlnwa/whatwg-url` in two steps:

1. **Update imports:** Replace the prefix `github.com/nlnwa/whatwg-url` with
   `github.com/nationallibraryofnorway/whatwg-url` in all Go source files,
   including tests. The old `canonicalizer` and `errors` packages have been
   removed; migrate those usages to the APIs in `url` as described below.
2. **Update the dependency:** Using Go 1.27 or later, run the following commands
   from your project's module root:

   ```bash
   go get github.com/nationallibraryofnorway/whatwg-url@v0.8.0
   go mod tidy
   go test ./...
   ```

`go get` updates the dependency version but does not rewrite Go import
statements. Once all imports are migrated, `go mod tidy` removes the unused old
module requirement from `go.mod`, provided no other dependencies still require
it.

This release also changes public APIs and validation behavior. Review
[validation errors and parse failures](#validation-errors-and-parse-failures)
and the [canonicalizer migration guide](#migration-from-the-removed-canonicalizer-api),
and adapt your code before running the tests.

Existing projects pinned to `v0.6.2` continue to work without changes. Future
releases will only be published under the new module path. Version `v0.7.0`
under the new path is retracted because it was published from a temporary
fork before the repository migration. `v0.8.0` is the first supported release
under the new path.

### Parser options and helper API migration

The following APIs also change when upgrading from `v0.6.2`:

| Previous API | Migration |
| --- | --- |
| `WithReportValidationErrors()` | Use `WithReportValidationErrors(true)`, or omit it because reporting is enabled by default. |
| `WithSpecialSchemes(map[string]string{...})` | Use `map[string]uint16` with numeric ports, for example `map[string]uint16{"http": 80}`. |
| `WithPreParseHostFunc` / `WithPostParseHostFunc` | Transform the input before parsing, or use `RewriteHostname` for transformations of a valid serialized hostname after parsing. |
| `WithEncodingOverride` | Handle any legacy query encoding in the caller. The parser no longer accepts an encoding override. |
| `WithSkipEqualsForEmptySearchParamsValue` | Use `RawQuery` with `RewriteQuery` when bare query terms must be preserved. |
| `Parser.BasicParser(...)` | Use `url.BasicParsing` and its `BasicParse(...)` method, for example `p.(url.BasicParsing).BasicParse(...)` for a parser from `NewParser`. |
| `Parser.PercentEncodeString(...)` | Use the package-level `url.PercentEncode(...)`. |
| `Parser.NewUrl()` | Parse an initial URL and modify it through the URL API. `url.Parser` now exposes only `Parse` and `ParseRef`. |
| `Url.DecodedPort() int` | The result is now `uint16`; convert explicitly if your application needs `int`. |
| `NewPercentEncodeSet`, `PercentEncodeSet.Set` / `Clear` with `uint` arguments | Pass `rune` values or rune literals. Convert slices to `[]rune` before expanding them. |
| `Url.SetSearchParams`, `SearchParams.Clone` | Mutate the URL's `SearchParams()` directly, copy form data with `SetSearch(source.SearchParams().String())`, or clone the whole URL with `Url.Clone()`. |
| `SearchParams.QueryEscape` | Use `SearchParams` for form query serialization. |
| `EmptyParserOption`, exported `IPv6Addr` | These types have been removed. Omit placeholder options and use `Hostname()` / `IsIPv6()` to inspect parsed IPv6 hosts. |

`NewParser` returns a `url.Parser`. To access the low-level `BasicParse` method,
use the `url.BasicParsing` interface. This example resolves a relative URL
against a base URL:

```go
package main

import (
    "fmt"
    "log"

    "github.com/nationallibraryofnorway/whatwg-url/url"
)

func main() {
    p := url.NewParser()
    basic := p.(url.BasicParsing)

    base, err := p.Parse("https://example.org/docs/guide/")
    if err != nil {
        log.Fatal(err)
    }

    u, err := basic.BasicParse("../api?q=go", base, nil, url.NoState)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(u.Href(false)) // https://example.org/docs/api?q=go
}
```

The third argument is `nil` to create a new URL, and `url.NoState` starts
normal parsing without a state override. For everyday parsing and relative URL
resolution, use `Parse` or `ParseRef`.

`ParseRef("", ref)` now parses the base URL and fails when it is empty. If you
previously used an empty base to parse an absolute URL, call `Parse(ref)`.

## Usage

API is similar to [Chapter 6 in WHATWG URL Standard](https://url.spec.whatwg.org/#api). See [documentation](https://pkg.go.dev/github.com/nationallibraryofnorway/whatwg-url) for details.

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

The old `github.com/nlnwa/whatwg-url/errors` package has been removed. Replace
its `Type`, `Description`, `Failure`, and `Url` helpers with the types and
sentinels in `url` described here. `Url.ValidationErrors()` now returns
`[]*ValidationError` rather than `[]error`. Retain the input URL in your
application if you need it in diagnostics.

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

Use `Parse` or `NewParser` for WHATWG parsing. To apply additional
canonicalization rules, you can rewrite individual URL components. For example,
to sort query parameters:

```go
u, _ := url.Parse("https://example.org/?q=a+b&flag&space=%20")
u.RewriteQuery(func(_ *url.Url, raw string, present bool) (string, bool) {
    query := url.ParseRawQuery(raw)
    query.SortAbsolute()
    return query.String(), present
})
fmt.Println(u.Query()) // flag&q=a+b&space=%20
```

These methods use the strings returned by your callbacks as-is, without parsing
or percent-encoding them again. Each callback must return a valid value for that
URL component. These methods are experimental extensions to the WHATWG URL API.

Use `SearchParams` for form data and `RawQuery` to preserve the query's original
text. `SearchParams` decodes percent escapes, turns `+` into a space, discards
empty terms, and treats `flag` like `flag=`. Changes are written back using
`application/x-www-form-urlencoded` encoding. `RawQuery` preserves those
distinctions.

#### Migration from the removed canonicalizer API

The `canonicalizer` package and its `WhatWg`, `WhatWgSortQuery`,
`GoogleSafeBrowsing`, and `Semantic` profiles have been removed. Use the
following `url` methods to replace the features you need:

- Replace WHATWG canonicalizer profiles with `Parse` or `NewParser`.
- Remove credentials, ports, and fragments with `SetUsername("")`,
  `SetPassword("")`, `SetPort("")`, and `SetHash("")` (or
  `RewriteFragment` when absent versus empty matters).
- Use `SearchParams` for form-query operations.
- Use `ParseRawQuery` inside `RewriteQuery` for lossless query sorting or
  filtering.
- Use `PercentDecodeRepeated` followed by `PercentEncode` for repeated escape
  normalization.
- Handle URLs without a scheme in your application before parsing.
- Replace component transformers with `RewriteHostname`, `RewritePathname`,
  `RewriteQuery`, and `RewriteFragment`.

For example, you can repeatedly decode percent escapes in a path, then encode
it again:

```go
escapeSet := url.NewPercentEncodeSet(0x20, ' ', '#', '%')
u.RewritePathname(func(_ *url.Url, path string) string {
    return url.PercentEncode(url.PercentDecodeRepeated(path), escapeSet)
})
```

This shows the decode-and-encode step used by
[Google Safe Browsing](https://github.com/google/safebrowsing/blob/master/urls.go).
It does not implement the complete Safe Browsing canonicalization rules.

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
