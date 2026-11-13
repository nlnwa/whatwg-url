package canonicalizer

import (
	"slices"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// RemoveUserInfo removes the username and password.
func RemoveUserInfo(u *url.Url) {
	if u != nil {
		u.SetUsername("")
		u.SetPassword("")
	}
}

// RemovePort removes even a non-default port.
func RemovePort(u *url.Url) {
	if u != nil {
		u.SetPort("")
	}
}

// RemoveFragment removes the fragment, including a present-but-empty fragment.
func RemoveFragment(u *url.Url) {
	u.RewriteFragment(func(_ *url.Url, _ string, _ bool) (string, bool) {
		return "", false
	})
}

// SortQuery stably sorts serialized query terms by their raw names. It retains
// duplicate order, empty terms, bare names, literal '+', and percent spelling.
func SortQuery(u *url.Url) {
	sortQuery(u, false)
}

// SortQueryAbsolute stably sorts complete serialized query terms.
func SortQueryAbsolute(u *url.Url) {
	sortQuery(u, true)
}

// RemoveEmptyQueryEquals converts empty-valued query terms such as "flag=" to
// bare terms such as "flag". It preserves nonempty values, duplicate order,
// encoded bytes, empty terms, and query presence without form decoding.
func RemoveEmptyQueryEquals(u *url.Url) {
	rewritePresentQuery(u, func(query url.Query) (url.Query, bool) {
		for i := range query {
			if query[i].Value == "" {
				query[i].HasEquals = false
			}
		}
		return query, true
	})
}

// MassageHost removes one leading www or wwwN domain label. It preserves IP
// addresses and does not otherwise normalize hosts.
func MassageHost(u *url.Url) {
	u.RewriteHostname(func(_ *url.Url, host string) string { return massageHost(host) })
}

// FilterQuery constructs a rule applying keep to raw serialized query terms.
// It preserves query presence unless every term is removed. A nil predicate
// returns a no-op rule. True keeps a term; terms are not form-decoded.
func FilterQuery(keep QueryFilter) Rule {
	if keep == nil {
		return func(*url.Url) {}
	}
	return func(u *url.Url) {
		rewritePresentQuery(u, func(query url.Query) (url.Query, bool) {
			query.Filter(keep)
			return query, len(query) != 0
		})
	}
}

func sortQuery(u *url.Url, absolute bool) {
	rewritePresentQuery(u, func(query url.Query) (url.Query, bool) {
		if absolute {
			query.SortAbsolute()
		} else {
			query.Sort()
		}
		return query, true
	})
}

// rewritePresentQuery transforms present queries and synchronizes the form view at
// this rule boundary. An empty query still contains one empty raw term; the
// transform decides whether to retain it or remove the query altogether.
func rewritePresentQuery(u *url.Url, transform func(url.Query) (url.Query, bool)) {
	u.RewriteQuery(func(_ *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
		if !hasQuery {
			return query, false
		}
		return transform(query)
	})
}

// RepeatedPercentDecoding repeatedly decodes escapes and re-encodes bytes in
// the hostname, path, query terms, and fragment. It preserves IPv6 syntax,
// optional-component presence, and raw query structure. No character encoding
// conversion or form decoding is performed.
// Hostname escaping follows the scheme, retaining permitted opaque-host
// punctuation. WithAllowForbiddenOpaqueHost also retains non-delimiting forbidden
// punctuation; structural delimiters and IPv6 brackets remain escaped.
func RepeatedPercentDecoding(opts ...DecodeOption) Rule {
	options := configureDecoding(false, opts)
	return func(u *url.Url) { repeatedPercentDecoding(u, url.PathPercentEncodeSet, options) }
}

func repeatedPercentDecoding(u *url.Url, pathSet *url.PercentEncodeSet, options decodeOptions) {
	u.RewriteHostname(func(u *url.Url, host string) string {
		if strings.HasPrefix(host, "[") {
			return host
		}
		set := opaqueHostEscapeSet
		if options.allowForbiddenOpaqueHost {
			set = laxOpaqueHostEscapeSet
		}
		if u.IsSpecialScheme() {
			set = url.HostPercentEncodeSet.Set('%')
		}
		return normalizeEscapes(host, set)
	})
	u.RewritePathname(func(u *url.Url, path string) string {
		set := pathSet.Set('%')
		if u.IsSpecialScheme() {
			set = set.Set('\\')
		}
		return normalizeEscapes(path, set)
	})
	rewriteDecodedQueryAndFragment(u)
}

func rewriteDecodedQueryAndFragment(u *url.Url) {
	rewritePresentQuery(u, func(query url.Query) (url.Query, bool) {
		for i := range query {
			query[i].Name = normalizeEscapes(query[i].Name, RepeatedQueryPercentEncodeSet)
			query[i].Value = normalizeEscapes(query[i].Value, RepeatedQueryPercentEncodeSet)
		}
		return query, true
	})
	u.RewriteFragment(func(_ *url.Url, fragment string, hasFragment bool) (string, bool) {
		return normalizeEscapes(fragment, url.FragmentPercentEncodeSet.Set('%')), hasFragment
	})
}

// Opaque hosts retain permitted punctuation while encoded forbidden host bytes
// remain escaped, including bytes exposed by repeated percent decoding.
var opaqueHostEscapeSet = url.C0OrSpacePercentEncodeSet.Set('#', '%', '/', ':', '<', '>', '?', '@', '[', '\\', ']', '^', '|')
var laxOpaqueHostEscapeSet = url.C0OrSpacePercentEncodeSet.Set('#', '%', '/', ':', '?', '@', '[', ']')

func normalizeEscapes(value string, set *url.PercentEncodeSet) string {
	return url.PercentEncode(url.PercentDecodeRepeated(value), set)
}

// RewriteHostname adapts serialized hostname rewriters into a Rule. The
// callbacks have the same contract as url.Url.RewriteHostname.
func RewriteHostname(rewriters ...url.ComponentRewriter) Rule {
	rewriters = slices.Clone(rewriters)
	return func(u *url.Url) { u.RewriteHostname(rewriters...) }
}

// RewritePathname adapts serialized path rewriters into a Rule. The callbacks
// have the same contract as url.Url.RewritePathname.
func RewritePathname(rewriters ...url.ComponentRewriter) Rule {
	rewriters = slices.Clone(rewriters)
	return func(u *url.Url) { u.RewritePathname(rewriters...) }
}

// RewriteQueryString adapts optional serialized query rewriters into a Rule. The
// callbacks have the same contract as url.Url.RewriteQueryString.
func RewriteQueryString(rewriters ...url.OptionalComponentRewriter) Rule {
	rewriters = slices.Clone(rewriters)
	return func(u *url.Url) { u.RewriteQueryString(rewriters...) }
}

// RewriteQuery adapts lossless query-term rewriters into a Rule. Callbacks
// share the contract of url.Url.RewriteQuery: terms flow through callbacks,
// with one final commit and SearchParams synchronization for this rule.
func RewriteQuery(rewriters ...url.QueryRewriter) Rule {
	rewriters = slices.Clone(rewriters)
	return func(u *url.Url) { u.RewriteQuery(rewriters...) }
}

// RewriteFragment adapts optional serialized fragment rewriters into a Rule.
// The callbacks have the same contract as url.Url.RewriteFragment.
func RewriteFragment(rewriters ...url.OptionalComponentRewriter) Rule {
	rewriters = slices.Clone(rewriters)
	return func(u *url.Url) { u.RewriteFragment(rewriters...) }
}
