package surt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
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

func TestIIPCURLCanonDefaultIASURTData(t *testing.T) {
	vectors := loadIIPCURLCanonVectors(t, "surt.json")
	runStringVectors(t, vectors["defaultIASurt"], func(input string) (string, error) {
		return String(input)
	})
}

func TestIIPCURLCanonIACanonicalizationData(t *testing.T) {
	vectors := loadIIPCURLCanonVectors(t, "surt.json")
	for _, group := range []string{"Canonicalizer.DefaultIA", "Canonicalizer.IA"} {
		t.Run(group, func(t *testing.T) {
			runStringVectors(t, vectors[group], func(input string) (string, error) {
				u, err := iaParser.Parse(input)
				if err != nil {
					return "", err
				}
				canonicalize(u, defaultOptions())
				return u.String(), nil
			})
		})
	}
}

func TestIIPCURLCanonCanonicalizationPrimitiveData(t *testing.T) {
	vectors := loadIIPCURLCanonVectors(t, "surt.json")

	t.Run("alphaRorderQuery", func(t *testing.T) {
		for input, want := range vectors["alphaRorderQuery"] {
			query := whatwgurl.ParseRawQuery(input)
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

func TestIIPCURLCanonSSURTData(t *testing.T) {
	vectors := loadIIPCURLCanonVectors(t, "ssurt.json")

	t.Run("ssurtHost", func(t *testing.T) {
		for input, want := range vectors["ssurtHost"] {
			if got := ssurtHost(input); got != want {
				t.Errorf("ssurtHost(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("reverseHost", func(t *testing.T) {
		for input, want := range vectors["reverseHost"] {
			if got := reverseHost(input); got != want {
				t.Errorf("reverseHost(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("ssurt", func(t *testing.T) {
		runStringVectors(t, vectors["ssurt"], func(input string) (string, error) {
			return SSURTString(input, true)
		})
	})
}

func TestIIPCURLCanonDataGroupsAreAccountedFor(t *testing.T) {
	tests := []struct {
		file string
		want []string
	}{
		{
			file: "surt.json",
			want: []string{
				"Canonicalizer.DefaultIA", "Canonicalizer.IA", "alphaRorderQuery",
				"defaultIASurt", "google", "googleSupplementalFromSurt",
				"massageHost", "stripPathSessionId", "unescapeRepeatedly",
			},
		},
		{
			file: "ssurt.json",
			want: []string{
				"reverseHost", "ssurt", "ssurtHost", "surt",
				"surtWithoutScheme", "surtWithoutTrailingComma",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			vectors := loadIIPCURLCanonVectors(t, test.file)
			got := make([]string, 0, len(vectors))
			for group := range vectors {
				got = append(got, group)
			}
			sort.Strings(got)
			sort.Strings(test.want)
			if !slices.Equal(got, test.want) {
				t.Fatalf("groups = %q, want %q", got, test.want)
			}
		})
	}
}
