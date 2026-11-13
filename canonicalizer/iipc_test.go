package canonicalizer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	whatwgurl "github.com/nationallibraryofnorway/whatwg-url/url"
)

const iipcURLCanonRevision = "842afd4d84089cbbef87dd3286edaf053caedb0a"

type iipcURLCanonVectors map[string]map[string]string

func loadIIPCURLCanonVectors(t *testing.T, name string) iipcURLCanonVectors {
	t.Helper()
	path := filepath.Join("..", "testdata", "iipc-urlcanon", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read IIPC urlcanon test data at revision %s: %v", iipcURLCanonRevision, err)
	}
	var vectors iipcURLCanonVectors
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return vectors
}

func runStringVectors(t *testing.T, vectors map[string]string, transform func(string) (string, error)) {
	t.Helper()
	inputs := make([]string, 0, len(vectors))
	for input := range vectors {
		inputs = append(inputs, input)
	}
	sort.Strings(inputs)
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			got, err := transform(input)
			if err != nil {
				t.Fatalf("transform: %v", err)
			}
			if want := vectors[input]; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestIIPCURLCanonIACanonicalizationData(t *testing.T) {
	vectors := loadIIPCURLCanonVectors(t, "surt.json")
	for _, group := range []string{"Canonicalizer.DefaultIA", "Canonicalizer.IA"} {
		t.Run(group, func(t *testing.T) {
			runStringVectors(t, vectors[group], func(input string) (string, error) {
				u, err := IA().Parse(input)
				if err != nil {
					return "", err
				}
				return u.String(), nil
			})
		})
	}
}

func TestIAAboveRootDotSegments(t *testing.T) {
	// Python SURT's normalizePath can retain above-root parents. The IA
	// profile resolves these with WHATWG parsing before its rules run.
	for _, input := range []string{
		"https://domain.com/../path",
		"https://domain.com/a/../path",
		"https://domain.com/a/../../path",
		"https://domain.com/%2e%2e/path",
	} {
		t.Run(input, func(t *testing.T) {
			u, err := IA().Parse(input)
			if err != nil || u.String() != "https://domain.com/path" {
				t.Fatalf("IA().Parse(%q) = %v, %v; want https://domain.com/path", input, u, err)
			}
			IA().Canonicalize(u)
			if u.String() != "https://domain.com/path" {
				t.Fatalf("second canonicalization changed path: %v", u)
			}
		})
	}
}

func TestIAQueryNameBoundaries(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"cfid%3Dx=1", "cfid%3dx=1"},
		{"CFID%253DX=1&cfid=removed", "cfid%3dx=1"},
		{"a%3Db=c", "a%3db=c"},
		{"a%3Db", "a%3db"},
		{"a%3Db=", "a%3db="},
		{"a%253Db=%253D%2526", "a%3db==%26"},
		{"a%3Db=2&a%3Db=1", "a%3db=1&a%3db=2"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			p := IA()
			u, err := p.Parser().Parse("http://example.org/?" + tc.input)
			if err != nil {
				t.Fatal(err)
			}
			params := u.SearchParams()
			for range 2 {
				p.Canonicalize(u)
				if u.Query() != tc.want {
					t.Fatalf("query = %q, want %q", u.Query(), tc.want)
				}
				if u.SearchParams() != params {
					t.Fatal("canonicalization replaced the SearchParams object")
				}
				if strings.HasPrefix(tc.want, "cfid%3d") && (params.Get("cfid=x") != "1" || params.Has("cfid")) {
					t.Fatal("SearchParams lost the escaped query-name boundary")
				}
			}
			reparsed, err := p.Parse(u.String())
			if err != nil || reparsed.String() != u.String() {
				t.Fatalf("reparse = %v, %v; want %v", reparsed, err, u)
			}
		})
	}
}

func TestIIPCURLCanonCanonicalizationPrimitiveData(t *testing.T) {
	vectors := loadIIPCURLCanonVectors(t, "surt.json")

	t.Run("alphaRorderQuery", func(t *testing.T) {
		for input, want := range vectors["alphaRorderQuery"] {
			query := whatwgurl.ParseQuery(input)
			query.SortAbsolute()
			if got := query.String(); got != want {
				t.Errorf("sorting %q = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("massageHost", func(t *testing.T) {
		for input, want := range vectors["massageHost"] {
			if got := massageHost(input); got != want {
				t.Errorf("massageHost(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("stripPathSessionId", func(t *testing.T) {
		for input, want := range vectors["stripPathSessionId"] {
			if got := stripPathSessionID(input); got != want {
				t.Errorf("stripPathSessionID(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("unescapeRepeatedly", func(t *testing.T) {
		for input, want := range vectors["unescapeRepeatedly"] {
			if got := whatwgurl.PercentDecodeRepeated(input); got != want {
				t.Errorf("PercentDecodeRepeated(%q) = %q, want %q", input, got, want)
			}
		}
	})
}

func TestUpstreamHostMassageCompatibility(t *testing.T) {
	tests := map[string]string{
		"foo.com":              "foo.com",
		"www.foo.com":          "foo.com",
		"www12.foo.com":        "foo.com",
		"www2foo.com":          "www2foo.com",
		"www2.www2foo.com":     "www2foo.com",
		"www.www2.example.com": "www2.example.com",
	}
	for input, want := range tests {
		if got := massageHost(input); got != want {
			t.Errorf("massageHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUpstreamPathSessionCompatibility(t *testing.T) {
	tests := map[string]string{
		"/(S(4hqa0555fwsecu455xqckv45))/mileg.aspx":                                                       "/mileg.aspx",
		"/(4hqa0555fwsecu455xqckv45)/mileg.aspx":                                                          "/mileg.aspx",
		"/(a(4hqa0555fwsecu455xqckv45)S(4hqa0555fwsecu455xqckv45)f(4hqa0555fwsecu455xqckv45))/mileg.aspx": "/mileg.aspx",
		"/photos/36050182@N05/":                                                                           "/photos/36050182@N05/",
	}
	for input, want := range tests {
		if got := stripPathSessionID(input); got != want {
			t.Errorf("stripPathSessionID(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIAPolicyOptions(t *testing.T) {
	const input = "http://www.example.org/?CFID=1&keep=2&flag"
	for _, tc := range []struct {
		name, want string
		profile    Profile
	}{
		{"defaults", "http://example.org/?flag&keep=2", IA()},
		{"disabled massage and filtering", "http://www.example.org/?cfid=1&flag&keep=2", IA(WithHostMassage(false), WithQueryFilters(nil))},
		{"custom replaces filter", "http://example.org/?cfid=1&flag", IA(WithQueryFilter(func(name, value string, hasEquals bool) bool {
			return name != "keep" || value != "2" || !hasEquals
		}))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := tc.profile.Parse(input)
			if err != nil || u.String() != tc.want {
				t.Fatalf("Parse() = %v, %v; want %q", u, err, tc.want)
			}
		})
	}
}
