package canonicalizer

import (
	"strconv"
	"strings"
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

var profileURLSink *url.Url

func BenchmarkProfiles(b *testing.B) {
	var query strings.Builder
	for i := 0; i < 64; i++ {
		if i > 0 {
			query.WriteByte('&')
		}
		query.WriteString("key" + strconv.Itoa(64-i) + "=value%20" + strconv.Itoa(i))
	}
	for _, profile := range []struct {
		name string
		p    Profile
	}{
		{"Sorted", WhatWgSortQuery()},
		{"Google", GoogleSafeBrowsing()},
		{"Semantic", Semantic()},
		{"IA", IA()},
	} {
		for _, input := range []struct{ name, raw string }{
			{"Clean", "https://www.example.org/path?q=a+b&flag"},
			{"Nested", "https://www.example.org/%252525252541?q=%252525252542&flag#fragment"},
			{"LargeQuery", "https://www.example.org/path?" + query.String()},
		} {
			b.Run(profile.name+"/"+input.name, func(b *testing.B) {
				b.Run("Parse", func(b *testing.B) {
					for b.Loop() {
						u, err := profile.p.Parser().Parse(input.raw)
						if err != nil {
							b.Fatal(err)
						}
						profileURLSink = u
					}
				})
				b.Run("ParseAndCanonicalize", func(b *testing.B) {
					for b.Loop() {
						u, err := profile.p.Parse(input.raw)
						if err != nil {
							b.Fatal(err)
						}
						profileURLSink = u
					}
				})
				for _, params := range []bool{false, true} {
					name := "Canonicalize"
					if params {
						name += "WithSearchParams"
					}
					b.Run(name, func(b *testing.B) {
						seed, err := profile.p.Parser().Parse(input.raw)
						if err != nil {
							b.Fatal(err)
						}
						if params {
							seed.SearchParams()
						}
						// Prepare fresh inputs in batches outside the timer so each
						// operation sees the original escapes and query order.
						const batchSize = 64
						var batch [batchSize]*url.Url
						b.ResetTimer()
						for i := 0; i < b.N; {
							b.StopTimer()
							n := min(batchSize, b.N-i)
							for j := 0; j < n; j++ {
								batch[j] = seed.Clone()
								if params {
									batch[j].SearchParams()
								}
							}
							b.StartTimer()
							for j := 0; j < n; j++ {
								profile.p.Canonicalize(batch[j])
								profileURLSink = batch[j]
							}
							i += n
						}
					})
				}
			})
		}
	}
}
