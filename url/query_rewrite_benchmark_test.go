package url

import "testing"

// The manual parse/transform/serialize pattern predates RewriteQuery.
// Compare equivalent batching to detect added allocation costs.
func rewriteQueryManually(u *Url, rewriters ...QueryRewriter) {
	u.RewriteQueryString(func(u *Url, raw string, hasQuery bool) (string, bool) {
		var query Query
		if hasQuery {
			query = ParseQuery(raw)
		}
		for _, rewrite := range rewriters {
			if rewrite != nil {
				query, hasQuery = rewrite(u, query, hasQuery)
			}
		}
		if !hasQuery {
			return "", false
		}
		return query.String(), true
	})
}

func BenchmarkRewriteQuery(b *testing.B) {
	query := benchmarkQuery(64)
	for i := range 3 {
		query[i].Value = "1"
	}
	identity := QueryRewriter(func(_ *Url, query Query, hasQuery bool) (Query, bool) {
		return query, hasQuery
	})
	var updates []QueryRewriter
	for i := range 3 {
		updates = append(updates, func(_ *Url, query Query, hasQuery bool) (Query, bool) {
			if query[i].Value == "1" {
				query[i].Value = "2"
			} else {
				query[i].Value = "1"
			}
			return query, hasQuery
		})
	}
	for _, tc := range []struct {
		name, suffix string
		rewriters    []QueryRewriter
	}{
		{"AbsentIdentity", "", []QueryRewriter{identity}},
		{"EmptyIdentity", "?", []QueryRewriter{identity}},
		{"Identity64", "?" + query.String(), []QueryRewriter{identity}},
		{"ThreeUpdates64", "?" + query.String(), updates},
	} {
		for _, formView := range []bool{false, true} {
			name := tc.name + "/WithoutSearchParams"
			if formView {
				name = tc.name + "/WithSearchParams"
			}
			b.Run(name, func(b *testing.B) {
				variants := []string{"ManualBatch", "APIBatch"}
				if len(tc.rewriters) > 1 {
					variants = append(variants, "APISeparate")
				}
				for _, variant := range variants {
					b.Run(variant, func(b *testing.B) {
						u, err := Parse("https://example.org/" + tc.suffix)
						if err != nil {
							b.Fatal(err)
						}
						if formView {
							u.SearchParams()
						}
						b.ReportAllocs()
						for b.Loop() {
							switch variant {
							case "ManualBatch":
								rewriteQueryManually(u, tc.rewriters...)
							case "APIBatch":
								u.RewriteQuery(tc.rewriters...)
							case "APISeparate":
								for _, rewrite := range tc.rewriters {
									u.RewriteQuery(rewrite)
								}
							}
						}
					})
				}
			})
		}
	}
	b.Run("NoCallbacks64", func(b *testing.B) {
		u, err := Parse("https://example.org/?" + query.String())
		if err != nil {
			b.Fatal(err)
		}
		u.SearchParams()
		b.ReportAllocs()
		for b.Loop() {
			u.RewriteQuery(nil, nil)
		}
	})
}

func TestRewriteQueryMatchesManualAllocationBudget(t *testing.T) {
	const input = "https://example.org/?q=a+b&space=%20&flag&bytes=%FF"
	for _, formView := range []bool{false, true} {
		for _, change := range []bool{false, true} {
			var counts [2]float64
			for variant := range 2 {
				u, err := Parse(input)
				if err != nil {
					t.Fatal(err)
				}
				if formView {
					u.SearchParams()
				}
				rewrite := func(_ *Url, query Query, hasQuery bool) (Query, bool) {
					if change {
						if query[0].Value == "1" {
							query[0].Value = "2"
						} else {
							query[0].Value = "1"
						}
					}
					return query, hasQuery
				}
				counts[variant] = testing.AllocsPerRun(100, func() {
					if variant == 0 {
						rewriteQueryManually(u, rewrite)
					} else {
						u.RewriteQuery(rewrite)
					}
				})
			}
			if counts[1] > counts[0] {
				t.Errorf("formView=%v change=%v: API allocated %v times, manual allocated %v", formView, change, counts[1], counts[0])
			}
		}
	}
}
