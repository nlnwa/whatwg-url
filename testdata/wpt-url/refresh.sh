#!/usr/bin/env bash

set -euo pipefail

# Usage: ./refresh.sh [commit-or-branch]
revision="${1:-master}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

commit_info=$(curl -fsS "https://api.github.com/repos/web-platform-tests/wpt/commits/${revision}")
commit_hash=$(echo "$commit_info" | jq -er .sha)
commit_date=$(echo "$commit_info" | jq -er .commit.committer.date)
readme_date=$(echo "$commit_info" | jq -er '.commit.committer.date | fromdateiso8601 | strftime("%e %B %Y") | ltrimstr(" ")')
base_url="https://raw.githubusercontent.com/web-platform-tests/wpt/${commit_hash}/url/resources"

files=(
  "setters_tests.json"
  "urltestdata.json"
  "toascii.json"
  "IdnaTestV2.json"
)

for file in "${files[@]}"; do
  curl -fsSLo "${script_dir}/${file}" "$base_url/$file"
done

sed -i.bak -E \
  "s@^(Whatwg-url parser is up to date as of )\\[[^]]*\\]\\(https://github.com/web-platform-tests/wpt/tree/[^)]*\\)@\\1[${readme_date}](https://github.com/web-platform-tests/wpt/tree/${commit_hash})@" \
  "${script_dir}/../../README.md"
rm "${script_dir}/../../README.md.bak"

echo "Commit: $commit_hash"
echo "Date:   $commit_date"
