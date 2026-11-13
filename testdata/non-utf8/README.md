# Common Crawl non-UTF-8 URL fixture

This fixture checks preservation of percent-encoded path and query bytes for
archival indexing. Extra escaping or Unicode replacement can cause index
writers and readers to produce different keys for the same input URL.

## Source

Sebastian Nagel published the
[list of non-UTF-8 URLs](https://commoncrawl-dev.s3.amazonaws.com/surt/non_utf8_urls_cc-main-2017-30.txt.xz)
on August 10, 2017, in
[Internet Archive SURT issue 19](https://github.com/internetarchive/surt/issues/19#issuecomment-321523765).
He extracted it from Common Crawl's July 2017 crawl, CC-MAIN-2017-30. The
checked-in `non_utf8_urls_cc-main-2017-30.txt` is its uncompressed contents.

The issue describes Python 3 replacing a legacy `%FC` path byte with U+FFFD,
while Python 2 retained the byte. Nagel explains that CDX portability requires
consistent keys across implementations and runtimes. His examples include
ISO-8859-2 and Windows-1251 as well as Latin-1, so conversion cannot assume one
source encoding. The related
[Common Crawl issue 6](https://github.com/commoncrawl/ia-web-commons/issues/6)
describes a different failure: adding an extra `%25` layer to existing escapes.

The fixture contains 533,168 ASCII serialized URL lines, 532,768 unique input
strings, and 48,681,027 bytes. Duplicates and original order are retained.
These are URLs with percent escapes, not arbitrary raw invalid-UTF-8 strings.
This fixture is maintained separately from the WPT refresh script.

SHA-256 of the uncompressed fixture, for manual verification:

```text
578a5182526de52e5cbce19131d7b892f344b1f719c3ad13c7aef5465ccf3525
```

```sh
sha256sum testdata/non-utf8/non_utf8_urls_cc-main-2017-30.txt
```

## Running the tests

Run from the repository root:

```sh
# Run the ordinary suite without long-running tests.
go test ./... -count=1

# Include long-running tests, including the full corpus.
go test -tags=long ./... -count=1

# Run just the corpus test.
go test -tags=long ./url -run '^TestNonUTF8Corpus$' -count=1

# Run the four upstream SURT examples.
go test ./surt -run '^TestUpstreamSURTNonUTF8Compatibility$' -count=1
```

`url/nonutf8_corpus_test.go` is excluded by default and enabled by the `long`
build tag. It contains one streaming test. For every row it checks
successful parsing, original versus serialized path/query bytes after exactly
one percent-decoding pass, stable serialization after reparsing, exact Query
round-trip, and unchanged URL serialization after reading SearchParams. It
allows the standard normalization of an empty HTTP path to `/`.

Focused parser, percent-helper, raw-query, component-rewriter, and SearchParams
regressions live in their ordinary test files. Preservation and form-semantics
examples remain in `url/example_test.go`.

## Upstream SURT regressions

[Internet Archive SURT PR 30](https://github.com/internetarchive/surt/pull/30)
adds four exact-output cases: UTF-8 and Latin-1 path variants, and the path and
query examples from Common Crawl issue 6. Their data pairs are retained verbatim
in `surt/compatibility_test.go`, with source head revision
`9a6717a64e374c4690cf6e28ae7d3504b3bda6cc`. They check SURT's lowercase escapes
and byte preservation. Four focused examples complement the corpus; they do
not provide equivalent real-world coverage or prove complete index parity.

## Historical component profile

An earlier investigation measured UTF-8 validity after exactly one
percent-decoding pass across all 533,168 rows on 2026-11-10:

| Path bytes | Query bytes | Rows |
| --- | --- | ---: |
| Invalid UTF-8 | Valid UTF-8 or absent | 316,575 |
| Invalid UTF-8 | Invalid UTF-8 | 345 |
| Valid UTF-8 | Invalid UTF-8 | 216,248 |
| Valid UTF-8 | Valid UTF-8 or absent | 0 |

These historical measurements include duplicates and count an absent query as
valid empty UTF-8. They are documentation, not current test assertions. UTF-8
validity does not identify the intended encoding or establish that bytes are
text. Literal regressions retain examples from lines 210, 212, 24487, and 46175.
At line 24487 one path contains recognizable UTF-8 `g%C3%A9n%C3%A9ral` alongside
legacy-looking `alg%E9rien`; line 46175 has `T%EAteDeFraisage` in its path and
valid UTF-8 `D%C3%A9fonceuse` in its query.

Earlier investigation runs also recorded 3,613 serialization spelling changes,
including 52 normalized empty HTTP paths, and 667 rows already containing
`%EF%BF%BD`. Form serialization changed spelling on 216,793 rows and invalid
UTF-8 was present in 216,593 queries. These are historical observations rather
than success criteria. The current corpus test compares bytes, so existing
replacement escapes are not treated as evidence of corruption by themselves.

## Historical SURT behavior

The December 2024 comparison in
[webarchive-commons PR 102](https://github.com/iipc/webarchive-commons/pull/102)
recorded three policies. These refer to the implementations discussed then,
not their current releases; the local row is an earlier full-corpus measurement.

| Policy and implementations | `%C3` | `%C3%23` | `%C3%80` |
| --- | --- | --- | --- |
| Extra escaping: jwarc, OutbackCDX, webarchive-commons at the time | `%25c3` | `%25c3%23` | `%c3%80` |
| Replacement: Java urlcanon | `%ef%bf%bd` | `%ef%bf%bd%23` | `%c3%80` |
| Byte preservation: Python urlcanon and surt, warcio.js | `%c3` | `%c3%23` | `%c3%80` |
| This repository's SURT, historical measurement | `%c3` | `%c3%23` | `%c3%80` |

Matching an existing index requires its canonicalization policy. Preserving
bytes does not reproduce keys generated using replacement or extra escaping.
The upstream discussion identifies compatibility consequences for existing
CDX files. This work provides no automatic migration or reindexing. Transcoding
remains deferred.
