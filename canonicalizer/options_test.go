package canonicalizer_test

import (
	"slices"
	"strings"
	"testing"

	canon "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestConfiguredRulesCaptureIndependentPredicates(t *testing.T) {
	first := canon.FilterQuery(func(name, _ string, _ bool) bool { return name != "a" })
	second := canon.FilterQuery(func(name, _ string, _ bool) bool { return name != "b" })
	p := canon.New(first, canon.RewriteQueryString(func(_ *url.Url, raw string, hasQuery bool) (string, bool) {
		return raw + "&b=2&c=3", hasQuery
	}), second)
	checkProfile(t, p, "https://example.org/?a=1&b=1&keep=0", "https://example.org/?keep=0&c=3")
	checkProfile(t, canon.New(first), "https://example.org/?a=1&b=1", "https://example.org/?b=1")
	checkProfile(t, canon.New(second), "https://example.org/?a=1&b=1", "https://example.org/?a=1")
}

func TestSharedRulesPreserveQueryTerms(t *testing.T) {
	filter := canon.FilterQuery(func(name, _ string, _ bool) bool { return name != "drop" })
	p := canon.New(canon.MassageHost, filter, canon.SortQueryAbsolute)
	checkProfile(t, p,
		"http://www.example.org/?q=a+b&q=%20&drop=1&flag&empty=&bytes=%FC&&dup=2&dup=1&encoded%26name=x%3Dy",
		"http://example.org/?&bytes=%FC&dup=1&dup=2&empty=&encoded%26name=x%3Dy&flag&q=%20&q=a+b")
	checkProfile(t, canon.New(filter, canon.SortQueryAbsolute), "http://www.example.org/?flag", "http://www.example.org/?flag")
	checkProfile(t, p, "http://[::1]/?flag", "http://[::1]/?flag")
}

func TestQueryFilterPresenceAndSynchronization(t *testing.T) {
	canon.FilterQuery(nil)(nil)
	keep := canon.New(canon.FilterQuery(func(_, _ string, _ bool) bool { return true }))
	for _, input := range []string{"custom:opaque", "custom:opaque?", "custom:opaque?#"} {
		checkProfile(t, keep, input, input)
		checkProfile(t, canon.New(canon.FilterQuery(nil)), input, input)
	}
	drop := canon.New(canon.FilterQuery(func(_, _ string, _ bool) bool { return false }))
	checkProfile(t, drop, "custom:opaque?", "custom:opaque")
	checkProfile(t, drop, "custom:opaque?x#", "custom:opaque#")

	u, err := url.Parse("https://example.org/?bytes=%FC&drop=1&flag")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	p := canon.New(canon.FilterQuery(func(name, _ string, _ bool) bool { return name != "drop" }))
	p.Canonicalize(u)
	if u.String() != "https://example.org/?bytes=%FC&flag" || u.SearchParams() != params || params.Has("drop") || params.Get("bytes") != "\ufffd" {
		t.Fatalf("filter did not preserve URL spelling and refresh SearchParams: %q, %q", u, params.String())
	}
	p.Canonicalize(nil)
}

func TestQueryFilterOptionSnapshotAndOrder(t *testing.T) {
	var seen []string
	filters := map[string]canon.QueryFilter{
		"z":   func(_, _ string, _ bool) bool { seen = append(seen, "z"); return true },
		"b":   func(_, _ string, _ bool) bool { seen = append(seen, "b"); return false },
		"a":   func(_, _ string, _ bool) bool { seen = append(seen, "a"); return true },
		"nil": nil,
	}
	option := canon.WithQueryFilters(filters)
	delete(filters, "b")
	filters["a"] = func(_, _ string, _ bool) bool { t.Fatal("option retained input map"); return true }
	first := canon.IA(option)
	delete(filters, "z")
	second := canon.IA(option)
	for _, p := range []canon.Profile{first, second} {
		seen = nil
		checkProfile(t, p, "https://example.org/?term", "https://example.org/")
		if !slices.Equal(seen, []string{"a", "b"}) {
			t.Fatalf("predicate order = %v", seen)
		}
	}
	for _, option := range []canon.IAOption{
		canon.WithQueryFilters(nil),
		canon.WithQueryFilters(map[string]canon.QueryFilter{}),
		canon.WithQueryFilters(map[string]canon.QueryFilter{"ignored": nil}),
		canon.WithQueryFilter(nil),
	} {
		checkProfile(t, canon.IA(option), "https://example.org/?cfid=1", "https://example.org/?cfid=1")
	}
}

func TestIAFactoryOptionPrecedenceAndIndependence(t *testing.T) {
	keep := func(name, _ string, _ bool) bool { return name == "term" }
	filters := map[string]canon.QueryFilter{"only-term": keep}
	base := canon.IA()
	custom := canon.IA(canon.WithHostMassage(false), canon.WithQueryFilters(filters))
	input := "https://www.example.org/?cfid=1&term&other"
	checkProfile(t, base, input, "https://example.org/?other&term")
	checkProfile(t, custom, input, "https://www.example.org/?term")
	checkProfile(t, canon.IA(), input, "https://example.org/?other&term")
	checkProfile(t, canon.IA(nil, canon.WithHostMassage(false), canon.WithHostMassage(true), canon.WithQueryFilter(nil)), input, "https://example.org/?cfid=1&other&term")
	checkProfile(t, canon.IA(canon.WithQueryFilters(filters), canon.WithQueryFilter(nil)), input, "https://example.org/?cfid=1&other&term")
	checkProfile(t, canon.IA(canon.WithQueryFilter(nil), canon.WithQueryFilters(filters)), input, "https://example.org/?term")
	checkProfile(t, canon.IA(canon.WithQueryFilters(nil), canon.WithQueryFilter(keep)), input, "https://example.org/?term")
	defaults := canon.DefaultSessionQueryFilters()
	delete(defaults, "cfid")
	if len(canon.DefaultSessionQueryFilters()) != 6 {
		t.Fatal("default filter maps share mutable state")
	}
	checkProfile(t, base, input, "https://example.org/?other&term")
}

func TestDecodeOptionsAreCapturedPerRule(t *testing.T) {
	options := []canon.DecodeOption{canon.WithAllowForbiddenOpaqueHost(true)}
	allow := canon.RepeatedPercentDecoding(options...)
	semantic := canon.Semantic(options...)
	options[0] = canon.WithAllowForbiddenOpaqueHost(false)
	escape := canon.RepeatedPercentDecoding(options...)
	parser := url.NewParser(url.WithLaxHostParsing())
	input := "custom://a^b/"
	checkProfile(t, canon.New(allow).WithParser(parser), input, input)
	checkProfile(t, canon.New(escape).WithParser(parser), input, "custom://a%5Eb/")
	checkProfile(t, semantic, input, input)
	checkProfile(t, canon.Semantic(options...), input, "custom://a%5Eb/")
	checkProfile(t, canon.Semantic(nil, canon.WithAllowForbiddenOpaqueHost(false), canon.WithAllowForbiddenOpaqueHost(true)), input, input)
	checkProfile(t, canon.New(canon.RepeatedPercentDecoding(nil)).WithParser(parser), input, "custom://a%5Eb/")
	checkProfile(t, canon.New(allow, escape).WithParser(parser), input, "custom://a%5Eb/")
	checkProfile(t, canon.New(escape, allow).WithParser(parser), input, input)
}

func TestIADefaultFiltersAreIndependentlyReplaceable(t *testing.T) {
	id := strings.Repeat("A", 32)
	asp := strings.Repeat("Z", 24)
	for _, tc := range []struct{ key, term, normalized string }{
		{"jsessionid", "JSESSIONID=" + id, "jsessionid=" + strings.ToLower(id)},
		{"phpsessid", "PHPSESSID=" + id, "phpsessid=" + strings.ToLower(id)},
		{"sid", "SID=" + id, "sid=" + strings.ToLower(id)},
		{"aspsessionid", "ASPSESSIONIDABCDEFGH=" + asp, "aspsessionidabcdefgh=" + strings.ToLower(asp)},
		{"cfid", "CFID=123", "cfid=123"},
		{"cftoken", "CFTOKEN", "cftoken"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			input := "http://example.org/?" + tc.term
			checkProfile(t, canon.IA(), input, "http://example.org/")
			filters := canon.DefaultSessionQueryFilters()
			if len(filters) != 6 || filters[tc.key] == nil {
				t.Fatalf("missing default filter %q", tc.key)
			}
			delete(filters, tc.key)
			checkProfile(t, canon.IA(canon.WithQueryFilters(filters)), input, "http://example.org/?"+tc.normalized)
			checkProfile(t, canon.IA(), input, "http://example.org/")
		})
	}
}

func TestIAFilterPredicatesRetainHistoricalConditions(t *testing.T) {
	filters := canon.DefaultSessionQueryFilters()
	keep := func(name, value string, hasEquals bool) bool {
		for _, predicate := range filters {
			if !predicate(name, value, hasEquals) {
				return false
			}
		}
		return true
	}
	for _, tc := range []struct {
		name, value  string
		equals, keep bool
	}{
		{"SID", strings.Repeat("a", 32), true, false},
		{"sid", strings.Repeat("a", 31), true, true},
		{"sid", strings.Repeat("-", 32), true, true},
		{"sid", strings.Repeat("a", 32), false, true},
		{"othersid", strings.Repeat("a", 32), true, true},
		{"ASPSESSIONIDABCDEFGH", strings.Repeat("!", 24), true, false},
		{"ASPSESSIONIDABCDEFGH", strings.Repeat("x", 23), true, true},
		{"ASPSESSIONIDABCDEFG", strings.Repeat("x", 24), true, true},
		{"ASPSESSIONIDABCDEFGH", strings.Repeat("x", 24), false, true},
		{"CFID", "", false, false},
		{"CFTOKEN", "", false, false},
		{"flag", "", false, true},
	} {
		if got := keep(tc.name, tc.value, tc.equals); got != tc.keep {
			t.Errorf("filter(%q, %q, %v) = %v, want %v", tc.name, tc.value, tc.equals, got, tc.keep)
		}
	}
}

func TestIACustomFilterAndSpecialScheme(t *testing.T) {
	filters := canon.DefaultSessionQueryFilters()
	delete(filters, "phpsessid")
	filters["tracking"] = func(name, _ string, _ bool) bool { return !strings.HasPrefix(name, "utm_") }
	p := canon.IA(canon.WithHostMassage(false), canon.WithQueryFilters(filters))
	id := strings.Repeat("a", 32)
	checkProfile(t, p, "http://www.example.org/?utm_source=x&phpsessid="+id+"&cfid=1&keep=2",
		"http://www.example.org/?keep=2&phpsessid="+id)
	p = canon.IA().WithParser(url.NewParser(url.WithSpecialSchemes(map[string]uint16{"archive": 81})))
	checkProfile(t, p, "archive://www.example.org:81/A?b=2&a=1", "archive://example.org/a?a=1&b=2")
	checkProfile(t, canon.IA(), "archive://www.example.org:81/A?b=2&a=1", "archive://example.org:81/a?a=1&b=2")
}
