package surt

import (
	"fmt"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// SSURTURL converts a parsed URL to an SSURT string.
//
// With includeScheme, the key includes the port, scheme, and credentials.
// The URL is formatted without further canonicalization.
func SSURTURL(u *url.Url, includeScheme bool) (string, error) {
	if u == nil {
		return "", nil
	}
	if u.Hostname() == "" {
		return u.String(), nil
	}

	var result strings.Builder
	hostname := u.Hostname()
	result.WriteString(ssurtHost(hostname))
	result.WriteString("//")

	if includeScheme {
		if u.Port() != "" {
			result.WriteString(u.Port())
			result.WriteByte(':')
		}
		result.WriteString(u.Scheme())
		if u.Username() != "" || u.Password() != "" {
			result.WriteByte('@')
			result.WriteString(u.Username())
			if u.Password() != "" {
				result.WriteByte(':')
				result.WriteString(u.Password())
			}
		}
		result.WriteByte(':')
	}

	result.WriteString(u.Pathname())
	if u.Query() != "" {
		result.WriteByte('?')
		result.WriteString(u.Query())
	}
	if u.Fragment() != "" {
		result.WriteByte('#')
		result.WriteString(u.Fragment())
	}
	return result.String(), nil
}

// ssurtHost reverses domain labels while retaining IP address spelling.
func ssurtHost(host string) string {
	if host == "" || strings.HasPrefix(host, "[") || isURLCanonIPv4(host) {
		return host
	}
	return reverseHost(strings.ToLower(host))
}

// reverseHost reverses dotted labels, changes literal commas to dots, and
// appends the comma required by SSURT domain serialization.
func reverseHost(host string) string {
	normalized := strings.ReplaceAll(host, ",", ".")
	var result strings.Builder
	result.Grow(len(host) + 1)
	end := len(host)
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == '.' {
			result.WriteString(normalized[i+1 : end])
			result.WriteByte(',')
			end = i
		}
	}
	result.WriteString(normalized[:end])
	result.WriteByte(',')
	return result.String()
}

func isURLCanonIPv4(host string) bool {
	parsed, err := url.Parse("http://" + host)
	return err == nil && parsed.IsIPv4()
}

// SSURTString parses rawURL and converts it to SSURT.
//
// Parsing uses the IA parser without canonicalization rules, retaining path
// case, credentials, query order, and fragments. To apply a profile explicitly,
// pass its parsed result to SSURTURL.
func SSURTString(rawURL string, includeScheme bool) (string, error) {
	u, err := defaultSSURTParser.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return SSURTURL(u, includeScheme)
}

// UnSSURT converts an SSURT string back to a URL string.
//
// Keys without scheme metadata use "http". Reversal cannot recover spelling
// or components removed by canonicalization.
func UnSSURT(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if scheme, ok := rawScheme(value); ok && shouldPassthroughScheme(scheme) && !strings.Contains(value, "//") {
		return value, nil
	}

	hostKey, remainder, found := strings.Cut(value, "//")
	if !found {
		return "", fmt.Errorf("invalid SSURT: missing '//' delimiter")
	}
	host, err := hostFromSURTKey(strings.TrimSuffix(hostKey, ","))
	if err != nil {
		return "", err
	}

	head, path := remainder, "/"
	if index := strings.IndexAny(remainder, "/?#"); index >= 0 {
		head, path = remainder[:index], remainder[index:]
		if path[0] != '/' {
			path = "/" + path
		}
	}

	scheme, port, userinfo := "http", "", ""
	if head != "" {
		if !strings.HasSuffix(head, ":") {
			return "", fmt.Errorf("invalid SSURT: malformed metadata segment")
		}
		core := strings.TrimSuffix(head, ":")
		if possiblePort, schemeAndUser, hasPortSeparator := strings.Cut(core, ":"); hasPortSeparator {
			// Accept the earlier //:scheme: spelling when reading, while emitting
			// the IIPC //scheme: spelling for a missing port.
			if possiblePort != "" {
				if !validPort(possiblePort) {
					return "", fmt.Errorf("invalid SSURT: malformed port")
				}
				port = possiblePort
			}
			core = schemeAndUser
		}
		if before, after, hasUser := strings.Cut(core, "@"); hasUser {
			scheme, userinfo = strings.ToLower(before), after
		} else if core != "" {
			scheme = strings.ToLower(core)
		}
		if parsed, ok := rawScheme(scheme + ":"); !ok || parsed != scheme {
			return "", fmt.Errorf("invalid SSURT: malformed scheme")
		}
	}

	authority := host
	if userinfo != "" {
		authority = userinfo + "@" + authority
	}
	if port != "" {
		authority += ":" + port
	}
	return scheme + "://" + authority + path, nil
}
