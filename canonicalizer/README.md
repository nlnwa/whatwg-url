# Canonicalizer

Package `canonicalizer` combines URL parsing with an ordered pipeline of rules.
Profiles implement `url.Parser`, so they can be supplied to APIs that accept a
parser, including `surt.WithParser`.

```go
profile := canonicalizer.New(
    canonicalizer.RemoveUserInfo,
    canonicalizer.RemoveFragment,
    canonicalizer.SortQuery,
).WithParser(url.NewParser(url.WithLaxHostParsing())).WithDefaultScheme("https")

u, err := profile.Parse("user@example.org/path?b=2&a=1#fragment")
// u.String(): https://example.org/path?a=1&b=2
```

`Parse` parses and then runs the rules. `ParseRef` resolves the reference against
the unmodified base before canonicalizing the result. Default-scheme fallback
retries only missing-scheme errors: it trims surrounding whitespace and prepends
`scheme:` to network-path references or `scheme://` to other schemeless inputs.
`Parser()` exposes the underlying parser without rules or scheme fallback.

`Canonicalize(u)` mutates an existing URL **in place**. Clone it first when the
original must be retained. Rules run once, in declaration order; nil rules and
nil URLs are ignored. Parser settings apply during parsing, not when
`Canonicalize` is called on an existing URL. Configuration methods return
independently configured profiles without changing their receiver.

## Migrating from v0.6.2

This guide covers configuration and normalization changes from `v0.6.2` to
`v0.8.0`.

Change imports from `github.com/nlnwa/whatwg-url/canonicalizer` to
`github.com/nationallibraryofnorway/whatwg-url/canonicalizer`. See the root
[upgrade guide](../README.md#upgrading-from-v062) for the dependency update,
Go version requirement, and changes to URL parsing and error handling.

### API changes

The four named profiles in `v0.6.2` were shared parser variables. They are now
factories returning independently configurable `Profile` values:

```go
// v0.6.2
u, err := canonicalizer.Semantic.Parse(raw)

// v0.8.0
u, err := canonicalizer.Semantic().Parse(raw)
```

Custom profiles use ordered rules, with parser settings configured separately:

```go
// v0.6.2
profile := canonicalizer.New(
    url.WithLaxHostParsing(),
    canonicalizer.WithRemoveUserInfo(),
    canonicalizer.WithRemoveFragment(),
    canonicalizer.WithSortQuery(canonicalizer.SortKeys),
    canonicalizer.WithDefaultScheme("https"),
)

// v0.8.0
profile := canonicalizer.New(
    canonicalizer.RemoveUserInfo,
    canonicalizer.RemoveFragment,
    canonicalizer.SortQuery,
).WithParser(url.NewParser(url.WithLaxHostParsing())).WithDefaultScheme("https")
```

| v0.6.2 API | v0.8.0 replacement |
| --- | --- |
| `WhatWg`, `WhatWgSortQuery`, `GoogleSafeBrowsing`, `Semantic` | Call the corresponding factory, for example `Semantic()`. |
| `New(...url.ParserOption) url.Parser` | `New(...Rule) Profile`. A `Profile` still implements `url.Parser`. |
| `WithRemoveUserInfo()`, `WithRemovePort()`, `WithRemoveFragment()` | Pass `RemoveUserInfo`, `RemovePort`, or `RemoveFragment` directly to `New` or `Then`. |
| `WithSortQuery(SortKeys)` / `WithSortQuery(SortParameter)` | Pass `SortQuery` / `SortQueryAbsolute`; see the query behavior changes below. Omit a sorting rule for `NoSort`. |
| `WithRepeatedPercentDecoding()` | Pass `RepeatedPercentDecoding()`. |
| Parser options mixed into `New(...)` | Use `.WithParser(url.NewParser(...))`. |
| `WithDefaultScheme("http")` | Use `.WithDefaultScheme("http")` on the profile. |
| `canonicalizer.LaxPathPercentEncodeSet`, `canonicalizer.LaxQueryPercentEncodeSet` | The lax path preset also permits `^` in v0.8.0. |
| `canonicalizer.RepeatedQueryPercentDecodeSet` | Rename to `canonicalizer.RepeatedQueryPercentEncodeSet`. This set controls encoding after decoding; no compatibility alias is provided. |

`Then`, `WithParser`, and `WithDefaultScheme` return new profiles; retain their
return values. `Canonicalize(u)` applies rules to an existing URL in place and
returns no value. Clone the URL first when its original value must be retained.
For custom component transformations, use the new `RewriteHostname`,
`RewritePathname`, `RewriteQueryString`, `RewriteQuery`, and `RewriteFragment`
rule adapters described [below](#rules-and-query-semantics).

### Behavior changes

The following changes can affect canonical strings:

- **Rule order is explicit.** In `v0.6.2`, options enabled steps in a fixed
  order: repeated decoding, port removal, credential removal, fragment removal,
  then query sorting. Rules now run in the order supplied to `New` and `Then`.
- **Query rules preserve serialized terms.** `v0.6.2` used form decoding and
  serialization through `SearchParams`. The new rules retain literal `+`,
  empty terms, escape spelling when sorting, and `flag` versus `flag=`.
  `SortQuery` compares raw names stably; `SortQueryAbsolute` compares complete
  serialized terms. `Semantic` now uses the latter: `?a=z&a=a` becomes
  `?a=a&a=z`. Use a custom rule calling `u.SearchParams().Sort()` when form
  semantics are wanted; this uses the current WHATWG form implementation.
- **Escape normalization preserves bytes.** The ISO-8859-1 encoding override
  used by `Semantic` in `v0.6.2` is gone. Literal invalid UTF-8 bytes accepted by
  its parser are percent-encoded without transcoding, and existing escaped
  bytes retain their byte values. There is no implicit character-set conversion.
- **Semantic normalization follows decoding.** Dot resolution and
  special-scheme slash collapsing happen after repeated decoding, so encoded
  path structure is handled in the intended order. Newly exposed special hosts
  receive domain and IPv4 normalization. Lax parsing retains malformed IP hosts
  without marking them as IP addresses, with validation history preserved.
- **Google Safe Browsing uses its matching representation.** Its operation
  order and minimal byte escaping now follow Google's published examples.
  Decoded query separators and path delimiters can remain literal; use component
  accessors for lookup expressions instead of reparsing the result as a
  navigation URL. See [GoogleSafeBrowsing](#googlesafebrowsing).
- **IA is a new profile.** `IA()` provides the Internet Archive-style policy
  used by the new SURT package, including path lowercasing, host massage, and
  session filtering.

For archive indexes or crawl deduplication, compare representative URLs from
your application before changing versions. Canonical keys can differ; an
existing index may require an explicit migration or version-specific lookup.

### Upstream origins and compatibility

The profile policies have distinct sources: WHATWG URL parsing for `WhatWg`,
Google's published matching algorithm for `GoogleSafeBrowsing`, and IIPC
urlcanon for the Semantic policy and IA canonicalization fixtures. The urlcanon
material is credited to Internet Archive and the National Library of Australia;
adapted code and tests retain their copyright and Apache 2.0 notices.

Compared with `v0.6.2`, Semantic's byte handling and normalization order are
closer to urlcanon. Tests use pinned upstream fixtures. Differences include
protection of encoded structural host delimiters and the use of this library's
WHATWG parser. See [Semantic](#semantic) and [IA](#ia) for policy-specific
differences, and
[test sources and licensing](#test-sources-and-licensing) for exact revisions
and attribution.

## Profiles

Each profile factory returns an independently configured profile. Call
`WhatWg()`, `WhatWgSortQuery()`, `GoogleSafeBrowsing()`, `Semantic()`, or `IA()`.

Choose a profile according to the distinctions your application needs to
preserve. Query order, credentials, path case, repeated slashes, and fragments
can affect server or application behavior; the table lists which each profile
normalizes.

| Profile | Behavior beyond standard WHATWG parsing |
| --- | --- |
| `WhatWg` | None. A starting profile for configuration and composition. |
| `WhatWgSortQuery` | Stable sorting by raw query name. |
| `GoogleSafeBrowsing` | Repeated escape normalization, host-dot cleanup, collapsed path slashes, port and fragment removal, and an `http` default scheme. |
| `Semantic` | Repeated escape normalization, host-dot cleanup, collapsed path slashes, credential and fragment removal, complete raw-term query sorting, an `http` default scheme, and special-scheme handling for gopher. |
| `IA` | IA-style canonicalization: credential and fragment removal, escape normalization, lowercasing, host massage, path-session removal, trailing-slash trimming, session-query filtering, and complete raw-term sorting. |

### WhatWg and WhatWgSortQuery

`WhatWg().Parse(raw)` behaves exactly like `url.Parse(raw)`. Use `url.Parse`
directly when no profile is needed. The named profile provides a uniform value
alongside the other profiles and a starting point for composition:

```go
profile := canonicalizer.WhatWg().Then(canonicalizer.RemoveFragment)
```

`New()` and the zero `Profile` have the same default parsing behavior.
`WhatWg().Canonicalize(u)` has no rules to run; it does not reparse or normalize an
existing URL. WHATWG normalization already happens in the
[URL parser](https://url.spec.whatwg.org/#url-parsing).

`WhatWgSortQuery` adds stable raw-name query sorting to WHATWG parsing and
preserves serialized terms.

### GoogleSafeBrowsing

Use this profile to prepare the canonical host, path, and query used in
[Google Safe Browsing URL matching](https://developers.google.com/safe-browsing/v4/urls-hashing#canonicalization).
Its purpose is to give alternative URL spellings the same matching
representation before generating Safe Browsing lookup expressions and hashes.
Expression generation (host suffixes and path prefixes), SHA-256 hashing, and
Safe Browsing API requests are separate steps for the caller to implement.

The default parser converts IDNs to ASCII and removes literal tabs, CR, and LF.
The profile removes the fragment before repeatedly percent-decoding the
components. It then trims and collapses host dots, normalizes IPv4 encodings
(including octal, hexadecimal, and shortened forms), and lowercases the host.
Path dot segments are resolved after decoding, followed by slash collapsing;
these path operations leave the query untouched. Finally, only bytes `<= 32`,
`>= 127`, `#`, and `%` are percent-escaped, using uppercase hex. The profile also
supplies `http` for missing schemes, ensures a hosted URL has a path, and removes
ports, matching the published examples.

This matching representation can expose decoded delimiters. For example,
`?q=%2526%253D` becomes `?q=&=`, and an encoded `?` in the path becomes a literal
`?` in `Pathname()`. Use the resulting URL record's component accessors when
constructing lookup expressions: serializing and reparsing the result can
change component boundaries. Keep the original URL for navigation. Scheme and
credentials remain available in the URL record; callers discard them when
constructing the host/path lookup expressions described by Google.

Use `GoogleSafeBrowsing().Parse` or `ParseRef` for the full operation order.
`Canonicalize` operates on the supplied URL's current components; parsing it
earlier with another parser may already have removed path segments. The
profile's `Parser()` preserves dot and empty segments, and
`WithParser` can override that configuration.

[`google_safe_browsing_test.go`](google_safe_browsing_test.go) checks all 33
published input/output pairs. Additional tests cover operation ordering, all
256 byte values in query escaping, IP metadata, and an existing `SearchParams`
object.

### Semantic

`Semantic` adapts
[IIPC urlcanon's semantic profile](https://github.com/iipc/urlcanon/blob/842afd4d84089cbbef87dd3286edaf053caedb0a/python/urlcanon/canon.py)
to this URL parser and its Rewrite API. Upstream defines
`semantic_precise` as browser-oriented normalization plus cleanup, then defines
`semantic` by adding fragment removal. The Java
[SemanticCanonicalizer](https://github.com/iipc/urlcanon/blob/842afd4d84089cbbef87dd3286edaf053caedb0a/java/src/org/netpreserve/urlcanon/SemanticCanonicalizer.java)
uses that same relationship.

The upstream profile was
[introduced for crawler deduplication and Wayback lookup](https://github.com/iipc/urlcanon/commit/06bae59728e444e4000cf68b50e2e5123a71dd00).
Its cleanup rules group URL spellings for those uses. `Semantic` is
experimental and uses the pinned urlcanon revision above as its compatibility
reference. The following sections describe its normalization rules and
differences from upstream.

[Brozzler uses urlcanon's semantic canonicalizer for crawl scoping](https://github.com/internetarchive/brozzler/blob/65736b8e5d5bd18d96533edae1f85900a05f09ea/job-conf.rst#automatic-scoping-based-on-seed-urls).
To derive the same seed scope, apply `Semantic`, remove the query, and serialize
the result as SSURT.
For example, its documented seed
`https://www.EXAMPLE.com:443/foo//bar?a=b&c=d#fdiap` produces the scope
`com,example,www,//https:/foo/bar`.

The upstream Python and Java suites share `semantic_precise.json`. The Go
tests pass all 280 inputs, interpreting JSON code points as bytes and removing
fragments from upstream expectations to test `Semantic`.

The parser uses `url.WithLaxHostParsing()` to retain invalid IPv4-like hosts
such as `09` as hostnames, without marking them as IPv4 addresses. Valid IPv4
addresses normalize normally. Standard `url.Parse` continues to reject invalid
IPv4 addresses. Malformed IPv6 hosts, including unclosed brackets, retain the
host spelling consumed by the parser without being marked as IPv6 addresses.
Valid IPv6 addresses still normalize normally. Validation events are retained,
and `WithFailOnValidationError()` prevents recovery. The same recovery applies
to other profiles configured with `WithLaxHostParsing()`.

`url.WithLaxHostParsing()` accepts forbidden opaque-host characters before
canonicalization. Semantic's `WithAllowForbiddenOpaqueHost` option defaults to
true, retaining non-delimiting punctuation such as `<`, `>`, `^`, and the
vertical bar to match upstream. Configure it when constructing the profile:

```go
profile := canonicalizer.Semantic(canonicalizer.WithAllowForbiddenOpaqueHost(false))
// custom://a^b/ becomes custom://a%5Eb/
```

This option controls escape normalization, not parsing. To reject invalid
hosts during parsing, supply a stricter parser with `WithParser`. Profiles
retaining forbidden punctuation can emit non-standard hostnames that require a
lax parser to read again. Structural URL delimiters, IPv6 brackets, controls,
spaces, and percent signs remain escaped when exposed by repeated decoding.
It defaults to false for `RepeatedPercentDecoding()` and true for `Semantic()`.
Pass it to the rule or profile factory being configured; `New()` and the zero
profile do not add escape normalization.

Like upstream, `Semantic` sorts complete serialized query terms after escape
normalization. For example, `?a=z&a=a` becomes `?a=a&a=z`. This also orders
terms with equal names, while retaining literal `+`, empty terms, and the
distinction between `flag` and `flag=`. Use `SortQuery` in a custom pipeline
when equal-name terms must retain their original order.

Empty hosts remain empty, including the host of `file:///`. A nonempty,
dot-only hostname becomes `0.0.0.0` after dot cleanup. For special schemes,
hostnames exposed by repeated decoding and dot cleanup receive domain and IPv4
normalization: `%2541.COM` becomes `a.com`, and `.127.1.` becomes `127.0.0.1`.
IDNA runs before dot cleanup when decoding exposes Unicode host bytes.
Malformed candidates retain their escaped spelling, and these normalization
attempts do not add validation events to the original URL. Opaque hosts preserve
permitted punctuation such as `!`, `_`, and `~`, along with forbidden
punctuation when `WithAllowForbiddenOpaqueHost(true)` is used. Special-scheme
hostnames also retain printable punctuation such as `_`, `!`, and `~`; their
escape set protects controls, space, non-ASCII bytes, percent signs, and
structural delimiters. This follows upstream
[`less_dumb_pct_encode`](https://github.com/iipc/urlcanon/blob/842afd4d84089cbbef87dd3286edaf053caedb0a/python/urlcanon/canon.py),
which uses the minimal Google byte escape set for hosts, with additional
delimiter protection here. Literal IPv6-style bracket syntax is retained,
including malformed hosts recovered by the parser; percent-decoded brackets
remain escaped.

The configured `WithAcceptInvalidCodepoints` parser option percent-encodes
literal invalid UTF-8 bytes without transcoding them. For example, both a
literal byte `0x91` and `%91` become `%91`; the valid UTF-8 encoding of U+0091
remains `%C2%91`. Invalid-byte validation is reported before recovery, and
fail-fast parsing takes precedence. Upstream's
[byte-handling discussion](https://github.com/iipc/webarchive-commons/pull/102)
also records the impact of inconsistent byte handling on archive index lookup.

Path normalization repeatedly decodes bytes, converts backslashes to slashes
for special schemes, resolves dot segments, collapses special-scheme slashes,
and then escapes the result. Non-special paths retain repeated slashes and
backslashes; dot resolution applies only to rooted paths. Rewriting preserves
whether the URL originally had an opaque or hierarchical path.

Semantic's parser enables `url.WithSkipDotSegmentNormalization()` and defers
slash collapsing to the pipeline. This preserves information that ordinary
parsing would discard before repeated decoding. `Semantic().Parser()` therefore
exposes paths before dot normalization. A replacement supplied with
`WithParser` must preserve those segments to get the same results.
`Canonicalize` only transforms the URL state supplied by its caller; it cannot
recover segments already removed by another parser. `ParseRef` resolves against
the uncanonicalized base, then canonicalizes the result once.

| Input | Semantic output |
| --- | --- |
| `http://example.org/a/%252e%252e/b` | `http://example.org/b` |
| `http://example.org/a/%252e%252e/../b` | `http://example.org/b` |
| `http://foo_bar/a//../b` | `http://foo_bar/a/b` |
| `custom://a%252Fb/` | `custom://a%2Fb/` |

In the last example, `%2F` stays escaped in the hostname; urlcanon emits a
literal slash. Keeping structural host delimiters escaped preserves the
hostname/path boundary when reparsed. Escape normalization operates on bytes
without character encoding conversion.

### IA

`IA` implements the Internet Archive-style policy used by the default SURT
parser. Tests use the `Canonicalizer.DefaultIA` and `Canonicalizer.IA` groups in
[IIPC urlcanon's SURT fixtures](https://github.com/iipc/urlcanon/blob/842afd4d84089cbbef87dd3286edaf053caedb0a/testdata/surt.json).
That family also appears in Internet Archive's
[Python SURT library](https://github.com/internetarchive/surt/blob/master/surt/DefaultIAURLCanonicalizer.py),
which identifies its default canonicalizer as a port of the Java
`DefaultIAURLCanonicalizer` from archive-commons.

One difference from Python SURT is above-root dot-segment handling. Its
[`normalizePath`](https://github.com/internetarchive/surt/blob/5e2c2dff9e684918ba7adf39523ffcf7ff4738cf/surt/GoogleURLCanonicalizer.py#L76-L114)
can retain `..` when there is no preceding path segment to remove. This
package's default IA parser applies WHATWG dot-segment normalization before
the canonicalization rules run:

| Original path | Python SURT normalized path | This package's IA path |
| --- | --- | --- |
| `/../path` | `/../path` | `/path` |
| `/a/../path` | `/path` | `/path` |
| `/a/../../path` | `/../path` | `/path` |

For example, `https://domain.com/../path` produces the SURT key
`com,domain)/path` here, rather than Python SURT's `com,domain)/../path`.
`TestIAAboveRootDotSegments` and the SURT canonicalization tests cover these
cases, including a single percent-encoded `..` segment (`%2e%2e`). Dot
resolution occurs during parsing; segments exposed later by repeated decoding
are not resolved again. A custom parser can change this behavior.

This matters for replay URLs: browsers treat an embedded original URL as part
of the outer URL's path. For example,
`https://archive.example/web/20200101000000/https://domain.com/../path`
normalizes to
`https://archive.example/web/20200101000000/https://path`, losing the embedded
hostname. Percent-encoding the dots as `%2e%2e` does not prevent that
[path normalization](https://url.spec.whatwg.org/#path-state).
Normalize the original path before embedding it in a replay URL.
Existing Python SURT indexes may also retain distinct keys for these spellings.

IIPC urlcanon's `aggressive` profile changes HTTPS to HTTP; `IA` retains the
scheme.

Configure IA's host massage and query filters when constructing the profile:

```go
filters := canonicalizer.DefaultSessionQueryFilters()
delete(filters, "phpsessid")
filters["tracking"] = func(name, value string, hasEquals bool) bool {
    return !strings.HasPrefix(name, "utm_")
}
profile := canonicalizer.IA(
    canonicalizer.WithHostMassage(false),
    canonicalizer.WithQueryFilters(filters),
)
```

The default filter names are `jsessionid`, `phpsessid`, `sid`, `aspsessionid`,
`cfid`, and `cftoken`. Each can be removed or replaced independently.
`jsessionid`, `phpsessid`, and `sid` remove values of 32 alphanumeric bytes;
ASP session names must have eight alphanumeric suffix bytes and a 24-byte
value. `cfid` and `cftoken` terms are removed regardless of value. Names are
matched without case sensitivity.

Filter callbacks receive normalized, lowercased serialized terms. **True keeps
a term**, following `Query.Filter`. All non-nil filters must keep a term;
they run in sorted key order, stopping at the first false result. Map keys
identify callbacks rather than restricting the names they inspect.

Encoded `=` characters remain escaped in names, including after repeated
decoding, so `cfid%3Dx=1` remains a distinct term from `cfid=x=1`. Values may
contain literal `=`; encoded `&` characters remain escaped in both fields.

`DefaultSessionQueryFilters()` returns a fresh, editable map. Pass a modified
collection to `WithQueryFilters`; the option snapshots its predicates when
created, and later map edits cannot affect existing or future profiles built
with that option. `WithQueryFilter(fn)` replaces the collection with one keep
predicate. Nil or empty collections disable filtering. Options apply in order,
so the last filter option wins. Nil options are ignored.

Host massage is enabled by default in `IA()`; use `WithHostMassage(false)`
to disable it. Factory options apply to the profile being constructed.

## Percent-encode presets

`canonicalizer` provides three percent-encode presets: `LaxPathPercentEncodeSet`,
`LaxQueryPercentEncodeSet`, and `RepeatedQueryPercentEncodeSet`. Pass them to
URL parser options or `url.PercentEncode`.

The presets have type `*url.PercentEncodeSet`. The lax path preset permits `<`,
`>`, and `^` beyond the standard path set; the lax query preset permits `"`
beyond the standard query set. The repeated-query preset escapes controls,
spaces, non-ASCII bytes, `#`, `%`, `&`, and `=` after decoding, preserving raw
term boundaries and literal `+`.

Use `Set` or `Clear` to derive a new set without changing the preset. See the
[parser configuration example](../README.md#percent-encode-set-customization).

## Rules and query semantics

Each rule has signature `func(*url.Url)`. Rules with configuration capture
arguments when constructed. Combine rules in execution order:

```go
profile := canonicalizer.New(
    canonicalizer.MassageHost,
    canonicalizer.FilterQuery(
        func(name, value string, hasEquals bool) bool { return name != "tracking" },
    ),
    canonicalizer.SortQuery,
)
```

`FilterQuery(keep)` constructs a rule using that keep predicate; nil produces
a no-op rule. Each rule captures its own configuration, so multiple filters in
one pipeline can have different predicates and run at different positions.
`RepeatedPercentDecoding(opts...)` also constructs a rule; use
`WithAllowForbiddenOpaqueHost(true)` to retain non-delimiting forbidden
opaque-host punctuation. Configure parsing separately through `WithParser`.

`RewriteHostname`, `RewritePathname`, `RewriteQueryString`, and `RewriteFragment`
adapt URL Rewrite callbacks into rules. They share the URL API's contract:
callbacks must return valid serialized components. Query and fragment callbacks
also return whether the component is present. Use `Then` to append rules.

`RewriteQuery(...url.QueryRewriter)` adapts the URL's query-term convenience
method into a rule. The rule snapshots the callback slice, parses once, runs
each non-nil callback in order, and commits once. Callbacks run for absent
queries too; `hasQuery` includes empty queries introduced by `?`, and the
returned boolean controls creation and removal independently
of the number of terms. For example:

```go
profile := canonicalizer.New(canonicalizer.RewriteQuery(
    func(_ *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
        query.SortAbsolute()
        return query, hasQuery
    },
))
```

Inside one rule, intermediate terms flow through callback arguments; URL
accessors and an existing `SearchParams` object receive the final result at
commit. Explicit setters take effect immediately, and the final returned
query takes precedence. Each subsequent rule sees the previous rule's committed
URL and form view.

Present-query identity callbacks still parse and serialize. Calls without
non-nil callbacks skip both operations.

Term-based query rules operate on `Query`. They retain literal `+`, empty
terms, and the distinction between `flag` and `flag=`. `SortQuery` stably compares raw
names; `SortQueryAbsolute` compares complete raw terms. Escape normalization
may change escape spelling while retaining term boundaries. For the form
semantics used in `v0.6.2`, use an explicit rule calling
`u.SearchParams().Sort()`.

GoogleSafeBrowsing uses `RewriteQueryString` to decode the entire query,
including encoded separators.

## Test sources and licensing

The upstream urlcanon and Brozzler material used here is licensed under Apache
2.0, as is this repository; see [LICENSE](../LICENSE). Copyright notices and
adaptation notices are retained in the adapted test files and fixture README.

| Tests | Source and adaptation |
| --- | --- |
| [Google Safe Browsing tests](google_safe_browsing_test.go) | The 33 examples from [Google's canonicalization documentation](https://developers.google.com/safe-browsing/v4/urls-hashing#canonicalization), plus regressions for operation order and byte escaping. Google's code samples are licensed under Apache 2.0. |
| `semantic_test.go` | urlcanon revision `842afd4d84089cbbef87dd3286edaf053caedb0a`, all 280 shared Python/Java `semantic_precise.json` cases; only fragment removal adapts expectations from `semantic_precise` to `semantic`. See the [fixture README](../testdata/iipc-urlcanon/README.md). |
| `java_test.go` | The same revision's [IpAddressesTest.java](https://github.com/iipc/urlcanon/blob/842afd4d84089cbbef87dd3286edaf053caedb0a/java/test/org/netpreserve/urlcanon/IpAddressesTest.java): all 11 successful IPv6 input/output pairs and five invalid inputs, adapted to bracketed URL hosts. Successful inputs run through Semantic; invalid inputs remain rejection tests with a standard parser injected into the profile. Separate tests cover lax recovery. |
| `path_dots_test.go` | The same revision's Apache-licensed `ResolvePathDotsTest.java` and unchanged `path_dots.json`: all 932 entries, including duplicate object keys, run against the dot-resolution primitive before slash collapsing. |
| `brozzler_test.go` | Brozzler revision `65736b8e5d5bd18d96533edae1f85900a05f09ea`, [test_frontier.py](https://github.com/internetarchive/brozzler/blob/65736b8e5d5bd18d96533edae1f85900a05f09ea/tests/test_frontier.py) (`test_basics`, `test_completed_page`, `test_hashtag_seed`) plus the documented seed example. Seven seed/scope pairs are adapted to Semantic, explicit query removal, and SSURT serialization; crawler scheduling and redirect policy are outside these tests. |

Run `go test ./canonicalizer ./surt` from the repository root.
