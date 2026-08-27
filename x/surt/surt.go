package surt

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	whatwgurl "github.com/nationallibraryofnorway/whatwg-url/url"
	"golang.org/x/net/publicsuffix"
)

type Options struct {
	TrailingComma bool
	ReverseIPAddr bool
	IncludeScheme bool
	HostMassage   bool
	QueryFilter   func(name, value string, hasEq bool) bool
}

type Option func(*Options)

func defaultOptions() Options {
	return Options{
		TrailingComma: false,
		ReverseIPAddr: true,
		IncludeScheme: false,
		HostMassage:   true,
		QueryFilter:   shouldFilterCanonicalParam,
	}
}

func WithTrailingComma(enabled bool) Option {
	return func(o *Options) {
		o.TrailingComma = enabled
	}
}

func WithReverseIPAddr(enabled bool) Option {
	return func(o *Options) {
		o.ReverseIPAddr = enabled
	}
}

func WithIncludeScheme(enabled bool) Option {
	return func(o *Options) {
		o.IncludeScheme = enabled
	}
}

func WithHostMassage(enabled bool) Option {
	return func(o *Options) {
		o.HostMassage = enabled
	}
}

func WithQueryParamFilter(filter func(name, value string, hasEq bool) bool) Option {
	return func(o *Options) {
		o.QueryFilter = filter
	}
}

func WithFilteredQueryParams(params ...string) Option {
	lookup := make(map[string]struct{}, len(params))
	for _, p := range params {
		lookup[strings.ToLower(p)] = struct{}{}
	}

	return func(o *Options) {
		o.QueryFilter = func(name, _ string, _ bool) bool {
			_, ok := lookup[strings.ToLower(name)]
			return ok
		}
	}
}

func newSURTCanonicalizer(cfg Options) whatwgurl.Parser {
	opts := []whatwgurl.ParserOption{
		whatwgurl.WithSkipEqualsForEmptySearchParamsValue(),
		whatwgurl.WithLaxHostParsing(),
		canonicalizer.WithSortQuery(canonicalizer.SortParameter),
		canonicalizer.WithRemoveFragment(),
		canonicalizer.WithPathTransformer(func(_ *whatwgurl.Url, path string) string {
			if len(path) > 1 && strings.HasSuffix(path, "/") {
				return strings.TrimSuffix(path, "/")
			}
			return path
		}),
	}

	if cfg.QueryFilter != nil {
		filter := cfg.QueryFilter
		opts = append(opts, canonicalizer.WithQueryTransformer(func(_ *whatwgurl.Url, query string) string {
			return filterQueryParameters(query, filter)
		}))
	}

	if cfg.HostMassage {
		opts = append(opts, canonicalizer.WithHostTransformer(func(_ *whatwgurl.Url, host string) string {
			return massageHost(host)
		}))
	}

	return canonicalizer.New(opts...)
}

// URL converts a parsed URL to IA-style SURT.
func URL(u *whatwgurl.Url, opts ...Option) (string, error) {
	if u == nil {
		return "", nil
	}

	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	if shouldPassthroughScheme(u.Scheme()) {
		return u.String(), nil
	}

	canon := newSURTCanonicalizer(cfg)

	normalized, err := canon.Parse(u.String())
	if err != nil {
		return "", err
	}
	u = normalized

	var result strings.Builder
	if cfg.IncludeScheme {
		result.WriteString(u.Protocol())
		result.WriteString("//")
	}

	hostname := u.Hostname()
	if hostname != "" {
		result.WriteString(hostToSURTKey(hostname, cfg.TrailingComma, cfg.ReverseIPAddr))
		result.WriteByte(')')
	}

	result.WriteString(u.Pathname())
	result.WriteString(u.Search())
	return result.String(), nil
}

// String parses a URL string and converts it to IA-style SURT.
func String(rawURL string, opts ...Option) (string, error) {
	if rawURL == "" {
		return "-", nil
	}

	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	canon := newSURTCanonicalizer(cfg)

	u, err := canon.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return URL(u, opts...)
}

func Surt(rawURL string, opts ...Option) (string, error) {
	return String(rawURL, opts...)
}

// UnSURT converts a SURT string back to a URL string.
//
// For SURTs produced by String, this returns a normalized URL representation.
// Since String applies canonicalization (sorting query params, removing fragment, etc.),
// round-trip equivalence is semantic, not byte-for-byte.
func UnSURT(surtValue string) (string, error) {
	if surtValue == "" || surtValue == "-" {
		return "", nil
	}

	if strings.Contains(surtValue, ":") && !strings.Contains(surtValue, ")") {
		return surtValue, nil
	}

	before, after, ok := strings.Cut(surtValue, ")")
	if !ok {
		return "", fmt.Errorf("invalid SURT: missing ')' delimiter")
	}

	hostPart := before
	pathPart := after

	scheme := "http"
	if i := strings.Index(hostPart, "://"); i >= 0 {
		scheme = strings.ToLower(hostPart[:i])
		hostPart = hostPart[i+3:]
	}

	host, err := hostFromSURTKey(hostPart)
	if err != nil {
		return "", err
	}

	if pathPart == "" {
		pathPart = "/"
	}
	if pathPart[0] != '/' && pathPart[0] != '?' && pathPart[0] != '#' {
		pathPart = "/" + pathPart
	}

	return scheme + "://" + host + pathPart, nil
}

func hostToSURTKey(hostname string, trailingComma bool, reverseIPAddr bool) string {
	if hostname == "" {
		return ""
	}

	if hostname[0] == '[' {
		if trailingComma {
			return hostname + ","
		}
		return hostname
	}

	if isIPv4Like(hostname) {
		if reverseIPAddr {
			octets := strings.Split(hostname, ".")
			for i, j := 0, len(octets)-1; i < j; i, j = i+1, j-1 {
				octets[i], octets[j] = octets[j], octets[i]
			}
			hostname = strings.Join(octets, ",")
		}
		if trailingComma {
			return hostname + ","
		}
		return hostname
	}

	hostname = strings.ToLower(hostname)

	if etld1, err := publicsuffix.EffectiveTLDPlusOne(hostname); err == nil {
		suffix, _ := publicsuffix.PublicSuffix(hostname)

		parts := make([]string, 0, 8)

		suffixParts := strings.Split(suffix, ".")
		for i := len(suffixParts) - 1; i >= 0; i-- {
			parts = append(parts, suffixParts[i])
		}

		etld1Parts := strings.Split(etld1, ".")
		if len(etld1Parts) > len(suffixParts) {
			parts = append(parts, etld1Parts[0])
		}

		if strings.HasSuffix(hostname, "."+etld1) {
			subdomain := strings.TrimSuffix(hostname, "."+etld1)
			if subdomain != "" {
				subParts := strings.Split(subdomain, ".")
				for i := len(subParts) - 1; i >= 0; i-- {
					parts = append(parts, subParts[i])
				}
			}
		}

		result := strings.Join(parts, ",")
		if trailingComma {
			return result + ","
		}
		return result
	}

	parts := strings.Split(hostname, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	result := strings.Join(parts, ",")
	if trailingComma {
		return result + ","
	}
	return result
}

func isIPv4Like(hostname string) bool {
	if ip := net.ParseIP(hostname); ip != nil {
		return ip.To4() != nil
	}

	parts := strings.Split(hostname, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

var wwwPrefixRegex = regexp.MustCompile(`^www\d*$`)
var aspsessionIDRegex = regexp.MustCompile(`(?i)^aspsessionid[a-z0-9]{8}$`)
var alphaNum32Regex = regexp.MustCompile(`(?i)^[a-z0-9]{32}$`)

func filterQueryParameters(query string, shouldFilter func(name, value string, hasEq bool) bool) string {
	if query == "" || query == "?" {
		return query
	}

	raw := strings.TrimPrefix(query, "?")
	if raw == "" {
		return query
	}

	segments := strings.Split(raw, "&")
	kept := make([]string, 0, len(segments))
	lastKeptIdx := -1

	for idx, segment := range segments {
		if segment == "" {
			continue
		}
		name, value, hasEq := strings.Cut(segment, "=")
		if shouldFilter(name, value, hasEq) {
			continue
		}
		kept = append(kept, segment)
		lastKeptIdx = idx
	}

	if len(kept) == 0 {
		return "?"
	}

	out := "?" + strings.Join(kept, "&")
	if strings.HasSuffix(raw, "&") || lastKeptIdx < len(segments)-1 {
		out += "&"
	}

	return out
}

func shouldFilterCanonicalParam(name, value string, hasEq bool) bool {
	lowerName := strings.ToLower(name)

	if lowerName == "jsessionid" && hasEq && alphaNum32Regex.MatchString(value) {
		return true
	}
	if lowerName == "phpsessid" && hasEq && alphaNum32Regex.MatchString(value) {
		return true
	}
	if lowerName == "sid" && hasEq && alphaNum32Regex.MatchString(value) {
		return true
	}
	if aspsessionIDRegex.MatchString(name) && hasEq && len(value) == 24 {
		return true
	}
	if lowerName == "cfid" || lowerName == "cftoken" {
		return true
	}

	return false
}

func massageHost(hostname string) string {
	if hostname == "" || strings.HasPrefix(hostname, "[") || isIPv4Like(hostname) {
		return hostname
	}
	parts := strings.Split(hostname, ".")
	for len(parts) > 1 && wwwPrefixRegex.MatchString(strings.ToLower(parts[0])) {
		parts = parts[1:]
	}
	return strings.Join(parts, ".")
}

func shouldPassthroughScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "filedesc", "warcinfo", "dns", "mailto":
		return true
	default:
		return false
	}
}

func hostFromSURTKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("invalid host key: empty")
	}

	if strings.HasPrefix(key, "[") {
		return strings.TrimSuffix(key, ","), nil
	}

	if !strings.Contains(key, ",") {
		return key, nil
	}

	parts := strings.Split(strings.TrimSuffix(key, ","), ",")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid host key: empty labels")
	}

	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return strings.Join(parts, "."), nil
}
