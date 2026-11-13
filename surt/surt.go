// Package surt creates [Sort-friendly URI Reordering Transform] (SURT) keys
// and supports the [IIPC SSURT format].
//
// SURT keys group URLs by reversed hostname for sorting and prefix matching.
// This package is experimental.
//
// [Sort-friendly URI Reordering Transform]: http://crawler.archive.org/articles/user_manual/glossary.html#surt
// [IIPC SSURT format]: https://github.com/iipc/urlcanon/blob/master/ssurt.rst
package surt

import (
	"fmt"
	"strings"

	whatwgurl "github.com/nationallibraryofnorway/whatwg-url/url"
)

// Options controls SURT formatting and the parser used by String.
type Options struct {
	TrailingComma bool
	ReverseIPAddr bool
	IncludeScheme bool
	Parser        whatwgurl.Parser
}

// Option changes SURT options.
type Option func(*Options)

func defaultOptions() Options {
	return Options{ReverseIPAddr: true, Parser: defaultSURTParser}
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

// WithParser selects the parser used by String. Profiles implementing
// url.Parser can supply any desired canonicalization. Nil restores the default
// IA parser. This option has no effect on URL, which formats an existing URL.
func WithParser(parser whatwgurl.Parser) Option {
	return func(options *Options) { options.Parser = parser }
}

// URL formats an already-parsed URL as a SURT. It does not canonicalize, clone,
// or mutate u. Canonicalize explicitly first when normalized keys are needed.
func URL(u *whatwgurl.Url, opts ...Option) (string, error) {
	return formatURL(u, applyOptions(opts)), nil
}

func formatURL(u *whatwgurl.Url, options Options) string {
	if u == nil {
		return ""
	}
	if u.Hostname() == "" || shouldPassthroughScheme(u.Scheme()) {
		return u.String()
	}
	return serializeSURT(u, options)
}

// String parses rawURL once with the selected parser, then formats it as a
// SURT. The default parser applies IA canonicalization. Empty input produces
// "-"; passthrough schemes retain their original input without parsing.
func String(rawURL string, opts ...Option) (string, error) {
	if rawURL == "" {
		return "-", nil
	}
	if scheme, ok := rawScheme(rawURL); ok && shouldPassthroughScheme(scheme) {
		return rawURL, nil
	}

	options := applyOptions(opts)
	parser := options.Parser
	if parser == nil {
		parser = defaultSURTParser
	}
	u, err := parser.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return formatURL(u, options), nil
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
// Keys without a scheme use "http". Canonicalization is lossy, so reversal
// cannot recover spelling or components removed when the key was created.
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
