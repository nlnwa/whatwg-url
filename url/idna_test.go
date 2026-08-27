package url

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

type idnaTest struct {
	Input           string  `json:"input"`
	Output          *string `json:"output"`
	Comment         string  `json:"comment"`
	UrlStandardOnly bool    `json:"urlStandardOnly"`
}

func loadIdnaTestData(filename string) ([]idnaTest, error) {
	var tests []idnaTest

	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = jsonFile.Close() }()
	data, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(data, &tests)

	var empty idnaTest
	n := 0
	for _, val := range tests {
		if val == empty {
			continue
		}
		tests[n] = val
		n++
	}
	return tests[:n], nil
}

// TestToASCII tests the ToASCII function using the test cases in testdata/toascii.json.
// The test cases are copied from the WHATWG URL Standard test suite.
func TestToASCII(t *testing.T) {
	tests, err := loadIdnaTestData("../testdata/toascii.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%03d %s", i, tt.Comment), func(t *testing.T) {
			testUrl := fmt.Sprintf("https://%s/x", tt.Input)
			u, err := testParser.Parse(testUrl)

			if tt.Output == nil {
				if err == nil {
					t.Fatalf("ToASCII(%v) = '%v', expected error but got none", tt.Input, u.Hostname())
				}
				return
			}

			if err != nil {
				t.Fatalf("ToASCII(%v) = '%v', unexpected error: %v", tt.Input, u.Hostname(), err)
			}

			gotHostname := u.Hostname()
			gotHost := u.Host()
			gotPathname := u.Pathname()
			gotHref := u.String()

			wantPathname := "/x"
			wantHref := fmt.Sprintf("https://%s/x", *tt.Output)

			want := *tt.Output

			if gotHost != want {
				t.Errorf("ToASCII(%v) host is '%v', want '%s'", tt.Input, gotHostname, want)
			}
			if gotHostname != want {
				t.Errorf("ToASCII(%v) hostname is '%v', want '%s'", tt.Input, gotHostname, want)
			}
			if gotPathname != wantPathname {
				t.Errorf("ToASCII(%v) pathname is '%v', want '%s'", tt.Input, gotPathname, wantPathname)
			}
			if gotHref != wantHref {
				t.Errorf("ToASCII(%v) href is '%v', want '%s'", tt.Input, gotHref, wantHref)
			}
		})
	}
}

// TestIdna tests the ToASCII function using the test cases in testdata/IdnaTestV2.json.
// The test cases are copied from the WHATWG URL Standard test suite.
func TestIdnaV2(t *testing.T) {
	tests, err := loadIdnaTestData("../testdata/IdnaTestV2.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%03d: %s", i, tt.Comment), func(t *testing.T) {
			got, err := domainToASCII(tt.Input, false)
			wantErr := tt.Output == nil
			if wantErr && err != nil {
				t.Logf("ToASCII(%v), got an expected error: %v", tt.Input, err)
				return
			}
			if err != nil && !wantErr {
				t.Fatalf("ToASCII(%v) = '%v', unexpected error: %v", tt.Input, got, err)
			}
			if err == nil && wantErr {
				t.Fatalf("ToASCII(%v) = '%v', expected error but got none", tt.Input, got)
			}

			want := *tt.Output
			if got != want {
				t.Errorf("ToASCII(%v) = '%v', want '%s'", tt.Input, got, want)
			}
		})
	}
}
