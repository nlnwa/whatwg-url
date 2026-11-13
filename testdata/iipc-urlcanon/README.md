# IIPC urlcanon test data

`surt.json`, `ssurt.json`, `semantic_precise.json`, and `path_dots.json` are copied
unchanged from
[`iipc/urlcanon`](https://github.com/iipc/urlcanon/tree/842afd4d84089cbbef87dd3286edaf053caedb0a/testdata)
at commit `842afd4d84089cbbef87dd3286edaf053caedb0a`.

The upstream project is Copyright 2016–2018 Internet Archive and Copyright
2016–2020 National Library of Australia, licensed under the Apache License 2.0.

The `canonicalizer` tests execute the IA canonicalization groups and related
primitive vectors. The `surt` tests cover SURT and SSURT serialization.
The `google` and `googleSupplementalFromSurt` groups and the IIPC raw SURT
formatter groups are not executed. Google's 33 published canonicalization
examples run separately in
[`google_safe_browsing_test.go`](../../canonicalizer/google_safe_browsing_test.go),
with their exact inputs and outputs verified against Google's original HTML
code block. All 33 also preserve this repository's original examples. See the
[profile documentation](../../canonicalizer/README.md#googlesafebrowsing).

`semantic_precise.json` is shared by upstream's Python `test_semantic_precise`
and Java `SemanticPreciseTest`. Its JSON strings represent bytes: code points
U+0000 through U+00FF map to the corresponding single bytes, rather than being
UTF-8 encoded. The Go test removes fragments from expected outputs to exercise
`Semantic` instead of `semantic_precise`.

All 280 Semantic cases retain upstream expectations apart from fragment
removal. There are no case-specific overrides or skipped cases. The profile
uses lax IP-host recovery and configurable opaque-host punctuation retention;
see the [profile documentation](../../canonicalizer/README.md#semantic).

`path_dots.json` is shared with upstream's Java `ResolvePathDotsTest`. All 932
entries run against the dot-resolution primitive before slash collapsing. The
test uses streaming JSON decoding to preserve duplicate object keys. The
adaptation retains the Java test's Copyright 2016 National Library of Australia
notice and Apache 2.0 license.

Run `./refresh.sh [revision]` from this directory to refresh the files. The
default revision is the pinned commit above; pass a commit hash explicitly
when intentionally updating the snapshot, then update this README and the
revision in both `canonicalizer/iipc_test.go` and `surt/iipc_urlcanon_test.go`.
