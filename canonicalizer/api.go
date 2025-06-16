package canonicalizer

import (
	"github.com/nationallibraryofnorway/whatwg-url/internal"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func New(opts ...url.ParserOption) url.Parser {
	parser := internal.NewParser()
	for _, opt := range opts {
		opt.Apply(parser)
	}

	profile := internal.NewCanonicalizer(parser)

	for _, opt := range opts {
		if o, ok := opt.(internal.CanonicalizerOption); ok {
			o.ApplyProfile(profile)
		}
	}
	return profile
}

var (
	LaxPathPercentEncodeSet       = internal.LaxPathPercentEncodeSet
	LaxQueryPercentEncodeSet      = internal.LaxQueryPercentEncodeSet
	RepeatedQueryPercentEncodeSet = internal.RepeatedQueryPercentEncodeSet
)

const (
	// Do not sort query.
	NoSort = internal.NoSort
	// Stable sort on query parameter keys.
	SortKeys = internal.SortKeys
	// Sort on entire query parameter.
	SortParameter = internal.SortParameter
)
