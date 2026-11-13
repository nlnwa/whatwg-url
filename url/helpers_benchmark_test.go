package url

import (
	"strconv"
	"strings"
	"testing"
)

var helperStringSink string

func benchmarkQuery(n int) Query {
	q := make(Query, n)
	for i := range q {
		q[i] = QueryParameter{"key" + strconv.Itoa(n-i), "value%20" + strconv.Itoa(i), true}
	}
	return q
}

func BenchmarkQueryHelpers(b *testing.B) {
	for _, absolute := range []bool{false, true} {
		name := "SortNames64"
		if absolute {
			name = "SortAbsolute64"
		}
		b.Run(name, func(b *testing.B) {
			seed := benchmarkQuery(64)
			q := make(Query, len(seed))
			b.ReportAllocs()
			for b.Loop() {
				copy(q, seed)
				if absolute {
					q.SortAbsolute()
				} else {
					q.Sort()
				}
			}
		})
	}
	b.Run("String97", func(b *testing.B) {
		q := benchmarkQuery(97)
		b.ReportAllocs()
		for b.Loop() {
			helperStringSink = q.String()
		}
	})
}

func BenchmarkPercentHelpers(b *testing.B) {
	for _, tc := range []struct{ name, input string }{
		{"ASCII", "unchanged/path+query"},
		{"ASCIIWithEquals", "unchanged/path?query=a+b"},
		{"SinglePass", "%41%2F%FF%2B+"},
		{"Nested8", "%" + strings.Repeat("25", 8) + "41"},
		{"Nested64", "%" + strings.Repeat("25", 64) + "41"},
		{"NestedPercent64", "prefix/%" + strings.Repeat("25", 64) + "25/suffix"},
		{"Boundaries", "%25%32%35%34%31"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				helperStringSink = PercentEncode(PercentDecodeRepeated(tc.input), helperPercentEncodeSet)
			}
		})
	}
	for _, decode := range []bool{false, true} {
		name := "EncodeUnchanged"
		if decode {
			name = "DecodeUnchanged"
		}
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				if decode {
					helperStringSink = PercentDecode("unchanged+a%GG")
				} else {
					helperStringSink = PercentEncode("unchanged+a", PathPercentEncodeSet)
				}
			}
		})
	}
}

func BenchmarkIdentityRewrites(b *testing.B) {
	for _, input := range []string{"https://example.org/path?q=a+b&x=%20&flag", "https://[::1]/path?q=a+b&x=%20&flag"} {
		b.Run(input, func(b *testing.B) {
			u, err := Parse(input)
			if err != nil {
				b.Fatal(err)
			}
			u.SearchParams()
			b.ReportAllocs()
			for b.Loop() {
				u.RewriteHostname(func(_ *Url, s string) string { return s })
				u.RewritePathname(func(_ *Url, s string) string { return s })
				u.RewriteQueryString(func(_ *Url, s string, hasQuery bool) (string, bool) { return s, hasQuery })
				u.RewriteFragment(func(_ *Url, s string, hasFragment bool) (string, bool) { return s, hasFragment })
			}
		})
	}
}
