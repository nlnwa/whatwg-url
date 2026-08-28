# IIPC urlcanon test data

`surt.json` and `ssurt.json` are vendored from
[`iipc/urlcanon`](https://github.com/iipc/urlcanon/tree/842afd4d84089cbbef87dd3286edaf053caedb0a/testdata)
at commit `842afd4d84089cbbef87dd3286edaf053caedb0a`.

The upstream project is Copyright 2016–2018 Internet Archive and Copyright
2016–2020 National Library of Australia, licensed under the Apache License 2.0.

Only groups relevant to the public APIs in `url` and `surt` are executed.
The remaining groups describe canonicalization profiles or the IIPC raw SURT
formatter, which are intentionally outside this module's API.

Run `./refresh.sh [revision]` from this directory to refresh the files. The
default revision is the pinned commit above; pass a commit hash explicitly
when intentionally updating the snapshot, then update this README and the
revision in `surt/iipc_urlcanon_test.go`.
