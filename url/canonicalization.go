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
type OptionalComponentRewriter func(*Url, string, bool) (string, bool)

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
	u.host = host
	u.refreshIPMetadata()
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
	u.path.setSerialized(pathname)
}

// RewriteQuery applies rewriters from left to right to the serialized query,
// excluding '?'. Nil rewriters are ignored. The boolean distinguishes an absent
// query from a present-but-empty query.
//
// Rewriters must return valid serialized queries. Returned values are stored
// without another parsing or encoding pass. If SearchParams has already been
// obtained, its contents are refreshed without replacing the object. This API
// is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
func (u *Url) RewriteQuery(rewriters ...OptionalComponentRewriter) {
	if u == nil {
		return
	}

	query, present := u.query, u.hasQuery
	for _, rewrite := range rewriters {
		if rewrite != nil {
			query, present = rewrite(u, query, present)
		}
	}
	u.query, u.hasQuery = query, present
	if !present {
		u.query = ""
		if !u.hasFragment {
			u.path.stripTrailingSpacesIfOpaque()
		}
	}
	u.refreshSearchParams()
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

	fragment, present := u.fragment, u.hasFragment
	for _, rewrite := range rewriters {
		if rewrite != nil {
			fragment, present = rewrite(u, fragment, present)
		}
	}
	u.fragment, u.hasFragment = fragment, present
	if !present {
		u.fragment = ""
		if !u.hasQuery {
			u.path.stripTrailingSpacesIfOpaque()
		}
	}
}

func (u *Url) refreshIPMetadata() {
	u.isIPv4 = serializedIPv4(u.host)
	u.isIPv6 = len(u.host) >= 2 && u.host[0] == '[' && u.host[len(u.host)-1] == ']'
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
