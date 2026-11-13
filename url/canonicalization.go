package url

// ComponentRewriter rewrites an already-serialized URL component. The value
// returned by a rewriter must itself be a valid serialized value for the
// component being rewritten; it is stored without parsing or percent-encoding.
//
// This API is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
type ComponentRewriter func(*Url, string) string

// OptionalComponentRewriter rewrites an already-serialized optional URL
// component. The boolean reports and controls whether the component is present,
// distinguishing an absent component from a present-but-empty one. The string
// returned by a rewriter must itself be a valid serialized component value; it
// is stored without parsing or percent-encoding.
//
// This API is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
type OptionalComponentRewriter func(u *Url, value string, hasComponent bool) (string, bool)

// QueryRewriter rewrites lossless serialized query terms. hasQuery reports
// whether the URL has a query, including an empty query introduced by '?'.
// The returned boolean controls query presence independently of the number
// of terms.
// Names and values must already be serialized; they are not percent-encoded
// or form-decoded by RewriteQuery. Terms can be modified in place or
// replaced; later edits to retained slices do not change the committed URL.
//
// This API is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
type QueryRewriter func(u *Url, query Query, hasQuery bool) (Query, bool)

// RewriteHostname applies rewriters from left to right to the serialized
// hostname, excluding the port. Nil rewriters are ignored. URLs without a host
// are left unchanged.
//
// Rewriters must return valid serialized hostnames. Returned values are stored
// without another parsing or encoding pass. This API is an EXPERIMENTAL
// extension and is not part of the WHATWG URL API.
func (u *Url) RewriteHostname(rewriters ...ComponentRewriter) {
	if u == nil || !u.hasHost {
		return
	}

	host := u.host
	for _, rewrite := range rewriters {
		if rewrite != nil {
			host = rewrite(u, host)
		}
	}
	if host != u.host {
		u.host = host
		u.refreshIPMetadata()
	}
}

// RewritePathname applies rewriters from left to right to the serialized path.
// Nil rewriters are ignored. Whether the path is opaque or hierarchical is
// preserved.
//
// Rewriters must return valid serialized paths. Returned values are stored
// without another parsing or encoding pass. This API is an EXPERIMENTAL
// extension and is not part of the WHATWG URL API.
func (u *Url) RewritePathname(rewriters ...ComponentRewriter) {
	if u == nil {
		return
	}

	pathname := u.path.String()
	for _, rewrite := range rewriters {
		if rewrite != nil {
			pathname = rewrite(u, pathname)
		}
	}
	if pathname != u.path.String() {
		u.path.setSerialized(pathname)
	}
}

// RewriteQueryString applies rewriters from left to right to the serialized query,
// excluding '?'. Nil rewriters are ignored. The boolean distinguishes an absent
// query from a present-but-empty query.
//
// Rewriters must return valid serialized queries. Returned values are stored
// without another parsing or encoding pass. If SearchParams has already been
// obtained, its contents are refreshed without replacing the object. This API
// is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
func (u *Url) RewriteQueryString(rewriters ...OptionalComponentRewriter) {
	if u == nil {
		return
	}

	query, hasQuery := u.query, u.hasQuery
	for _, rewrite := range rewriters {
		if rewrite != nil {
			query, hasQuery = rewrite(u, query, hasQuery)
		}
	}
	if !hasQuery {
		query = ""
		if !u.hasFragment {
			u.path.stripTrailingSpacesIfOpaque()
		}
	}
	if query != u.query || hasQuery != u.hasQuery {
		u.query, u.hasQuery = query, hasQuery
		u.refreshSearchParams()
	}
}

// RewriteQuery parses the query once, applies each non-nil rewriter once
// from left to right, then serializes and commits the final result through
// RewriteQueryString. A nil URL is ignored.
//
// Callbacks also run for absent queries, initially receiving nil terms and
// false. A present empty query initially has one empty term. Only the returned
// boolean controls presence: nil or empty terms with true produce an empty
// query, while false removes it regardless of the returned terms.
//
// Intermediate terms and presence flow through callback arguments; URL
// accessors and SearchParams do not reflect them until the final commit.
// Setters called by callbacks take effect immediately, but the final returned
// query takes precedence. An existing SearchParams object is refreshed only
// if the final query differs from the URL's state after callbacks.
//
// Calls without non-nil callbacks skip parsing and serialization, retaining
// RewriteQueryString's opaque-path cleanup. Present-query identity callbacks still
// require parsing and serialization, unlike string-based RewriteQueryString.
// This API is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
func (u *Url) RewriteQuery(rewriters ...QueryRewriter) {
	if u == nil {
		return
	}
	for len(rewriters) > 0 && rewriters[0] == nil {
		rewriters = rewriters[1:]
	}
	if len(rewriters) == 0 {
		u.RewriteQueryString()
		return
	}
	u.RewriteQueryString(func(u *Url, raw string, hasQuery bool) (string, bool) {
		var query Query
		if hasQuery {
			query = ParseQuery(raw)
		}
		for _, rewrite := range rewriters {
			if rewrite != nil {
				query, hasQuery = rewrite(u, query, hasQuery)
			}
		}
		if !hasQuery {
			return "", false
		}
		return query.String(), true
	})
}

// RewriteFragment applies rewriters from left to right to the serialized
// fragment, excluding '#'. Nil rewriters are ignored. The boolean distinguishes
// an absent fragment from a present-but-empty fragment.
//
// Rewriters must return valid serialized fragments. Returned values are stored
// without another parsing or encoding pass. This API is an EXPERIMENTAL
// extension and is not part of the WHATWG URL API.
func (u *Url) RewriteFragment(rewriters ...OptionalComponentRewriter) {
	if u == nil {
		return
	}

	fragment, hasFragment := u.fragment, u.hasFragment
	for _, rewrite := range rewriters {
		if rewrite != nil {
			fragment, hasFragment = rewrite(u, fragment, hasFragment)
		}
	}
	if !hasFragment {
		fragment = ""
		if !u.hasQuery {
			u.path.stripTrailingSpacesIfOpaque()
		}
	}
	if fragment != u.fragment || hasFragment != u.hasFragment {
		u.fragment, u.hasFragment = fragment, hasFragment
	}
}

func (u *Url) refreshIPMetadata() {
	u.isIPv4 = u.IsSpecialScheme() && serializedIPv4(u.host)
	u.isIPv6 = false
	if len(u.host) >= 2 && u.host[0] == '[' && u.host[len(u.host)-1] == ']' {
		_, err := parseIPv6([]byte(u.host[1:len(u.host)-1]), nil)
		u.isIPv6 = err == nil
	}
}

func serializedIPv4(host string) bool {
	parts := 0
	value := 0
	digits := 0
	for i := 0; i < len(host); i++ {
		switch {
		case host[i] >= '0' && host[i] <= '9':
			value = value*10 + int(host[i]-'0')
			if value > 255 {
				return false
			}
			digits++
		case host[i] == '.' && digits > 0 && parts < 3:
			parts++
			value = 0
			digits = 0
		default:
			return false
		}
	}
	return parts == 3 && digits > 0
}
