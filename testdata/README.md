# Test fixtures

Tests stay in their normal Go packages; this directory holds their input data.

| Directory | Contents and maintenance |
| --- | --- |
| [wpt-url](wpt-url/) | WPT parser, setter, and IDNA JSON fixtures. `wpt-url/refresh.sh [revision]` downloads beside the script and updates the root README's WPT revision, regardless of the caller's working directory. |
| [non-utf8](non-utf8/README.md) | Common Crawl collection of ASCII serialized URLs with percent-encoded bytes; provenance, manual checksum verification, and the full-corpus test command. |
| [wpt-searchparams](wpt-searchparams/README.md) | Basic, manually maintained Go-port fixtures with pinned sources and documented coverage; no JavaScript harness. |
| [iipc-urlcanon](iipc-urlcanon/README.md) | Independent SURT and SSURT reference fixtures and refresh instructions. |

The URL tests exercise parsing, serialization, queries, and SearchParams.
One streaming test in `url/nonutf8_corpus_test.go` checks every Common Crawl
fixture row when enabled with the `long` build tag. Focused regressions live
in their usual test files, with executable examples in `url/example_test.go`.
The SURT package
independently tests its ordinary SURT/SSURT fixtures and four upstream
non-UTF-8 preservation cases.
