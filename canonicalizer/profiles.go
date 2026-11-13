package canonicalizer

import (
	"regexp"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// WhatWg returns a profile performing standard WHATWG parsing without additional
// rules. Its Parse method behaves like url.Parse. Use the profile as a starting
// point for configuration and composition with Then.
func WhatWg() Profile { return New() }

// WhatWgSortQuery returns a profile performing WHATWG parsing and stable raw
// query-name sorting. It preserves terms without form decoding or serialization.
func WhatWgSortQuery() Profile { return New(SortQuery) }

// GoogleSafeBrowsing returns a profile for Google Safe Browsing-style matching.
// It normalizes repeated escapes and host dots, collapses path slashes, removes
// ports and fragments, and supplies http for missing schemes. It applies Google's
// byte escape set, including to decoded query separators, for matching rather
// than navigation. It does not generate lookup expressions or hashes.
func GoogleSafeBrowsing() Profile {
	return New(
		RemoveFragment,
		googleRepeatedPercentDecoding,
		RemovePort,
	).WithParser(url.NewParser(
		url.WithLaxHostParsing(),
		url.WithPathPercentEncodeSet(LaxPathPercentEncodeSet),
		url.WithQueryPercentEncodeSet(LaxQueryPercentEncodeSet),
		url.WithSkipDotSegmentNormalization(),
		url.WithAcceptInvalidCodepoints(),
		url.WithPercentEncodeSinglePercentSign(),
	)).WithDefaultScheme("http")
}

// Semantic returns a profile that normalizes repeated escapes and host dots,
// collapses path slashes, removes credentials and fragments, and sorts complete
// raw query terms.
// It preserves bytes, supplies http for missing schemes, and recognizes gopher
// as a special scheme. Its parser recovers invalid IP hosts and forbidden
// opaque-host characters; opaque-host punctuation is retained by default.
// Dot resolution and special-scheme slash collapsing follow repeated decoding.
func Semantic(opts ...DecodeOption) Profile {
	options := configureDecoding(true, opts)
	return New(
		func(u *url.Url) { semanticRepeatedPercentDecoding(u, options) },
		RemoveUserInfo,
		SortQueryAbsolute,
		RemoveFragment,
	).WithParser(url.NewParser(
		url.WithLaxHostParsing(),
		url.WithPathPercentEncodeSet(LaxPathPercentEncodeSet),
		url.WithQueryPercentEncodeSet(LaxQueryPercentEncodeSet),
		url.WithSkipDotSegmentNormalization(),
		url.WithAcceptInvalidCodepoints(),
		url.WithPercentEncodeSinglePercentSign(),
		url.WithAllowSettingPathForNonBaseUrl(),
		url.WithSpecialSchemes(map[string]uint16{"gopher": 70}),
	)).WithDefaultScheme("http")
}

var consecutiveDots = regexp.MustCompile(`\.{2,}`)

func normalizeHostDots(_ *url.Url, host string) string {
	return consecutiveDots.ReplaceAllString(strings.Trim(host, "."), ".")
}
