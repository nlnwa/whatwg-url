// Package canonicalizer builds URL canonicalization profiles from ordered
// functions. Profiles use the url Rewrite API and are experimental extensions
// to WHATWG parsing. Canonicalization mutates a URL in place; clone it first
// when the original must be retained.
package canonicalizer

import (
	"errors"
	"slices"
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// Rule changes a parsed URL in place. General URL rules leave valid serialized
// components, following the contract of url's Rewrite methods. Matching profiles
// such as GoogleSafeBrowsing intentionally emit minimally escaped component
// bytes; their documentation describes the resulting serialization limits.
type Rule func(*url.Url)

// Profile combines parsing with an ordered pipeline of rules. Its zero value
// performs standard WHATWG parsing without additional canonicalization.
// Configuration methods return new profiles; they do not modify the receiver.
// Profiles may be shared by goroutines if their parser and rules are safe to share.
type Profile struct {
	parser        url.Parser
	rules         []Rule
	defaultScheme string
}

var standardParser = url.NewParser()

var _ url.Parser = Profile{}

// New builds a profile that applies rules in declaration order, ignoring nil
// rules. The supplied slice is copied.
func New(rules ...Rule) Profile {
	return Profile{rules: slices.Clone(rules)}
}

// Then returns a profile with additional rules appended to its pipeline.
func (p Profile) Then(rules ...Rule) Profile {
	p.rules = append(slices.Clone(p.rules), rules...)
	return p
}

// WithParser returns a profile using parser. A nil parser selects standard
// WHATWG parsing. Parser settings apply before any canonicalization rules.
func (p Profile) WithParser(parser url.Parser) Profile {
	p.parser = parser
	return p
}

// WithDefaultScheme returns a profile that retries missing-scheme inputs with
// scheme. An empty scheme disables fallback. Other parse failures are returned
// unchanged. The scheme is supplied without a trailing colon.
func (p Profile) WithDefaultScheme(scheme string) Profile {
	p.defaultScheme = scheme
	return p
}

// Parser returns the underlying parser without the profile's rules or
// default-scheme fallback. Use this when canonicalization will happen later.
func (p Profile) Parser() url.Parser {
	if p.parser == nil {
		return standardParser
	}
	return p.parser
}

// Parse parses raw and applies the pipeline to the resulting URL.
func (p Profile) Parse(raw string) (*url.Url, error) {
	parser := p.Parser()
	u, err := parser.Parse(raw)
	if p.defaultScheme != "" && errors.Is(err, url.ErrMissingSchemeNonRelativeURL) {
		u, err = parser.Parse(p.addScheme(raw))
	}
	if err != nil {
		return nil, err
	}
	p.Canonicalize(u)
	return u, nil
}

// ParseRef resolves ref against the unmodified base and canonicalizes only the
// result. Default-scheme fallback applies to base, never to ref.
func (p Profile) ParseRef(base, ref string) (*url.Url, error) {
	parser := p.Parser()
	u, err := parser.ParseRef(base, ref)
	if p.defaultScheme != "" && errors.Is(err, url.ErrMissingSchemeNonRelativeURL) {
		// A missing-scheme error can also come from resolving against an opaque
		// base. Retry only when the base itself is missing its scheme.
		if _, baseErr := parser.Parse(base); errors.Is(baseErr, url.ErrMissingSchemeNonRelativeURL) {
			u, err = parser.ParseRef(p.addScheme(base), ref)
		}
	}
	if err != nil {
		return nil, err
	}
	p.Canonicalize(u)
	return u, nil
}

func (p Profile) addScheme(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		return p.defaultScheme + ":" + raw
	}
	return p.defaultScheme + "://" + raw
}

// Canonicalize applies each non-nil rule once, in order, to u in place. A nil
// URL is ignored. Clone the URL before calling to retain its original value.
func (p Profile) Canonicalize(u *url.Url) {
	if u == nil {
		return
	}
	for _, rule := range p.rules {
		if rule != nil {
			rule(u)
		}
	}
}
