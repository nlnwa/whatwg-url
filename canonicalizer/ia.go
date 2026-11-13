package canonicalizer

import (
	"regexp"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// IA returns the Internet Archive-style canonicalization profile used by SURT
// by default.
// Host massage and the six session query filters can be independently
// configured through factory options.
func IA(opts ...IAOption) Profile {
	options := iaOptions{hostMassage: true, queryFilter: combineQueryFilters(DefaultSessionQueryFilters())}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return New(func(u *url.Url) { canonicalizeIA(u, options) }).
		WithParser(url.NewParser(
			url.WithLaxHostParsing(),
			url.WithCollapseConsecutiveSlashes(),
			url.WithAcceptInvalidCodepoints(),
			url.WithPercentEncodeSinglePercentSign(),
		)).
		WithDefaultScheme("http")
}

// DefaultSessionQueryFilters returns a fresh collection of the six keep
// predicates used by IA. Each entry can be removed or replaced independently
// before passing the collection to WithQueryFilters. True keeps a term.
func DefaultSessionQueryFilters() map[string]QueryFilter {
	return map[string]QueryFilter{
		"jsessionid": keepUnlessSessionID("jsessionid"),
		"phpsessid":  keepUnlessSessionID("phpsessid"),
		"sid":        keepUnlessSessionID("sid"),
		"aspsessionid": func(name, value string, hasEquals bool) bool {
			return !hasEquals || !aspsessionIDRegex.MatchString(name) || len(value) != 24
		},
		"cfid":    func(name, _ string, _ bool) bool { return !strings.EqualFold(name, "cfid") },
		"cftoken": func(name, _ string, _ bool) bool { return !strings.EqualFold(name, "cftoken") },
	}
}

var iaEscapeSet = url.NewPercentEncodeSet(0x20, ' ', '#', '%')
var iaQueryPartEscapeSet = iaEscapeSet.Set('&')
var iaQueryNameEscapeSet = iaQueryPartEscapeSet.Set('=')

func canonicalizeIA(u *url.Url, options iaOptions) {
	if shouldPassthroughScheme(u.Scheme()) {
		return
	}
	u.SetUsername("")
	u.SetPassword("")
	u.RewriteFragment(func(_ *url.Url, _ string, _ bool) (string, bool) {
		return "", false
	})

	u.RewriteHostname(func(_ *url.Url, host string) string {
		if strings.HasPrefix(host, "[") {
			return strings.ToLower(host)
		}
		host = normalizeEscapes(host, iaEscapeSet)
		host = strings.ToLower(host)
		if options.hostMassage {
			host = massageHost(host)
		}
		return host
	})

	u.RewritePathname(func(_ *url.Url, path string) string {
		path = strings.ToLower(normalizeEscapes(path, iaEscapeSet))
		path = stripPathSessionID(path)
		if len(path) > 1 {
			path = strings.TrimSuffix(path, "/")
		}
		return path
	})

	rewritePresentQuery(u, func(query url.Query) (url.Query, bool) {
		if len(query) == 1 && query[0].Name == "" && !query[0].HasEquals {
			return nil, false
		}
		for i := range query {
			// An exposed '=' in a name would become a separator on the next
			// parse. Values can contain '=' without changing term boundaries.
			query[i].Name = strings.ToLower(normalizeEscapes(query[i].Name, iaQueryNameEscapeSet))
			query[i].Value = strings.ToLower(normalizeQueryPart(query[i].Value))
		}
		query.Filter(options.queryFilter)
		if len(query) == 0 {
			return query, false
		}
		query.SortAbsolute()
		return query, true
	})
}

func normalizeQueryPart(value string) string {
	return normalizeEscapes(value, iaQueryPartEscapeSet)
}

func isFourDecimalLabelIP(hostname string) bool {
	labels := strings.Split(hostname, ".")
	if len(labels) != 4 {
		return false
	}
	for _, label := range labels {
		if len(label) < 1 || len(label) > 3 {
			return false
		}
		for i := 0; i < len(label); i++ {
			if label[i] < '0' || label[i] > '9' {
				return false
			}
		}
	}
	return true
}

var wwwPrefixRegex = regexp.MustCompile(`(?i)^www[0-9]*$`)

func massageHost(hostname string) string {
	if hostname == "" || strings.HasPrefix(hostname, "[") || isFourDecimalLabelIP(hostname) {
		return hostname
	}
	labels := strings.Split(hostname, ".")
	if len(labels) > 1 && wwwPrefixRegex.MatchString(labels[0]) {
		labels = labels[1:]
	}
	return strings.Join(labels, ".")
}

var pathSessionRegex = regexp.MustCompile(`/\((?:[A-Za-z0-9]{24}|(?:[A-Za-z]\([A-Za-z0-9]{24}\))+?)\)/`)

func stripPathSessionID(path string) string {
	return pathSessionRegex.ReplaceAllString(path, "/")
}

var aspsessionIDRegex = regexp.MustCompile(`(?i)^aspsessionid[a-z0-9]{8}$`)
var alphaNum32Regex = regexp.MustCompile(`(?i)^[a-z0-9]{32}$`)

func keepUnlessSessionID(parameter string) QueryFilter {
	return func(name, value string, hasEquals bool) bool {
		return !strings.EqualFold(name, parameter) || !hasEquals || !alphaNum32Regex.MatchString(value)
	}
}

func shouldPassthroughScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "filedesc", "warcinfo", "dns", "mailto":
		return true
	default:
		return false
	}
}
