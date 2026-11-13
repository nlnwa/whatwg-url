package canonicalizer

import (
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// Google's final escaping rule: bytes <= 32, >= 127, '#' and '%'. This is a
// matching representation; decoded URL or query delimiters remain literal.
var googleEscapeSet = url.NewPercentEncodeSet(0x21, '#', '%')

// Reuse the URL library's IDNA and IPv4 algorithms after repeated decoding and
// host-dot cleanup expose a new hostname. Parsing a separate candidate keeps
// its validation events out of the original URL's diagnostic history.
var googleHostParser = url.NewParser(url.WithReportValidationErrors(false))

func googleRepeatedPercentDecoding(u *url.Url) {
	u.RewriteHostname(func(u *url.Url, original string) string {
		// Parsed bracketed hosts already have IPv6 normalization and metadata.
		// Preserve the syntax of both valid IPv6 and laxly recovered hosts.
		if strings.HasPrefix(original, "[") {
			return normalizeEscapes(strings.ToLower(url.PercentEncode(original, googleEscapeSet)), googleEscapeSet)
		}
		host := url.PercentDecodeRepeated(original)
		for i := 0; i < len(host); i++ {
			if host[i] >= 0x80 {
				// IDNA can expose dot separators or empty labels. Complete it
				// before dot cleanup and IPv4 parsing. A nonnumeric suffix
				// prevents the candidate parser from interpreting IPv4 early.
				candidate := url.PercentEncode(host, opaqueHostEscapeSet)
				if parsed, err := googleHostParser.Parse("http://" + candidate + ".invalid/"); err == nil {
					host = strings.TrimSuffix(parsed.Hostname(), ".invalid")
				}
				break
			}
		}
		host = normalizeHostDots(u, host)
		// Escape non-ASCII bytes before lowercasing so invalid UTF-8 is not
		// replaced by U+FFFD. The candidate parser decodes these for IDNA.
		host = strings.ToLower(url.PercentEncode(host, googleEscapeSet))
		if host != original && host != "" {
			// Escape structural host delimiters only for this parsing attempt.
			// The final representation below uses Google's minimal escape set.
			candidate := url.PercentEncode(url.PercentDecodeRepeated(host), opaqueHostEscapeSet)
			if parsed, err := googleHostParser.Parse("http://" + candidate + "/"); err == nil {
				host = parsed.Hostname()
			}
		}
		return normalizeEscapes(host, googleEscapeSet)
	})
	u.RewritePathname(func(u *url.Url, path string) string {
		path = url.PercentDecodeRepeated(path)
		// The profile parser preserves segments and empty segments so that
		// both operations happen after decoding, in the documented order.
		path = resolvePathDots(path, false)
		path = consecutiveSlashes.ReplaceAllString(path, "/")
		if path == "" && u.Hostname() != "" {
			path = "/"
		}
		return url.PercentEncode(path, googleEscapeSet)
	})
	u.RewriteQueryString(func(_ *url.Url, query string, hasQuery bool) (string, bool) {
		// Decode the entire serialized query. Google's rules do not preserve
		// escaped '&' or '=' and do not apply path cleanup or form decoding.
		return normalizeEscapes(query, googleEscapeSet), hasQuery
	})
}
