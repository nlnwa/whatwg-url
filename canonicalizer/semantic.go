// Copyright (C) 2016-2018 Internet Archive
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Path dot resolution is adapted from urlcanon's resolve_path_dots at revision
// 842afd4d84089cbbef87dd3286edaf053caedb0a.

package canonicalizer

import (
	"regexp"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

var semanticHostEscapeSet = url.C0OrSpacePercentEncodeSet.Set('#', '%', '/', ':', '?', '@', '[', '\\', ']')
var consecutiveSlashes = regexp.MustCompile(`/+`)

// Candidate parsing reuses domain and IPv4 normalization without adding its
// validation events to the original URL. Failed candidates retain lax recovery.
var semanticHostParser = url.NewParser(url.WithReportValidationErrors(false))

func semanticRepeatedPercentDecoding(u *url.Url, options decodeOptions) {
	u.RewriteHostname(func(u *url.Url, original string) string {
		// Literal bracket syntax belongs to the parsed host, including lax
		// recovery. Brackets exposed by decoding are ordinary escaped bytes.
		if original == "" || strings.HasPrefix(original, "[") {
			return original
		}
		set := semanticHostEscapeSet
		if !u.IsSpecialScheme() {
			set = opaqueHostEscapeSet
			if options.allowForbiddenOpaqueHost {
				set = laxOpaqueHostEscapeSet
			}
		}
		host := normalizeEscapes(original, set)
		if u.IsSpecialScheme() && host != original && strings.Contains(host, "%") {
			// IDNA can expose more dots. A nonnumeric suffix defers IPv4
			// parsing until those dots have been normalized below.
			if parsed, err := semanticHostParser.Parse("http://" + host + ".invalid/"); err == nil {
				host = strings.TrimSuffix(parsed.Hostname(), ".invalid")
			}
		}
		host = normalizeHostDots(u, host)
		if host == "" {
			host = "0.0.0.0"
		}
		if u.IsSpecialScheme() && host != original {
			// Structural delimiters remain escaped in host, so candidate
			// parsing cannot reinterpret them as other URL components.
			if parsed, err := semanticHostParser.Parse("http://" + host + "/"); err == nil {
				host = parsed.Hostname()
			}
			if u.Scheme() == "file" && host == "localhost" {
				host = ""
			}
		}
		return host
	})
	u.RewritePathname(func(u *url.Url, path string) string {
		path = url.PercentDecodeRepeated(path)
		if u.IsSpecialScheme() {
			path = strings.ReplaceAll(path, "\\", "/")
		}
		path = resolvePathDots(path, u.IsSpecialScheme())
		if u.IsSpecialScheme() {
			path = consecutiveSlashes.ReplaceAllString(path, "/")
		}
		return url.PercentEncode(path, LaxPathPercentEncodeSet.Set('%'))
	})
	rewriteDecodedQueryAndFragment(u)
}

// resolvePathDots removes dot segments from rooted paths while retaining
// separators and empty segments. Special paths also recognize backslashes.
func resolvePathDots(path string, special bool) string {
	separator := func(b byte) bool { return b == '/' || special && b == '\\' }
	if path == "" || !separator(path[0]) {
		return path
	}
	parts := []string{path[:1]}
	for start := 1; ; {
		end := start
		for end < len(path) && !separator(path[end]) {
			end++
		}
		segment := path[start:end]
		dots := strings.ReplaceAll(strings.ToLower(segment), "%2e", ".")
		switch dots {
		case "..":
			if len(parts) > 1 {
				parts = parts[:len(parts)-1]
			}
			if len(parts) > 1 {
				parts = parts[:len(parts)-1]
			}
		case ".":
		default:
			parts = append(parts, segment)
			if end < len(path) {
				parts = append(parts, path[end:end+1])
			}
		}
		if end == len(path) {
			break
		}
		start = end + 1
	}
	return strings.Join(parts, "")
}
