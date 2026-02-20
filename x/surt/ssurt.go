package surt

import (
	"fmt"
	"net"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	whatwgurl "github.com/nationallibraryofnorway/whatwg-url/url"
)

var ssurtCanonicalizer = canonicalizer.New(
	whatwgurl.WithSkipEqualsForEmptySearchParamsValue(),
	whatwgurl.WithLaxHostParsing(),
)

func SSURTURL(u *whatwgurl.Url, includeScheme bool) (string, error) {
	if u == nil {
		return "", nil
	}

	var result strings.Builder
	hostname := u.Hostname()
	if hostname != "" {
		if hostname[0] == '[' {
			result.WriteString(hostname)
		} else if net.ParseIP(hostname).To4() != nil {
			result.WriteString(hostname)
		} else {
			splitOnDot := strings.Split(hostname, ".")
			for partIndex := len(splitOnDot) - 1; partIndex >= 0; partIndex-- {
				result.WriteString(splitOnDot[partIndex])
				result.WriteByte(',')
			}
		}
		result.WriteString("//")
	}

	if includeScheme {
		if u.Port() != "" {
			result.WriteString(u.Port())
		}
		result.WriteByte(':')
		result.WriteString(strings.TrimSuffix(u.Protocol(), ":"))
		if u.Username() != "" {
			result.WriteByte('@')
			result.WriteString(u.Username())
		}
		if u.Password() != "" {
			result.WriteByte(':')
			result.WriteString(u.Password())
		}
		result.WriteByte(':')
	}

	result.WriteString(u.Pathname())
	result.WriteString(u.Search())
	result.WriteString(u.Hash())

	return result.String(), nil
}

func SSURTString(rawURL string, includeScheme bool) (string, error) {
	u, err := ssurtCanonicalizer.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return SSURTURL(u, includeScheme)
}

// UnSSURT converts an SSURT string back to a URL string.
func UnSSURT(ssurtValue string) (string, error) {
	if ssurtValue == "" {
		return "", nil
	}

	if strings.Contains(ssurtValue, ":") && !strings.Contains(ssurtValue, "//") {
		return ssurtValue, nil
	}

	delimIdx := strings.Index(ssurtValue, "//")
	if delimIdx < 0 {
		return "", fmt.Errorf("invalid SSURT: missing '//' delimiter")
	}

	hostPart := ssurtValue[:delimIdx]
	remainder := ssurtValue[delimIdx+2:]

	host, err := hostFromSURTKey(hostPart)
	if err != nil {
		return "", err
	}

	port := ""
	scheme := "http"
	userinfo := ""
	path := "/"

	head := remainder
	if slashIdx := strings.IndexByte(remainder, '/'); slashIdx >= 0 {
		head = remainder[:slashIdx]
		path = remainder[slashIdx:]
	}

	if head != "" {
		if !strings.HasSuffix(head, ":") {
			return "", fmt.Errorf("invalid SSURT: malformed port/scheme/userinfo segment")
		}
		core := head[:len(head)-1]
		before, after, ok := strings.Cut(core, ":")
		if !ok {
			return "", fmt.Errorf("invalid SSURT: malformed port/scheme separator")
		}

		port = before
		schemeAndUser := after
		if before, after, ok := strings.Cut(schemeAndUser, "@"); ok {
			scheme = strings.ToLower(before)
			userinfo = after
		} else if schemeAndUser != "" {
			scheme = strings.ToLower(schemeAndUser)
		}
	}

	authority := host
	if userinfo != "" {
		authority = userinfo + "@" + authority
	}
	if port != "" {
		authority = authority + ":" + port
	}

	return scheme + "://" + authority + path, nil
}
