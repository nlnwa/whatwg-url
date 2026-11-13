package url

import (
	"slices"
	"testing"
)

func TestRewriteHostnameClassification(t *testing.T) {
	for _, tc := range []struct {
		input string
		ipv4  bool
		ipv6  bool
	}{
		{"custom://127.0.0.1/", false, false},
		{"http://127.0.0.1/", true, false},
		{"http://[::1]/", false, true},
		{"custom://[::1]/", false, true},
		{"http://[BOGUS]/", false, false},
		{"custom://[BOGUS]/", false, false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			u, err := NewParser(WithLaxHostParsing()).Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if allocs := testing.AllocsPerRun(100, func() {
				u.RewriteHostname(nil, func(_ *Url, host string) string { return host })
			}); allocs != 0 {
				t.Errorf("identity hostname rewrite allocated %v times", allocs)
			}
			if u.IsIPv4() != tc.ipv4 || u.IsIPv6() != tc.ipv6 || u.String() != tc.input {
				t.Fatalf("identity rewrite changed host or metadata: %s", u)
			}
			for _, host := range []string{"127.0.0.2", "[::2]", "[invalid]", "example.org"} {
				u.RewriteHostname(func(_ *Url, _ string) string { return host })
				if u.IsIPv4() != (host == "127.0.0.2" && u.IsSpecialScheme()) || u.IsIPv6() != (host == "[::2]") {
					t.Fatalf("changed host %q has ipv4=%v ipv6=%v", host, u.IsIPv4(), u.IsIPv6())
				}
			}
		})
	}
}

func TestRewriteCallbacksCallingSetters(t *testing.T) {
	u, err := Parse("https://127.0.0.1/original?original=1#original")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	var seen []string
	u.RewriteHostname(
		func(u *Url, host string) string {
			seen = append(seen, host)
			u.SetHostname("[::1]")
			return host
		}, nil,
		func(_ *Url, host string) string { seen = append(seen, host); return host },
	)
	if !slices.Equal(seen, []string{"127.0.0.1", "127.0.0.1"}) || !u.IsIPv4() || u.IsIPv6() || u.Hostname() != "127.0.0.1" {
		t.Fatal("hostname callbacks did not restore returned host and metadata in order")
	}
	u.RewritePathname(func(u *Url, path string) string { u.SetPathname("/setter"); return path })
	if u.Pathname() != "/original" {
		t.Fatal("pathname callback did not restore returned path")
	}
	seen = nil
	u.RewriteQueryString(
		func(u *Url, raw string, hasQuery bool) (string, bool) {
			seen = append(seen, raw)
			u.SetSearch("?setter=2")
			return raw, hasQuery
		}, nil,
		func(_ *Url, raw string, hasQuery bool) (string, bool) {
			seen = append(seen, raw)
			return raw, hasQuery
		},
	)
	if !slices.Equal(seen, []string{"original=1", "original=1"}) || u.Query() != "original=1" || params.Get("original") != "1" || params.Has("setter") || u.SearchParams() != params {
		t.Fatal("query callback did not restore query and existing SearchParams in order")
	}
	seen = nil
	u.RewriteFragment(
		func(u *Url, raw string, hasFragment bool) (string, bool) {
			seen = append(seen, raw)
			u.SetHash("#setter")
			return raw, hasFragment
		}, nil,
		func(_ *Url, raw string, hasFragment bool) (string, bool) {
			seen = append(seen, raw)
			return raw, hasFragment
		},
	)
	if !slices.Equal(seen, []string{"original", "original"}) || u.Fragment() != "original" {
		t.Fatal("fragment callbacks did not restore returned fragment in order")
	}
	// When the callback returns the setter's state, its synchronization is
	// already complete and must remain available to subsequent callbacks.
	u.RewriteQueryString(func(u *Url, _ string, _ bool) (string, bool) {
		u.SetSearch("?setter=3")
		return u.query, u.hasQuery
	}, func(_ *Url, raw string, hasQuery bool) (string, bool) {
		if params.Get("setter") != "3" {
			t.Fatal("setter's form view was not available to next callback")
		}
		return raw, hasQuery
	})
	if params.Get("setter") != "3" || u.Query() != "setter=3" {
		t.Fatal("setter's query state was lost")
	}
}

func TestIdentityRewritesAllocationBudget(t *testing.T) {
	u, err := Parse("https://[::1]/path?q=a+b&space=%20&flag#fragment")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	if allocs := testing.AllocsPerRun(100, func() {
		u.RewriteHostname(func(_ *Url, s string) string { return s })
		u.RewritePathname(func(_ *Url, s string) string { return s })
		u.RewriteQueryString(func(_ *Url, s string, hasQuery bool) (string, bool) { return s, hasQuery })
		u.RewriteFragment(func(_ *Url, s string, hasFragment bool) (string, bool) { return s, hasFragment })
	}); allocs != 0 {
		t.Fatalf("identity rewrites allocated %v times", allocs)
	}
	if u.SearchParams() != params || params.Get("q") != "a b" || params.Get("space") != " " || !params.Has("flag") {
		t.Fatal("identity rewrites changed form view")
	}
}

func TestRewriteOptionalCleanup(t *testing.T) {
	for _, query := range []bool{false, true} {
		for _, present := range []bool{false, true} {
			for _, otherPresent := range []bool{false, true} {
				for _, remove := range []bool{false, true} {
					u, err := Parse("data:text")
					if err != nil {
						t.Fatal(err)
					}
					if query {
						u.hasQuery, u.hasFragment = present, otherPresent
					} else {
						u.hasFragment, u.hasQuery = present, otherPresent
					}
					params := u.SearchParams()
					u.RewritePathname(func(_ *Url, _ string) string { return "text  " })
					var rewriters []OptionalComponentRewriter
					if remove {
						rewriters = append(rewriters, func(_ *Url, _ string, _ bool) (string, bool) { return "ignored", false })
					}
					if query {
						u.RewriteQueryString(rewriters...)
					} else {
						u.RewriteFragment(rewriters...)
					}
					want := "text  "
					if (remove || !present) && !otherPresent {
						want = "text"
					}
					if u.Pathname() != want || u.SearchParams() != params || u.query != "" || u.fragment != "" {
						t.Fatalf("query=%v present=%v other=%v remove=%v: URL=%q", query, present, otherPresent, remove, u)
					}
					if query && u.hasQuery != (present && !remove) || !query && u.hasFragment != (present && !remove) {
						t.Fatal("optional presence changed unexpectedly")
					}
				}
			}
		}
	}
}
