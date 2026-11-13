package url

import "testing"

func TestSkipDotSegmentNormalization(t *testing.T) {
	parser := NewParser(WithSkipDotSegmentNormalization())
	for _, tc := range []struct{ input, preserved, standard string }{
		{"http://example.org/a/./b/../%2e/%2E%2e/c", "/a/./b/../%2e/%2E%2e/c", "/c"},
		{"custom://host/a/../b/.", "/a/../b/.", "/b/"},
		{"custom:opaque/../a", "opaque/../a", "opaque/../a"},
		{"file:///C|/a/../b", "/C:/a/../b", "/C:/b"},
		{"http://example.org/a//../b/..", "/a//../b/..", "/a/"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			u, err := parser.Parse(tc.input)
			if err != nil || u.Pathname() != tc.preserved {
				t.Fatalf("preserving parse = %v, %v; want %q", u, err, tc.preserved)
			}
			standard, err := Parse(tc.input)
			if err != nil || standard.Pathname() != tc.standard {
				t.Fatalf("standard parse = %v, %v; want %q", standard, err, tc.standard)
			}
		})
	}
	u, err := parser.Parse("http://example.org/")
	if err != nil {
		t.Fatal(err)
	}
	u.SetPathname("/a/../%2e/b/.")
	if u.Pathname() != "/a/../%2e/b/." {
		t.Fatalf("setter lost dot segments: %v", u)
	}
	u, err = parser.ParseRef("http://example.org/a/../dir/", "../%2e/child")
	if err != nil || u.Pathname() != "/a/../dir/../%2e/child" {
		t.Fatalf("reference lost dot segments: %v, %v", u, err)
	}
	u, err = NewParser(WithSkipDotSegmentNormalization(), WithCollapseConsecutiveSlashes()).Parse("http://example.org//a///../b")
	if err != nil || u.Pathname() != "/a/../b" {
		t.Fatalf("slash collapsing no longer independent: %v, %v", u, err)
	}
}
