// Package surt creates [Sort-friendly URI Reordering Transform] (SURT) keys
// and supports the [IIPC SSURT format].
//
// SURT is an Internet Archive convention, not a web standard. This package is
// experimental, including its reverse operations and SSURT API.
//
// [Sort-friendly URI Reordering Transform]: http://crawler.archive.org/articles/user_manual/glossary.html#surt
// [IIPC SSURT format]: https://github.com/iipc/urlcanon/blob/master/ssurt.rst
package surt

import (
	"fmt"
	"regexp"
	"strings"

	whatwgurl "github.com/nationallibraryofnorway/whatwg-url/url"
)

// Options controls SURT serialization and IA-style canonicalization.
type Options struct {
	TrailingComma bool
	ReverseIPAddr bool
	IncludeScheme bool
	HostMassage   bool
	QueryFilter   func(name, value string, hasEq bool) bool
}

// Option changes SURT options.
type Option func(*Options)

func defaultOptions() Options {
	return Options{
		ReverseIPAddr: true,
		HostMassage:   true,
		QueryFilter:   shouldFilterCanonicalParam,
	}
}

// WithTrailingComma controls whether a comma is appended to the SURT host key.
func WithTrailingComma(enabled bool) Option {
	return func(options *Options) { options.TrailingComma = enabled }
}

// WithReverseIPAddr controls reversal of four-decimal-label IP-like hosts.
func WithReverseIPAddr(enabled bool) Option {
	return func(options *Options) { options.ReverseIPAddr = enabled }
}

// WithIncludeScheme controls whether output starts with "scheme://(".
func WithIncludeScheme(enabled bool) Option {
	return func(options *Options) { options.IncludeScheme = enabled }
}

// WithHostMassage controls removal of leading www and wwwN labels.
func WithHostMassage(enabled bool) Option {
	return func(options *Options) { options.HostMassage = enabled }
}

// WithQueryParamFilter replaces the default IA session-parameter filter.
// Returning true removes a serialized parameter. A nil filter disables query
// filtering.
func WithQueryParamFilter(filter func(name, value string, hasEq bool) bool) Option {
	return func(options *Options) { options.QueryFilter = filter }
}

// WithFilteredQueryParams replaces the default filter with a case-insensitive
// filter that removes the named parameters.
func WithFilteredQueryParams(params ...string) Option {
	lookup := make(map[string]struct{}, len(params))
	for _, parameter := range params {
		lookup[strings.ToLower(parameter)] = struct{}{}
	}
	return WithQueryParamFilter(func(name, _ string, _ bool) bool {
		_, found := lookup[strings.ToLower(name)]
		return found
	})
}

var iaParser = whatwgurl.NewParser(
	whatwgurl.WithLaxHostParsing(),
	whatwgurl.WithCollapseConsecutiveSlashes(),
	whatwgurl.WithAcceptInvalidCodepoints(),
	whatwgurl.WithPercentEncodeSinglePercentSign(),
)

var iaEscapeSet = whatwgurl.NewPercentEncodeSet(0x20, ' ', '#', '%')
var iaQueryPartEscapeSet = iaEscapeSet.Set('&')

// URL converts a parsed URL to an IA-style SURT. It clones u and never mutates
// the caller's URL.
func URL(u *whatwgurl.Url, opts ...Option) (string, error) {
	if u == nil {
		return "", nil
	}
	u = u.Clone()

	if shouldPassthroughScheme(u.Scheme()) {
		return u.String(), nil
	}

	options := applyOptions(opts)
	canonicalize(u, options)
	return serializeSURT(u, options), nil
}

// String parses rawURL and converts it to an IA-style SURT.
func String(rawURL string, opts ...Option) (string, error) {
	if rawURL == "" {
		return "-", nil
	}
	if scheme, ok := rawScheme(rawURL); ok && shouldPassthroughScheme(scheme) {
		return rawURL, nil
	}

	u, err := iaParser.Parse(rawURL)
	if err != nil {
		fallback := strings.TrimSpace(rawURL)
		if _, hasScheme := rawScheme(fallback); hasScheme {
			return "", err
		}
		u, err = iaParser.Parse("http://" + fallback)
	}
	if err != nil {
		return "", err
	}
	return URL(u, opts...)
}

// Surt is an alias for String retained for compatibility.
func Surt(rawURL string, opts ...Option) (string, error) {
	return String(rawURL, opts...)
}

func applyOptions(opts []Option) Options {
	options := defaultOptions()
	for _, option := range opts {
		if option != nil {
			option(&options)
		}
	}
	return options
}

func canonicalize(u *whatwgurl.Url, options Options) {
	u.SetUsername("")
	u.SetPassword("")
	u.RewriteFragment(func(_ *whatwgurl.Url, _ string, _ bool) (string, bool) {
		return "", false
	})

	u.RewriteHostname(func(_ *whatwgurl.Url, host string) string {
		if strings.HasPrefix(host, "[") {
			return strings.ToLower(host)
		}
		host = normalizeEscapes(host, iaEscapeSet)
		host = strings.ToLower(host)
		if options.HostMassage {
			host = massageHost(host)
		}
		return host
	})

	u.RewritePathname(func(_ *whatwgurl.Url, path string) string {
		path = strings.ToLower(normalizeEscapes(path, iaEscapeSet))
		path = stripPathSessionID(path)
		if len(path) > 1 {
			path = strings.TrimSuffix(path, "/")
		}
		return path
	})

	u.RewriteQueryString(func(_ *whatwgurl.Url, raw string, present bool) (string, bool) {
		if !present || raw == "" {
			return "", false
		}
		query := whatwgurl.ParseQuery(raw)
		for i := range query {
			query[i].Name = strings.ToLower(normalizeQueryPart(query[i].Name))
			query[i].Value = strings.ToLower(normalizeQueryPart(query[i].Value))
		}
		if options.QueryFilter != nil {
			query.Filter(func(name, value string, hasEquals bool) bool {
				return !options.QueryFilter(name, value, hasEquals)
			})
		}
		if len(query) == 0 {
			return "", false
		}
		query.SortAbsolute()
		return query.String(), true
	})
}

func normalizeEscapes(value string, set *whatwgurl.PercentEncodeSet) string {
	return whatwgurl.PercentEncode(whatwgurl.PercentDecodeRepeated(value), set)
}

func normalizeQueryPart(value string) string {
	return normalizeEscapes(value, iaQueryPartEscapeSet)
}

func serializeSURT(u *whatwgurl.Url, options Options) string {
	var result strings.Builder
	if options.IncludeScheme {
		result.WriteString(u.Protocol())
		result.WriteString("//(")
	}

	hostname := u.Hostname()
	if hostname != "" {
		result.WriteString(hostToSURTKey(hostname, options.ReverseIPAddr))
		if u.Port() != "" {
			result.WriteByte(':')
			result.WriteString(u.Port())
		}
		if options.TrailingComma {
			result.WriteByte(',')
		}
		result.WriteByte(')')
	}

	result.WriteString(u.Pathname())
	if u.Query() != "" {
		result.WriteByte('?')
		result.WriteString(u.Query())
	}
	return result.String()
}

func hostToSURTKey(hostname string, reverseIPAddr bool) string {
	if hostname == "" || strings.HasPrefix(hostname, "[") {
		return hostname
	}
	if isFourDecimalLabelIP(hostname) && !reverseIPAddr {
		return hostname
	}
	labels := strings.Split(strings.ToLower(hostname), ".")
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return strings.Join(labels, ",")
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

func shouldFilterCanonicalParam(name, value string, hasEquals bool) bool {
	lowerName := strings.ToLower(name)
	if hasEquals && alphaNum32Regex.MatchString(value) {
		switch lowerName {
		case "jsessionid", "phpsessid", "sid":
			return true
		}
	}
	if hasEquals && aspsessionIDRegex.MatchString(name) && len(value) == 24 {
		return true
	}
	return lowerName == "cfid" || lowerName == "cftoken"
}

func shouldPassthroughScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "filedesc", "warcinfo", "dns", "mailto":
		return true
	default:
		return false
	}
}

func rawScheme(raw string) (string, bool) {
	for i := 0; i < len(raw); i++ {
		character := raw[i]
		switch {
		case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z':
		case i > 0 && (character >= '0' && character <= '9' || character == '+' || character == '-' || character == '.'):
		case i > 0 && character == ':':
			return raw[:i], true
		default:
			return "", false
		}
	}
	return "", false
}

// UnSURT converts a SURT key back to a normalized URL string.
//
// This is an EXPERIMENTAL, non-IA extension. Canonicalization is lossy, so a
// reverse result is not necessarily byte-for-byte equivalent to the input URL.
func UnSURT(value string) (string, error) {
	if value == "" || value == "-" {
		return "", nil
	}
	if scheme, ok := rawScheme(value); ok && shouldPassthroughScheme(scheme) && !strings.Contains(value, ")") {
		return value, nil
	}

	hostPart, remainder, found := strings.Cut(value, ")")
	if !found {
		return "", fmt.Errorf("invalid SURT: missing ')' delimiter")
	}

	scheme := "http"
	if marker := strings.Index(hostPart, "://("); marker >= 0 {
		if marker == 0 {
			return "", fmt.Errorf("invalid SURT: empty scheme")
		}
		scheme = strings.ToLower(hostPart[:marker])
		if parsed, ok := rawScheme(scheme + ":"); !ok || parsed != scheme {
			return "", fmt.Errorf("invalid SURT: malformed scheme")
		}
		hostPart = hostPart[marker+4:]
	} else if strings.Contains(hostPart, "(") || strings.Contains(hostPart, "://") {
		return "", fmt.Errorf("invalid SURT: malformed scheme prefix")
	}

	hostPart = strings.TrimSuffix(hostPart, ",")
	hostKey, port, err := splitSURTHostPort(hostPart)
	if err != nil {
		return "", err
	}
	host, err := hostFromSURTKey(hostKey)
	if err != nil {
		return "", err
	}
	if port != "" {
		host += ":" + port
	}
	if remainder == "" {
		remainder = "/"
	} else if remainder[0] != '/' {
		remainder = "/" + remainder
	}
	return scheme + "://" + host + remainder, nil
}

func splitSURTHostPort(hostPart string) (host, port string, err error) {
	if strings.HasPrefix(hostPart, "[") {
		end := strings.IndexByte(hostPart, ']')
		if end < 0 {
			return "", "", fmt.Errorf("invalid SURT: malformed IPv6 host")
		}
		if end+1 == len(hostPart) {
			return hostPart, "", nil
		}
		if hostPart[end+1] != ':' || end+2 == len(hostPart) {
			return "", "", fmt.Errorf("invalid SURT: malformed port")
		}
		port = hostPart[end+2:]
		if !validPort(port) {
			return "", "", fmt.Errorf("invalid SURT: malformed port")
		}
		return hostPart[:end+1], port, nil
	}
	if index := strings.LastIndexByte(hostPart, ':'); index >= 0 {
		if index == 0 || index+1 == len(hostPart) {
			return "", "", fmt.Errorf("invalid SURT: malformed port")
		}
		port = hostPart[index+1:]
		if !validPort(port) {
			return "", "", fmt.Errorf("invalid SURT: malformed port")
		}
		return hostPart[:index], port, nil
	}
	return hostPart, "", nil
}

func validPort(port string) bool {
	if port == "" || len(port) > 5 {
		return false
	}
	value := 0
	for i := 0; i < len(port); i++ {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
		value = value*10 + int(port[i]-'0')
	}
	return value <= 65535
}

func hostFromSURTKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("invalid SURT: empty host key")
	}
	if strings.HasPrefix(key, "[") {
		if !strings.HasSuffix(key, "]") {
			return "", fmt.Errorf("invalid SURT: malformed IPv6 host")
		}
		return key, nil
	}
	if !strings.Contains(key, ",") {
		return key, nil
	}
	labels := strings.Split(key, ",")
	for _, label := range labels {
		if label == "" {
			return "", fmt.Errorf("invalid SURT: empty host label")
		}
	}
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return strings.Join(labels, "."), nil
}
