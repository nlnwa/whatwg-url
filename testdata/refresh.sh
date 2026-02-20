#!/usr/bin/env bash

set -euo pipefail

base_url="https://raw.githubusercontent.com/web-platform-tests/wpt/master/url/resources"

files=(
  "setters_tests.json"
  "urltestdata.json"
  "toascii.json"
  "IdnaTestV2.json"
)

for file in "${files[@]}"; do
  curl -s -O "$base_url/$file"
done

commit_info=$(curl -s https://api.github.com/repos/web-platform-tests/wpt/commits/master)
commit_hash=$(echo "$commit_info" | jq -r .sha)
commit_date=$(echo "$commit_info" | jq -r .commit.committer.date)

echo "Commit: $commit_hash"
echo "Date:   $commit_date"
