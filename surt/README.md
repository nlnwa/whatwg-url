# SURT and SSURT

Package `github.com/nationallibraryofnorway/whatwg-url/surt` builds
Internet Archive-style Sort-friendly URI Reordering Transform keys from the
public `url` APIs. The package is experimental. `UnSURT` converts a key back
to a URL string, using `http` when the key omits the scheme. Reversal cannot
recover spelling or components removed by canonicalization. SSURT support
follows the [IIPC format description](https://github.com/iipc/urlcanon/blob/master/ssurt.rst)
and includes the corresponding `UnSSURT` helper. The SURT format is described in the
[Heritrix glossary](http://crawler.archive.org/articles/user_manual/glossary.html#surt).

`surt.String` uses [`canonicalizer.IA()`](../canonicalizer/README.md#ia) with its default settings as its parser.
Select any other `url.Parser`, including a configured profile, with
`surt.WithParser`:

```go
profile := canonicalizer.IA(
    canonicalizer.WithHostMassage(false),
    canonicalizer.WithQueryFilters(nil),
)
key, err := surt.String("http://www.example.org/Path?CFID=1",
    surt.WithParser(profile))
// key: org,example,www)/path?cfid=1
```

`surt.URL` formats an already-parsed URL without canonicalizing, cloning, or
mutating it. For explicit composition, call `profile.Parse(rawURL)` and then
`surt.URL(u)`. Formatting options control scheme inclusion, trailing commas,
and IP reversal. `WithParser` applies only to `String`; nil restores its
default parser. Empty input produces `-`, and passthrough schemes retain their
original input without invoking the parser. Default-scheme fallback belongs
to the selected profile.

SSURT uses a parse-only default, retaining path case, credentials,
query order, and fragments. To select canonicalization explicitly, pass the
result of a profile's `Parse` method to `surt.SSURTURL`.

When formatting a host as a SURT key, the formatter omits credentials and
fragments. SSURT keys include the scheme, port, and credentials only when
`includeScheme` is true. Key serialization omits empty query markers (`?`)
and empty fragment markers (`#`). Reversal therefore does not guarantee an
exact round trip even without canonicalization.

The relevant language-neutral vectors from IIPC urlcanon are vendored under
`testdata/iipc-urlcanon` and run as part of `go test ./...`.

The [Common Crawl corpus test](../testdata/non-utf8/README.md) checks URL-parser
byte preservation across 533,168 fixture rows with `go test -tags=long ./...`.
The SURT suite includes the four non-UTF-8 preservation cases from
[Internet Archive SURT PR 30](https://github.com/internetarchive/surt/pull/30).
Matching an existing index requires the same canonicalization policy used to
create its keys.
