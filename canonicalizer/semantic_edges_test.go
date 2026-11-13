package canonicalizer_test

import (
	"testing"

	canon "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestSemanticDecodedPaths(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"http://example.org/a/%252e%252e/b", "http://example.org/b"},
		{"http://example.org/a/%252e%252e/../b", "http://example.org/b"},
		{"http://example.org/a//../b", "http://example.org/a/b"},
		{"http://example.org/a/%252F../b", "http://example.org/a/b"},
		{"http://example.org/a/%255c../b", "http://example.org/a/b"},
		{"http://example.org/a/%252e", "http://example.org/a/"},
		{"http://example.org/a/%252e%252e", "http://example.org/"},
		{"http://example.org/a///b/?", "http://example.org/a/b/?"},
		{"http://example.org/a/%253F%2523%25zz", "http://example.org/a/%3F%23%25zz"},
		{"custom://host/a//../b", "custom://host/a/b"},
		{"custom://host/a//b", "custom://host/a//b"},
		{"custom://host/a/%255c../b", "custom://host/a/\\../b"},
		{"custom:opaque/%252e%252e/b?", "custom:opaque/../b?"},
		{"custom:%252Fa/%252e%252e/b", "custom:/b"},
		{"file:///C:/a/%252e%252e/b", "file:///C:/b"},
		{"gopher://example.org:70/a/%252e%252e/b", "gopher://example.org/b"},
		{"http://example.org/a/%252e%252e/\x91?x=1#", "http://example.org/%91?x=1"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			u := checkProfile(t, canon.Semantic(), tc.input, tc.want)
			canon.Semantic().Canonicalize(u)
			if u.String() != tc.want {
				t.Fatalf("second canonicalization = %q, want %q", u, tc.want)
			}
		})
	}
	for _, tc := range []struct{ base, ref, want string }{
		{"http://example.org/a/%252e%252e/dir/", "../b", "http://example.org/b"},
		{"http://example.org/a/../dir/", "%252e%252e/b", "http://example.org/b"},
		{"http://example.org/a//dir/", "../b", "http://example.org/a/b"},
		{"http://example.org/a/%252e%252e/", "?x", "http://example.org/?x"},
		{"file:///C:/a/", "../b", "file:///C:/b"},
	} {
		u, err := canon.Semantic().ParseRef(tc.base, tc.ref)
		if err != nil || u.String() != tc.want {
			t.Fatalf("ParseRef(%q, %q) = %v, %v; want %q", tc.base, tc.ref, u, err, tc.want)
		}
	}
	u, err := canon.Semantic().Parser().Parse("http://example.org/a/%252e%252e/../b")
	if err != nil || u.Pathname() != "/a/%252e%252e/../b" {
		t.Fatalf("parse-only access lost segments: %v, %v", u, err)
	}
	opaque := checkProfile(t, canon.Semantic(), "custom:%252Fa/%252e%252e/b", "custom:/b")
	if !opaque.OpaquePath() {
		t.Fatal("rewriting changed the opaque path kind")
	}
	u, err = canon.Semantic().Parser().Parse("http://example.org/a/%252e%252e/../b?b=2&a=%2531&a=0")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	canon.Semantic().Canonicalize(u)
	if u.Pathname() != "/b" || u.Query() != "a=0&a=1&b=2" || u.SearchParams() != params || params.Get("a") != "0" {
		t.Fatalf("canonicalization failed to synchronize existing SearchParams: %v, %v", u, params)
	}
}

func TestSemanticSpecialHostPunctuation(t *testing.T) {
	checkProfile(t, canon.New(canon.RepeatedPercentDecoding()), "http://foo_bar/", "http://foo%5Fbar/")
	for _, tc := range []struct{ input, want string }{
		{"http://foo_bar/", "http://foo_bar/"},
		{"http://!foo_~.org/", "http://!foo_~.org/"},
		{"http://%2521foo%255F%257E.org/", "http://!foo_~.org/"},
		{"http://foo%252F%253A%2540%255B%255C%255D%253F%2523.org/", "http://foo%2F%3A%40%5B%5C%5D%3F%23.org/"},
		{"http://foo%2520%2525%257F%2591.org/", "http://foo%20%25%7F%91.org/"},
		{"custom://a%252Fb/", "custom://a%2Fb/"},
		{"custom://%255Bbad%255D/", "custom://%5Bbad%5D/"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			u := checkProfile(t, canon.Semantic(), tc.input, tc.want)
			host := u.Hostname()
			canon.Semantic().Canonicalize(u)
			if u.String() != tc.want || u.Hostname() != host {
				t.Fatalf("second canonicalization changed host: %v", u)
			}
			parsed, err := canon.Semantic().Parser().Parse(u.String())
			if err != nil || parsed.Hostname() != host || parsed.Pathname() != u.Pathname() {
				t.Fatalf("reparse changed component boundaries: %v, %v", parsed, err)
			}
		})
	}
}

func TestSemanticDecodedHostNormalization(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		ipv4        bool
	}{
		{"http://%2541.COM/", "http://a.com/", false},
		{"http://.127.1./", "http://127.0.0.1/", true},
		{"http://%2530x7f.1/", "http://127.0.0.1/", true},
		{"http://%25C3%25BC.example/", "http://xn--tda.example/", false},
		{"http://%25EF%25BC%2591%25E3%2580%2582%25EF%25BC%2592/", "http://1.0.0.2/", true},
		{"http://%25E3%2580%2582/", "http://0.0.0.0/", true},
		{"http://%2530%2539/", "http://09/", false},
		{"http://%25FF.example/", "http://%FF.example/", false},
		{"http://a%252Fb/", "http://a%2Fb/", false},
		{"custom://%2541.COM/", "custom://A.COM/", false},
		{"custom://.127.1./", "custom://127.1/", false},
		{"file://%256cocalhost/path", "file:///path", false},
		{"gopher://.127.1.:70/", "gopher://127.0.0.1/", true},
	} {
		t.Run(tc.input, func(t *testing.T) {
			p := canon.Semantic()
			u, err := p.Parser().Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			diagnostics := len(u.ValidationErrors())
			p.Canonicalize(u)
			for range 2 {
				if u.String() != tc.want || u.IsIPv4() != tc.ipv4 || u.IsIPv6() {
					t.Fatalf("canonicalization = %q, IPv4=%v, IPv6=%v; want %q, IPv4=%v", u, u.IsIPv4(), u.IsIPv6(), tc.want, tc.ipv4)
				}
				if len(u.ValidationErrors()) != diagnostics {
					t.Fatal("host normalization changed the original validation history")
				}
				p.Canonicalize(u)
			}
			for _, raw := range []string{tc.input, u.String()} {
				parsed := checkProfile(t, p, raw, tc.want)
				if parsed.IsIPv4() != tc.ipv4 || parsed.IsIPv6() {
					t.Fatalf("reparsed host metadata differs: %v", parsed)
				}
			}
		})
	}
	u, err := canon.Semantic().ParseRef("http://.127.1./dir/", "child")
	if err != nil || u.String() != "http://127.0.0.1/dir/child" || !u.IsIPv4() {
		t.Fatalf("relative resolution = %v, %v", u, err)
	}
}

func TestProfilesRecoverMalformedIPv6(t *testing.T) {
	for _, profile := range []canon.Profile{canon.Semantic(), canon.GoogleSafeBrowsing(), canon.IA(), canon.New(canon.RepeatedPercentDecoding()).WithParser(url.NewParser(url.WithLaxHostParsing()))} {
		for _, input := range []string{"http://[1::2::3]/", "http://[bogus]/", "custom://[unclosed/path"} {
			u := checkProfile(t, profile, input, input)
			if u.IsIPv4() || u.IsIPv6() || len(u.ValidationErrors()) == 0 {
				t.Fatalf("recovered host metadata/validation = %v", u)
			}
			profile.Canonicalize(u)
			if u.String() != input || u.IsIPv6() {
				t.Fatalf("second canonicalization changed recovered host: %v", u)
			}
		}
	}
}
