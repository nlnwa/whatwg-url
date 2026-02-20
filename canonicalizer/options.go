package canonicalizer

import (
	"github.com/nationallibraryofnorway/whatwg-url/internal"
	"github.com/nationallibraryofnorway/whatwg-url/url"
	"golang.org/x/text/encoding/charmap"
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

// WithRepeatedPercentDecodingEncoding sets the character encoding used when decoding percent-encoded octets
// during repeated percent decoding.
//
// Use this for legacy inputs where octets were generated from a non-UTF-8 encoding.
//
// This setting only affects canonicalizer repeated percent decoding and does not change core parser behavior.
//
// This API is EXPERIMENTAL.
func WithRepeatedPercentDecodingEncoding(encoding *charmap.Charmap) url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.RepeatedPercentDecodeEncoding = encoding
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

// WithHostTransformer applies a host transformation during canonicalization.
// The input and output are hostnames without port.
//
// This API is EXPERIMENTAL.
func WithHostTransformer(transforms ...func(u *url.Url, host string) string) url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.HostTransformers = append(p.HostTransformers, transforms...)
	})
}

// WithPathTransformer applies a path transformation during canonicalization.
//
// This API is EXPERIMENTAL.
func WithPathTransformer(transforms ...func(u *url.Url, path string) string) url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.PathTransformers = append(p.PathTransformers, transforms...)
	})
}

// WithQueryTransformer applies a query transformation during canonicalization.
// The query string is provided without the leading '?'.
//
// This API is EXPERIMENTAL.
func WithQueryTransformer(transforms ...func(u *url.Url, query string) string) url.ParserOption {
	return internal.NewCanonicalizerOption(func(p *internal.Canonicalizer) {
		p.QueryTransformers = append(p.QueryTransformers, transforms...)
	})
}
