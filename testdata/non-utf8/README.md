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
regressions run in the ordinary suite. See `url/example_test.go` for byte
preservation and form-decoding examples.

## Upstream SURT regressions

[Internet Archive SURT PR 30](https://github.com/internetarchive/surt/pull/30)
adds four exact-output cases: UTF-8 and Latin-1 path variants, and the path and
query examples from Common Crawl issue 6. Their data pairs are retained verbatim
in `surt/compatibility_test.go`, with source head revision
`9a6717a64e374c4690cf6e28ae7d3504b3bda6cc`. They check SURT's lowercase escapes
and byte preservation.

Matching an existing index requires the canonicalization policy used to create
its keys. Byte preservation, Unicode replacement, and extra percent-escaping
produce different keys for some inputs. See the comparison in
[webarchive-commons PR 102](https://github.com/iipc/webarchive-commons/pull/102).
