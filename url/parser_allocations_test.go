//go:build !race

package url

import "testing"

// These budgets measure uninstrumented parsing. Race instrumentation changes
// allocation counts; functional parser tests still run in race builds.
func TestParseAllocationBudgets(t *testing.T) {
	// Match BenchmarkParse's parse-and-serialize operation, including its
	// invalid-host case, without depending on fixture numbering staying fixed.
	for _, tc := range []struct {
		name   string
		input  string
		budget float64
	}{
		{"2", "https://test:@test", 13},
		{"4", "non-special://test:@test/x", 14},
		{"87", "file://[example]/", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(1000, func() {
				got, err := testParser.Parse(tc.input)
				if err == nil {
					_ = got.String()
				}
			})
			if allocs > tc.budget {
				t.Errorf("Parse(%q) and serialization allocated %.0f times; budget %.0f", tc.input, allocs, tc.budget)
			}
		})
	}
}
