package canonicalizer

import (
	"github.com/nationallibraryofnorway/whatwg-url/internal"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func WithNormalizePercentEncodedSpaceToPlus() url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.PercentEncodedSpaceToPlus = true
	})
}

// WithRemoveUserInfo removes username and password from url.
//
// This API is EXPERIMENTAL.
func WithRemoveUserInfo() url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.RemoveUserInfo = true
	})
}

// WithRemovePort always removes port from url. Default is to remove port if port is default port for scheme.
//
// This API is EXPERIMENTAL.
func WithRemovePort() url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.RemovePort = true
	})
}

// WithRemoveFragment removes the fragment part of the url.
//
// This API is EXPERIMENTAL.
func WithRemoveFragment() url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.RemoveFragment = true
	})
}

// WithRepeatedPercentDecoding enables repeated percent decoding.
// This is used to decode percent encoded characters recursively.
//
// For example, "%2520" -> "%20" -> " " (space).
//
// This API is EXPERIMENTAL.
func WithRepeatedPercentDecoding() url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.RepeatedPercentDecoding = true
	})
}

// WithRepeatedDecodeQueryPercentEncodeSet sets the percent encoding set applied after repeated percent decoding of the search parameters.
//
// This API is EXPERIMENTAL.
func WithRepeatedDecodeQueryPercentEncodeSet(encodeSet *url.PercentEncodeSet) url.ParserOption {
	return internal.NewCanonicalizerOption(func(o *internal.Canonicalizer) {
		o.RepeatedQueryPercentEncodeSet = encodeSet
	})
}

// WithDefaultScheme sets a scheme to add if url is missing scheme.
//
// This API is EXPERIMENTAL.
func WithDefaultScheme(scheme string) url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.DefaultScheme = scheme
	})
}

// WithSortQuery sets sort type for query parameters.
// [NoSort] = no sort, [SortKeys] = sort keys (but leave repeated keys in same order), [SortParameter] = sort key and value
//
// This API is EXPERIMENTAL.
func WithSortQuery(sortType internal.QuerySort) url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.SortQuery = sortType
	})
}
