package canonicalizer

import (
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestQueryRulesPresence(t *testing.T) {
	for _, tc := range []struct {
		name         string
		profile      Profile
		emptyPresent bool
	}{
		{"sort", New(SortQuery), true},
		{"absolute", New(SortQueryAbsolute), true},
		{"remove empty equals", New(RemoveEmptyQueryEquals), true},
		{"decode", New(RepeatedPercentDecoding()), true},
		{"google", GoogleSafeBrowsing(), true},
		{"semantic", Semantic(), true},
		{"keep", New(FilterQuery(func(_, _ string, _ bool) bool { return true })), true},
		{"drop", New(FilterQuery(func(_, _ string, _ bool) bool { return false })), false},
		{"IA", IA(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, suffix := range []string{"", "?"} {
				wantPresent := suffix == "?" && tc.emptyPresent
				p := tc.profile.Then(RewriteQueryString(func(_ *url.Url, raw string, hasQuery bool) (string, bool) {
					if raw != "" || hasQuery != wantPresent {
						t.Fatalf("input suffix=%q: raw=%q present=%v, want presence=%v", suffix, raw, hasQuery, wantPresent)
					}
					return raw, hasQuery
				}))
				if _, err := p.Parse("http://example.org/" + suffix); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	for _, p := range []Profile{
		IA(),
		New(FilterQuery(func(name, _ string, _ bool) bool { return name != "sid" })),
	} {
		u, err := p.Parse("http://example.org/?sid=0123456789abcdef0123456789abcdef")
		if err != nil || u.String() != "http://example.org/" {
			t.Fatalf("fully filtered query: %v, %v", u, err)
		}
	}
}

func TestQueryRulesSynchronizeAtEveryBoundary(t *testing.T) {
	u, err := url.Parse("https://example.org/?b=%2541&a=2&drop=3")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	observe := func(want string) Rule {
		return func(u *url.Url) {
			if u.Query() != want || u.SearchParams() != params || params.String() != want {
				t.Fatalf("after rule: query=%q form=%q, want %q", u.Query(), params.String(), want)
			}
		}
	}
	p := New(
		RepeatedPercentDecoding(), observe("b=A&a=2&drop=3"),
		SortQueryAbsolute, observe("a=2&b=A&drop=3"),
		FilterQuery(func(name, _ string, _ bool) bool { return name != "drop" }), observe("a=2&b=A"),
	)
	p.Canonicalize(u)
}

func TestRewriteQueryAdapterSnapshotsCallbacks(t *testing.T) {
	callbacks := []url.QueryRewriter{nil, func(_ *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
		return append(query, url.QueryParameter{Name: "flag"}), true
	}}
	rule := RewriteQuery(callbacks...)
	callbacks[1] = func(_ *url.Url, _ url.Query, _ bool) (url.Query, bool) {
		t.Fatal("rule retained the caller's callback slice")
		return nil, false
	}
	rule(nil)
	for _, input := range []string{"https://example.org/", "https://example.org/?a=1"} {
		u, err := url.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		rule(u)
		if !u.SearchParams().Has("flag") {
			t.Fatalf("raw-query adapter did not run: %v", u)
		}
	}
}

func TestRewriteQueryAdapterCommitBoundaries(t *testing.T) {
	u, err := url.Parse("https://example.org/?a=1")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	p := New(RewriteQuery(
		func(_ *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
			query[0].Value = "2"
			return query, hasQuery
		}, nil,
		func(u *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
			if query[0].Value != "2" || u.Query() != "a=1" || params.Get("a") != "1" {
				t.Fatal("batched callbacks did not receive local terms and the committed URL")
			}
			return query, hasQuery
		},
	), RewriteQuery(func(u *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
		if u.Query() != "a=2" || params.Get("a") != "2" || u.SearchParams() != params {
			t.Fatal("previous rule was not committed before the next rule")
		}
		query[0].Value = "3"
		return query, hasQuery
	}))
	p.Canonicalize(u)
	if u.Query() != "a=3" || params.Get("a") != "3" || u.SearchParams() != params {
		t.Fatal("final raw-query rule did not synchronize the existing form view")
	}
}
