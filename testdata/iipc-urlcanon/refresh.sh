#!/usr/bin/env bash

set -euo pipefail

revision="${1:-842afd4d84089cbbef87dd3286edaf053caedb0a}"
base_url="https://raw.githubusercontent.com/iipc/urlcanon/${revision}/testdata"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for file in surt.json ssurt.json semantic_precise.json path_dots.json; do
  curl -fsSLo "${script_dir}/${file}" "${base_url}/${file}"
done

echo "Refreshed IIPC urlcanon test data from revision ${revision}"
