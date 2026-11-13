package canonicalizer

import (
	"maps"
	"slices"
)

// QueryFilter returns true to keep a serialized query term, following
// url.Query.Filter. It does not receive form-decoded names or values.
type QueryFilter func(name, value string, hasEquals bool) bool

type decodeOptions struct {
	allowForbiddenOpaqueHost bool
}

// DecodeOption configures escape normalization in RepeatedPercentDecoding or
// Semantic. Options apply in declaration order; nil options are ignored.
type DecodeOption func(*decodeOptions)

// WithAllowForbiddenOpaqueHost retains non-delimiting forbidden opaque-host
// punctuation during escape normalization. Structural delimiters, brackets,
// controls, spaces, and percent signs remain escaped when exposed by decoding.
// Enabled rules can emit non-standard hosts that require lax parsing to read
// again. Parsing itself is configured separately through WithParser.
func WithAllowForbiddenOpaqueHost(allow bool) DecodeOption {
	return func(options *decodeOptions) { options.allowForbiddenOpaqueHost = allow }
}

func configureDecoding(allow bool, opts []DecodeOption) decodeOptions {
	options := decodeOptions{allowForbiddenOpaqueHost: allow}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

type iaOptions struct {
	hostMassage bool
	queryFilter QueryFilter
}

// IAOption configures rules constructed by IA. Options apply in declaration
// order; nil options are ignored. Each factory call captures its own settings.
type IAOption func(*iaOptions)

// WithHostMassage enables or disables removal of one www or wwwN host label in
// IA. Host massage is enabled by default.
func WithHostMassage(enabled bool) IAOption {
	return func(options *iaOptions) { options.hostMassage = enabled }
}

// WithQueryFilter replaces IA's session-query filters with one keep predicate.
// A nil predicate disables filtering.
func WithQueryFilter(keep QueryFilter) IAOption {
	return func(options *iaOptions) { options.queryFilter = keep }
}

// WithQueryFilters replaces IA's filters with a snapshot of named predicates.
// Map keys identify predicates rather than restricting which terms they inspect.
// Predicates run in sorted key order, stopping at the first false result. Nil
// predicates are ignored; nil or empty maps disable filtering. Later changes
// to the supplied map do not affect this option or profiles built from it.
func WithQueryFilters(filters map[string]QueryFilter) IAOption {
	return WithQueryFilter(combineQueryFilters(filters))
}

func combineQueryFilters(filters map[string]QueryFilter) QueryFilter {
	keys := slices.Sorted(maps.Keys(filters))
	var predicates []QueryFilter
	for _, key := range keys {
		if keep := filters[key]; keep != nil {
			predicates = append(predicates, keep)
		}
	}
	if len(predicates) == 0 {
		return nil
	}
	return func(name, value string, hasEquals bool) bool {
		for _, keep := range predicates {
			if !keep(name, value, hasEquals) {
				return false
			}
		}
		return true
	}
}
