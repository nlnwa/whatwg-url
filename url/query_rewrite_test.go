package url

import (
	"slices"
	"testing"
)

func TestRewriteQueryPresence(t *testing.T) {
	for _, tc := range []struct {
		name, suffix string
		query        Query
		hasQuery     bool
		want         string
	}{
		{"create absent", "", Query{{"flag", "", false}}, true, "?flag"},
		{"absent identity", "", nil, false, ""},
		{"nil present", "?old=1", nil, true, "?"},
		{"empty present", "?old=1", Query{}, true, "?"},
		{"empty identity", "?", Query{{}}, true, "?"},
		{"remove", "?old=1", nil, false, ""},
		{"ignore absent terms", "?old=1", Query{{"ignored", "", false}}, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const base = "https://example.org/"
			u, err := Parse(base + tc.suffix)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			u.RewriteQuery(nil, func(got *Url, query Query, hasQuery bool) (Query, bool) {
				calls++
				if got != u || hasQuery != (tc.suffix != "") {
					t.Fatal("callback received wrong URL or presence")
				}
				if tc.suffix == "" && query != nil || tc.suffix == "?" && !slices.Equal(query, Query{{}}) {
					t.Fatalf("initial terms = %#v", query)
				}
				return tc.query, tc.hasQuery
			}, nil)
			if calls != 1 || u.String() != base+tc.want {
				t.Fatalf("calls=%d URL=%q, want %q", calls, u, base+tc.want)
			}
		})
	}
}

func TestRewriteQueryPipelineAndOwnership(t *testing.T) {
	u, err := Parse("https://example.org/?old=1")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	owned := Query{{"new", "a+b%2f", true}, {"flag", "ignored", false}}
	var calls []int
	u.RewriteQuery(
		func(u *Url, query Query, hasQuery bool) (Query, bool) {
			calls = append(calls, 1)
			if !hasQuery || !slices.Equal(query, ParseQuery("old=1")) {
				t.Fatalf("initial query = %#v, present=%v", query, hasQuery)
			}
			return owned, false
		}, nil,
		func(u *Url, query Query, hasQuery bool) (Query, bool) {
			calls = append(calls, 2)
			// Even while absent, intermediate terms are passed on unchanged.
			// The ignored bare-term value also proves there was no round trip.
			if hasQuery || &query[0] != &owned[0] || query[1].Value != "ignored" {
				t.Fatalf("intermediate terms/presence = %#v, %v", query, hasQuery)
			}
			if u.Query() != "old=1" || params.Get("old") != "1" || params.Has("new") {
				t.Fatal("intermediate edits were committed before the last callback")
			}
			return query, true
		},
	)
	if !slices.Equal(calls, []int{1, 2}) || u.Query() != "new=a+b%2f&flag" || u.SearchParams() != params || params.Get("new") != "a b/" || params.Has("old") {
		t.Fatalf("committed query/form/order: %v, %v, %v", u, params, calls)
	}
	owned[0].Name = "mutated"
	owned[1].Value = "also mutated"
	if u.Query() != "new=a+b%2f&flag" {
		t.Fatal("committed query aliases a retained callback slice")
	}
}

func TestRewriteQueryLossless(t *testing.T) {
	u, err := Parse("https://example.org/")
	if err != nil {
		t.Fatal(err)
	}
	const raw = "&q=a+b&q=%20&flag&empty=&dup=2&&dup=1&lower=%2f&bad=%GG&bytes=\xff\x00&"
	u.RewriteQueryString(func(_ *Url, _ string, _ bool) (string, bool) { return raw, true })
	u.RewriteQuery(func(_ *Url, query Query, hasQuery bool) (Query, bool) {
		return query, hasQuery
	})
	if u.Query() != raw {
		t.Fatalf("identity rewrite changed bytes: %q", u.Query())
	}
}

func TestRewriteQuerySetterCallbacks(t *testing.T) {
	for _, keepSetter := range []bool{false, true} {
		u, err := Parse("https://example.org/?original=1")
		if err != nil {
			t.Fatal(err)
		}
		params := u.SearchParams()
		u.RewriteQuery(func(u *Url, query Query, hasQuery bool) (Query, bool) {
			u.SetSearch("?setter=2")
			if keepSetter {
				return ParseQuery(u.Query()), true
			}
			return query, hasQuery
		}, func(u *Url, query Query, hasQuery bool) (Query, bool) {
			if u.Query() != "setter=2" || params.Get("setter") != "2" {
				t.Fatal("setter changes were not immediately visible")
			}
			return query, hasQuery
		})
		want := "original=1"
		if keepSetter {
			want = "setter=2"
		}
		if u.Query() != want || u.SearchParams() != params || params.String() != want {
			t.Fatalf("query/form after setter callback: %v, %v; want %q", u, params, want)
		}
	}
}

func TestRewriteQueryOptionalCleanup(t *testing.T) {
	for _, fragment := range []bool{false, true} {
		for _, rewriters := range [][]QueryRewriter{
			nil,
			{nil, nil},
			{func(_ *Url, query Query, hasQuery bool) (Query, bool) { return query, hasQuery }},
			{func(_ *Url, _ Query, _ bool) (Query, bool) { return nil, false }},
		} {
			u, err := Parse("data:text")
			if err != nil {
				t.Fatal(err)
			}
			if fragment {
				u.SetHash("#")
			}
			u.RewritePathname(func(_ *Url, _ string) string { return "text  " })
			u.RewriteQuery(rewriters...)
			want := "text"
			if fragment {
				want += "  "
			}
			if u.Pathname() != want || u.hasQuery {
				t.Fatalf("optional cleanup with fragment=%v: %q", fragment, u)
			}
		}
	}
	u, err := Parse("data:text?q=1")
	if err != nil {
		t.Fatal(err)
	}
	u.RewritePathname(func(_ *Url, _ string) string { return "text  " })
	params := u.SearchParams()
	u.RewriteQuery(func(_ *Url, _ Query, _ bool) (Query, bool) { return nil, false })
	if u.String() != "data:text" || u.hasQuery || u.SearchParams() != params || params.Has("q") {
		t.Fatalf("removing a present query did not clean the opaque path/form view: %v, %v", u, params)
	}
	var absent *Url
	absent.RewriteQuery(func(_ *Url, _ Query, _ bool) (Query, bool) {
		t.Fatal("callback ran on a nil URL")
		return nil, false
	})
}

func TestRewriteQueryNoCallbacksAllocationBudget(t *testing.T) {
	u, err := Parse("https://example.org/?q=a+b&space=%20&flag")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	if allocs := testing.AllocsPerRun(100, func() {
		u.RewriteQuery()
		u.RewriteQuery(nil, nil)
	}); allocs != 0 {
		t.Fatalf("empty callback lists allocated %v times", allocs)
	}
	if u.Query() != "q=a+b&space=%20&flag" || u.SearchParams() != params || params.Get("q") != "a b" {
		t.Fatal("empty callback lists changed query or form view")
	}
}
