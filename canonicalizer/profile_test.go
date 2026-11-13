package canonicalizer_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	canon "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func checkProfile(t *testing.T, p canon.Profile, input, want string) *url.Url {
	t.Helper()
	u, err := p.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != want {
		t.Fatalf("Parse(%q) = %q, want %q", input, u, want)
	}
	return u
}

func TestProfiles(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		profile           canon.Profile
	}{
		{"WHATWG", "HTTP://WWW.Example.org:80/A?b=2&a=1#F", "http://www.example.org/A?b=2&a=1#F", canon.WhatWg()},
		{"sorted WHATWG", "http://example.org/?b=2&a=z&a=a&flag&q=a+b", "http://example.org/?a=z&a=a&b=2&flag&q=a+b", canon.WhatWgSortQuery()},
		{"semantic", "http://User:Pass@WWW.Example.org:80/A//B/%25FC?b=two+words&a=%2526&b=1#F", "http://www.example.org/A/B/%FC?a=%26&b=1&b=two+words", canon.Semantic()},
		{"semantic gopher", "gopher://Example.org:70/A?b=1&a=2", "gopher://example.org/A?a=2&b=1", canon.Semantic()},
		{"semantic lax path", "http://example.org/%253C%253E%255E", "http://example.org/<>^", canon.Semantic()},
		{"IA", "HTTP://User:Pass@WWW.Example.org:80/A/%25FC/?B=2&A=1#F", "http://example.org/a/%fc?a=1&b=2", canon.IA()},
		{"IA passthrough", "dns:WWW.Example.org?B=2&A=1#F", "dns:WWW.Example.org?B=2&A=1#F", canon.IA()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			checkProfile(t, tc.profile, tc.input, tc.want)
		})
	}
}

func TestPipelineRunsInOrderAndMutatesInput(t *testing.T) {
	u, err := url.Parse("https://user:pass@example.org:8443/original#frag")
	if err != nil {
		t.Fatal(err)
	}
	var stages []string
	p := canon.New(nil,
		func(got *url.Url) {
			if got != u {
				t.Fatal("pipeline replaced the input URL")
			}
			stages = append(stages, got.Pathname())
			got.RewritePathname(func(_ *url.Url, _ string) string { return "/first" })
		},
		func(got *url.Url) {
			stages = append(stages, got.Pathname())
			got.RewritePathname(func(_ *url.Url, path string) string { return path + "/second" })
		},
		canon.RemoveUserInfo, canon.RemovePort, canon.RemoveFragment,
	)
	p.Canonicalize(nil)
	p.Canonicalize(u)
	if !slices.Equal(stages, []string{"/original", "/first"}) || u.String() != "https://example.org/first/second" {
		t.Fatalf("stages = %q, URL = %q", stages, u)
	}
}

func TestProfileConfigurationIsIndependent(t *testing.T) {
	rules := []canon.Rule{canon.RemoveFragment}
	base := canon.New(rules...)
	rules[0] = canon.RemoveUserInfo
	credentials := base.Then(canon.RemoveUserInfo)
	port := base.Then(canon.RemovePort)
	const input = "https://user:pass@example.org:8443/path#frag"
	checkProfile(t, base, input, "https://user:pass@example.org:8443/path")
	checkProfile(t, credentials, input, "https://example.org:8443/path")
	checkProfile(t, port, input, "https://user:pass@example.org/path")
	checkProfile(t, canon.Profile{}, input, input)
	checkProfile(t, canon.New(), input, input)

	withScheme := base.WithDefaultScheme("https")
	checkProfile(t, withScheme, "example.org", "https://example.org/")
	if _, err := base.Parse("example.org"); !errors.Is(err, url.ErrMissingSchemeNonRelativeURL) {
		t.Fatalf("base profile acquired a default scheme: %v", err)
	}
	if _, err := withScheme.WithDefaultScheme("").Parse("example.org"); !errors.Is(err, url.ErrMissingSchemeNonRelativeURL) {
		t.Fatalf("empty default scheme did not disable fallback: %v", err)
	}

	lax := base.WithParser(url.NewParser(url.WithLaxHostParsing()))
	checkProfile(t, lax, "https://host%23.example/", "https://host%23.example/")
	for _, p := range []canon.Profile{base, lax.WithParser(nil)} {
		if _, err := p.Parse("https://host%23.example/"); err == nil {
			t.Fatal("standard profile acquired lax recovery")
		}
	}
}

func TestDefaultSchemeAndRelativeResolution(t *testing.T) {
	p := canon.New(canon.RewritePathname(func(_ *url.Url, path string) string {
		return strings.TrimSuffix(path, "/")
	})).WithDefaultScheme("https")
	for _, input := range []string{" example.org/path ", " //example.org/path "} {
		checkProfile(t, p, input, "https://example.org/path")
	}
	for _, base := range []string{"https://example.org/dir/", " example.org/dir/ ", " //example.org/dir/ "} {
		u, err := p.ParseRef(base, "child?b&a#f")
		if err != nil || u.String() != "https://example.org/dir/child?b&a#f" {
			t.Fatalf("ParseRef(%q) = %v, %v; base must resolve before rules", base, u, err)
		}
	}
	for _, input := range []string{"http://example.org:bad/", "http://1.2.3.999/", "http://[1::2::3]/"} {
		if _, err := p.Parse(input); err == nil {
			t.Errorf("invalid explicit URL was recovered: %q", input)
		}
		if _, err := p.ParseRef(input, "child"); err == nil {
			t.Errorf("invalid explicit base was recovered: %q", input)
		}
	}
	if _, err := p.ParseRef("mailto:user@example.org", "child"); !errors.Is(err, url.ErrMissingSchemeNonRelativeURL) {
		t.Fatalf("opaque base resolution = %v, want missing-scheme failure", err)
	}
}

type recordingParser struct {
	url.Parser
	parse, parseRef int
}

func (p *recordingParser) Parse(raw string) (*url.Url, error) {
	p.parse++
	return p.Parser.Parse(raw)
}

func (p *recordingParser) ParseRef(base, ref string) (*url.Url, error) {
	p.parseRef++
	return p.Parser.ParseRef(base, ref)
}

func TestParserInjectionAndParseOnlyAccess(t *testing.T) {
	parser := &recordingParser{Parser: url.NewParser()}
	p := canon.New(canon.RemoveFragment).WithParser(parser).WithDefaultScheme("https")
	if p.Parser() != parser {
		t.Fatal("Parser did not return injected parser")
	}
	checkProfile(t, p, "https://example.org/#f", "https://example.org/")
	u, err := p.ParseRef("https://example.org/", "child#f")
	if err != nil || u.String() != "https://example.org/child" || parser.parse != 1 || parser.parseRef != 1 {
		t.Fatalf("parser delegation = %v, %v; calls = %d, %d", u, err, parser.parse, parser.parseRef)
	}
	checkProfile(t, canon.New().WithParser(p.Parser()), "https://example.org/#f", "https://example.org/#f")
	if _, err := p.Parser().Parse("example.org"); !errors.Is(err, url.ErrMissingSchemeNonRelativeURL) {
		t.Fatalf("parse-only access applied default scheme: %v", err)
	}
}

func TestDiagnosticsSurviveCanonicalization(t *testing.T) {
	u := checkProfile(t, canon.GoogleSafeBrowsing(), "http://host%23.com/#f", "http://host%23.com/")
	var codes []url.ValidationErrorCode
	for _, occurrence := range u.ValidationErrors() {
		codes = append(codes, occurrence.Code())
	}
	if !slices.Equal(codes, []url.ValidationErrorCode{url.CodeDomainPercentEncoded, url.CodeDomainToASCII}) {
		t.Fatalf("recovery diagnostics = %v", codes)
	}
	strict := canon.GoogleSafeBrowsing().WithParser(url.NewParser(url.WithLaxHostParsing(), url.WithFailOnValidationError()))
	if _, err := strict.Parse("http://host%23.com/"); !errors.Is(err, url.ErrDomainPercentEncoded) {
		t.Fatalf("strict parsing did not take precedence: %v", err)
	}
}

func TestQuerySorting(t *testing.T) {
	const input = "https://example.org/?b=2&a=z&a=a&a&=v&&q=a+b&q=%20&bytes=%FC"
	checkProfile(t, canon.New(canon.SortQuery), input, "https://example.org/?=v&&a=z&a=a&a&b=2&bytes=%FC&q=a+b&q=%20")
	checkProfile(t, canon.New(canon.SortQueryAbsolute), input, "https://example.org/?&=v&a&a=a&a=z&b=2&bytes=%FC&q=%20&q=a+b")
}

func TestRemoveEmptyQueryEquals(t *testing.T) {
	p := canon.New(canon.RemoveEmptyQueryEquals)
	canon.RemoveEmptyQueryEquals(nil)
	for _, tc := range []struct{ name, input, want string }{
		{"absent", "https://example.org/", "https://example.org/"},
		{"empty", "https://example.org/?", "https://example.org/?"},
		{"flag", "https://example.org/?flag=", "https://example.org/?flag"},
		{"empty name and value", "https://example.org/?=", "https://example.org/?"},
		{"empty terms", "https://example.org/?&flag=&&=&", "https://example.org/?&flag&&&"},
		{"duplicates", "https://example.org/?a=z&a=&a=a&a&a=", "https://example.org/?a=z&a&a=a&a&a"},
		{"nonempty values", "https://example.org/?a=0&b=+&c=%20&d==&=value&flag", "https://example.org/?a=0&b=+&c=%20&d==&=value&flag"},
		{"encoded bytes", "https://example.org/?%66lag=&q=%2f%FC%26%3d%2B&bad=%GG%2&%3D=", "https://example.org/?%66lag&q=%2f%FC%26%3d%2B&bad=%GG%2&%3D"},
		{"other components", "https://user:pass@example.org:8443/flag=?flag=#flag=", "https://user:pass@example.org:8443/flag=?flag#flag="},
		{"opaque path", "custom:opaque?=#", "custom:opaque?#"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkProfile(t, p, tc.input, tc.want)
			u, err := url.Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			params := u.SearchParams()
			want, err := url.Parse(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			form := want.SearchParams().String()
			for range 2 {
				p.Canonicalize(u)
				if u.String() != tc.want || u.SearchParams() != params || params.String() != form {
					t.Fatalf("Canonicalize: URL=%q form=%q, want URL=%q form=%q", u, params.String(), tc.want, form)
				}
			}
		})
	}
}

func TestRemoveEmptyQueryEqualsWithSortQuery(t *testing.T) {
	checkProfile(t, canon.New(canon.SortQuery, canon.RemoveEmptyQueryEquals),
		"https://example.org/?b=&a=z&a=&a=a&a&b=2",
		"https://example.org/?a=z&a&a=a&a&b&b=2")
}

func TestRepeatedPercentDecodingPreservesBytes(t *testing.T) {
	p := canon.New(canon.RepeatedPercentDecoding())
	for _, tc := range []struct{ input, want string }{
		// Escaped bytes retain their values without character transcoding.
		{"http://example.com/%E5", "http://example.com/%E5"},
		{"http://example.com/%80", "http://example.com/%80"},
		{"http://example.org/%25FC?q=%2526%253D%252B&flag&flag=&q=a+b&&#%25FE", "http://example.org/%FC?q=%26%3D+&flag&flag=&q=a+b&&#%FE"},
		{"http://[2001:db8::1]/%25FC", "http://[2001:db8::1]/%FC"},
		{"custom:%25FC?q=%25FD#%25FE", "custom:%FC?q=%FD#%FE"},
		{"http://example.org/%GG%?q=%GG%2", "http://example.org/%25GG%25?q=%25GG%252"},
	} {
		t.Run(tc.input, func(t *testing.T) { checkProfile(t, p, tc.input, tc.want) })
	}
}

func TestPercentNormalizationPreservesComponentBoundaries(t *testing.T) {
	const input = "http://example%252F.org/a%253Fb%2523c%255Cd?q=%2526%253D"
	p := canon.New(canon.RepeatedPercentDecoding()).WithParser(url.NewParser(url.WithLaxHostParsing()))
	u := checkProfile(t, p, input, "http://example%2F.org/a%3Fb%23c%5Cd?q=%26%3D")
	if u.Hostname() != "example%2F.org" || u.Query() != "q=%26%3D" || u.Fragment() != "" {
		t.Fatalf("escape normalization changed component boundaries: %q", u)
	}
}

func TestOptionalComponentsAndOpaquePaths(t *testing.T) {
	for _, input := range []string{"custom:opaque", "custom:opaque?", "custom:opaque#", "custom:opaque?#"} {
		u := checkProfile(t, canon.New(canon.SortQuery, canon.RepeatedPercentDecoding()), input, input)
		if !u.OpaquePath() {
			t.Fatal("pipeline changed the path kind")
		}
	}
	var queryPresent, fragmentPresent bool
	p := canon.New(
		canon.RewriteQueryString(func(_ *url.Url, raw string, hasQuery bool) (string, bool) {
			queryPresent = hasQuery
			return raw, hasQuery
		}),
		canon.RewriteFragment(func(_ *url.Url, raw string, hasFragment bool) (string, bool) {
			fragmentPresent = hasFragment
			return raw, hasFragment
		}),
	)
	checkProfile(t, p, "custom:opaque?#", "custom:opaque?#")
	if !queryPresent || !fragmentPresent {
		t.Fatal("optional rewriters did not observe present-but-empty components")
	}
	checkProfile(t, p, "custom:opaque", "custom:opaque")
	if queryPresent || fragmentPresent {
		t.Fatal("optional rewriters did not observe absent components")
	}
}

func TestRewriteAdaptersSnapshotCallbacksAndRefreshMetadata(t *testing.T) {
	rewriters := []url.ComponentRewriter{nil, func(_ *url.Url, _ string) string { return "127.0.0.1" }}
	p := canon.New(canon.RewriteHostname(rewriters...))
	rewriters[1] = func(_ *url.Url, _ string) string { return "changed.example" }
	u := checkProfile(t, p, "https://example.org/", "https://127.0.0.1/")
	if !u.IsIPv4() || u.IsIPv6() {
		t.Fatal("rewritten hostname retained stale IP metadata")
	}
}

func TestCanonicalizeRefreshesExistingSearchParams(t *testing.T) {
	u, err := url.Parse("https://example.org/?q=%25FC&flag")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	clone := u.Clone()
	canon.New(canon.RepeatedPercentDecoding(), canon.SortQuery).Canonicalize(u)
	if u.SearchParams() != params || params.Get("q") != "\ufffd" || !params.Has("flag") {
		t.Fatalf("stale SearchParams after canonicalization: %q", params.String())
	}
	if u.String() != "https://example.org/?flag&q=%FC" || clone.String() != "https://example.org/?q=%25FC&flag" {
		t.Fatalf("URL = %q, caller's clone = %q", u, clone)
	}
}

func TestSharedProfile(t *testing.T) {
	p := canon.IA().Then(canon.RemovePort)
	for range 8 {
		t.Run("parallel", func(t *testing.T) {
			t.Parallel()
			for range 10 {
				checkProfile(t, p, "http://www.example.org:8080/A?B=2&A=1#f", "http://example.org/a?a=1&b=2")
			}
		})
	}
}
