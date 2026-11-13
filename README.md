# whatwg-url

whatwg-url is a Go implementation of the [WHATWG URL Standard](https://url.spec.whatwg.org/).
It provides URL parsing, relative-reference resolution, component setters,
and form query parameters, with experimental canonicalization and SURT packages.

> **Module path:** `github.com/nationallibraryofnorway/whatwg-url`.
> See [Upgrading from v0.6.2](#upgrading-from-v062) for migration instructions.

## Status

The parser passes the bundled URL parsing, setter, and IDNA tests from
[web-platform-tests](https://github.com/web-platform-tests/wpt/tree/c48d58747e1f211527fb695fd60548a997fae617/url).
Applicable SearchParams cases are covered by [Go adaptations](testdata/wpt-searchparams/README.md).

WPT fixture revision: [19 September 2026](https://github.com/web-platform-tests/wpt/tree/c48d58747e1f211527fb695fd60548a997fae617).

## Requirements

Go 1.27 or later is required. The IDNA and Unicode normalization dependencies
use Unicode 17.0 tables with this toolchain.

## Upgrading from v0.6.2

**This upgrade includes breaking changes to the module path, public APIs, and
validation behavior.**

Starting with `v0.8.0`, the module path is
`github.com/nationallibraryofnorway/whatwg-url`. Consumers of `v0.6.2` must
migrate from `github.com/nlnwa/whatwg-url` in two steps:

1. **Update imports:** Replace the prefix `github.com/nlnwa/whatwg-url` with
   `github.com/nationallibraryofnorway/whatwg-url` in all Go source files,
   including tests. The old `errors` package has been removed; migrate those
   usages to `url`.
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
and the [canonicalizer migration guide](canonicalizer/README.md#migrating-from-v062),
and adapt your code before running the tests.

Existing projects pinned to `v0.6.2` continue to work without changes. Releases
from `v0.8.0` use the new module path. This guide compares `v0.6.2` with `v0.8.0`.

### Parser options and helper API migration

The following APIs also change when upgrading from `v0.6.2`:

| Previous API | Migration |
| --- | --- |
| `WithReportValidationErrors()` | Use `WithReportValidationErrors(true)`, or omit it because reporting is enabled by default. |
| `WithSpecialSchemes(map[string]string{...})` | Use `map[string]uint16` with numeric ports, for example `map[string]uint16{"http": 80}`. |
| `WithPreParseHostFunc` / `WithPostParseHostFunc` | Transform the input before parsing, or use `RewriteHostname` for transformations of a valid serialized hostname after parsing. |
| `WithEncodingOverride` | The option has been removed without a transcoding replacement. Already percent-encoded path/query bytes need no encoding hint and are preserved by the parser. |
| `Parser.BasicParser(...)` | Use `url.BasicParsing` and its `BasicParse(...)` method, for example `p.(url.BasicParsing).BasicParse(...)` for a parser from `NewParser`. |
| `Parser.PercentEncodeString(...)` | Use the package-level `url.PercentEncode(...)`. |
| `Parser.NewUrl()` | Parse an initial URL and modify it through the URL API. `url.Parser` now exposes only `Parse` and `ParseRef`. |
| `Url.DecodedPort() int` | The result is now `uint16`; convert explicitly if your application needs `int`. |
| `NewPercentEncodeSet`, `PercentEncodeSet.Set` / `Clear` with `uint` arguments | Pass `rune` values or rune literals. Convert slices to `[]rune` before expanding them. |
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

### SearchParams migration

`SearchParams` now consistently uses WHATWG `application/x-www-form-urlencoded`
semantics. In `v0.6.2`, it combined form behavior, such as converting spaces to
`+`, with the URL parser's configurable escaping and decoding. Form processing
now uses UTF-8 and form escaping independently of those parser settings.

These changes can affect both returned values and serialized URLs:

| Operation | v0.6.2 behavior | v0.8.0 behavior |
| --- | --- | --- |
| Decode `q=a+b%2Bc` | `Get("q")` returned `"a b c"`. | Returns `"a b+c"`: literal `+` becomes a space before percent-decoding, so `%2B` remains a plus. |
| Decode malformed UTF-8, such as `q=%FC` | Without an encoding override, returned the raw byte `0xFC`. | Returns U+FFFD (`�`). Mutating the parameters serializes it as `%EF%BF%BD`. Different invalid byte sequences can produce the same decoded name or value. |
| Serialize names and values | Used the parser's query encode set and optional encoding override. | Uses UTF-8 form encoding: spaces become `+`; literal `+`, `&`, `=`, `%`, and `~` become `%2B`, `%26`, `%3D`, `%25`, and `%7E`. Parser encode-set options do not change form serialization. |
| `Sort()` | Compared decoded names as Go strings. | Stably compares UTF-16 code units. For example, U+10000 sorts before U+E000. |
| `SortAbsolute()` | Compared concatenated names and values. | Compares names first, then values, using UTF-16 order for both. For example, `ab=a&a=z` becomes `a=z&ab=a`. |
| Mutate a query to an empty parameter list | Could leave a trailing `?`. | Removes the query, including `?`. If the URL has an opaque path and no fragment, trailing path spaces are also stripped. |

Reading parameters leaves the URL's serialized query unchanged. Mutations,
including `Sort()` and `Iterate()`, serialize the whole parameter list, so the
escaping and decoding changes above can affect parameters you did not edit.
Use `RewriteQuery` for edits that must preserve serialized terms, literal `+`,
empty terms, or the distinction between `flag` and `flag=`.

The API changes are:

| API | Migration |
| --- | --- |
| `Url.SetSearchParams` | Mutate `u.SearchParams()` directly. To copy form data from another URL, use `u.SetSearch(source.SearchParams().String())`. |
| `SearchParams.Clone` | Use `u.Clone().SearchParams()` for an independent form view. Mutating a cloned URL's parameters now updates the clone rather than the original URL. |
| `SearchParams.QueryEscape` | Pass decoded names and values to `Append` or `Set`; use `String()` to serialize the form data. |
| `WithSkipEqualsForEmptySearchParamsValue` | Removed. Form serialization writes `flag=` for an empty value. Use `RewriteQuery` to preserve bare terms. |
| `SearchParams.Filter` | New method: `Filter(func(name, value string) bool)` keeps pairs for which the predicate returns true and synchronizes the URL. |

`Iterate` still receives `*NameValuePair`, but pairs are now stored as values
in a slice. Modify them within the callback; use the parameter methods for
later edits instead of retaining pair pointers across mutations.

### Error handling migration

The old `github.com/nlnwa/whatwg-url/errors` package has been removed. Replace
its `Type`, `Description`, `Failure`, and `Url` helpers with the types and
sentinels described in
[Validation errors and parse failures](#validation-errors-and-parse-failures).

`ValidationError.Code()` replaces `Type`, and `ValidationError.Detail()`
replaces `Description`. Use `errors.Is(err, url.ErrParseFailure)` in place of
`Failure`. Errors no longer store the input URL; retain it at the call site
if you previously used the `Url` helper for logging.

## Usage

The API follows the [WHATWG URL API](https://url.spec.whatwg.org/#api).
See the [package reference](https://pkg.go.dev/github.com/nationallibraryofnorway/whatwg-url/url) for details.

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

`WithLaxHostParsing()` recovers domain, opaque-host, IPv4, and IPv6 parsing failures.
IPv4 recovery retains the normalized domain as a hostname, without marking it
as an IPv4 address. IPv6 recovery retains the consumed host spelling, including
unclosed brackets, without marking it as IPv6. Valid IP addresses still normalize.

`WithSkipDotSegmentNormalization()` preserves literal and percent-encoded dot
segments during hierarchical path parsing. The `canonicalizer.Semantic()` and
`canonicalizer.GoogleSafeBrowsing()` profiles use this option to resolve dots
after repeated decoding; ordinary parsing uses standard dot normalization.

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
    fmt.Println(u == nil)                           // true
    fmt.Println(errors.Is(err, url.ErrParseFailure)) // true
    fmt.Println(errors.Is(err, url.ErrPortInvalid))  // true
    for _, validationErr := range parseErr.ValidationErrors() {
        fmt.Println(validationErr.Code()) // port-invalid
    }
}
```

### Character encoding

The URL parser preserves percent-encoded bytes in paths and queries without
requiring an encoding hint. For example, `https://example.org/%FC?q=%FC`
serializes unchanged. Percent escapes do not identify a character encoding;
invalid UTF-8 sequences can remain in serialized path and query components.

`WithAcceptInvalidCodepoints()` also preserves literal invalid UTF-8 input
bytes in paths, queries, and fragments by percent-encoding each byte. It does
not transcode those bytes or change valid UTF-8 text. Validation is recorded
before recovery, and `WithFailOnValidationError()` takes precedence.

Use `RewriteQuery` when edits must preserve serialized query
terms, including literal `+`, empty terms, and the distinction between `flag`
and `flag=`. The byte-oriented percent helpers operate on bytes without
character-set decoding. For archival indexing, select operations consistent
with the index's canonicalization policy.

See the [fixture documentation](testdata/non-utf8/README.md) and the
[executable examples](url/example_test.go) for details.

### SearchParams

`SearchParams` implements the WHATWG URL Standard's
`application/x-www-form-urlencoded` parsing and serialization rules. Parsing
splits the query into name/value pairs, converts literal `+` to space,
percent-decodes names and values, and decodes the resulting bytes as UTF-8.
Invalid UTF-8 sequences are replaced with U+FFFD. Empty terms are discarded,
and a bare term such as `flag` is treated as an empty value, equivalent to
`flag=`. Duplicate pairs retain their order.

Reading `SearchParams` does not modify the serialized URL. Mutating it
serializes all pairs back into the query using UTF-8 form encoding: spaces
become `+`, and characters outside the form encoding's allowed set are
percent-encoded. For example, `%FC` represents `ü` in ISO-8859-1, but the byte
`0xFC` is invalid in UTF-8. SearchParams therefore decodes it as the replacement
character `�` (U+FFFD), which is written as `%EF%BF%BD` when a mutation serializes
the query. This can change bytes and merge distinctions between originally
different byte sequences.

### Percent-encode set customization

Use `Set` and `Clear` to derive a new `url.PercentEncodeSet` without changing
the original:

```go
set := url.QueryPercentEncodeSet.Set('#', '&').Clear(';')
p := url.NewParser(url.WithQueryPercentEncodeSet(set))
```

`canonicalizer` provides `LaxPathPercentEncodeSet`, `LaxQueryPercentEncodeSet`,
and `RepeatedQueryPercentEncodeSet`. Pass a preset to a parser option:

```go
p := url.NewParser(
    url.WithPathPercentEncodeSet(canonicalizer.LaxPathPercentEncodeSet),
)
```

See the [preset documentation](canonicalizer/README.md#percent-encode-presets)
for their escaping policies.

### Canonicalization

The experimental `canonicalizer` package applies ordered rules to parsed URLs.
Use a built-in profile or compose rules for your canonicalization policy.
See the [canonicalizer README](canonicalizer/README.md) for profiles,
configuration, and examples.

### Experimental SURT support

The experimental `surt` package generates SURT and SSURT keys for URL indexing
and provides helpers to convert keys back to URL strings. See the
[SURT README](surt/README.md) for usage, canonicalization options, and format
differences.

## Testing

Run from the repository root:

```sh
go test ./...
go test -race ./...
go vet ./...
```

Use `go test -tags=long ./...` to include the full Common Crawl non-UTF-8 URL
corpus. See [test fixtures](testdata/README.md) for sources and refresh commands.
