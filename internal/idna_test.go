package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"testing"

	"golang.org/x/net/idna"
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

// runSkipped is a flag to control whether to run tests that are normally skipped
var runSkipped bool

func TestMain(m *testing.M) {
	flag.BoolVar(&runSkipped, "runSkipped", false, "run tests that are normally skipped (e.g. known failures)")
	flag.Parse()
	os.Exit(m.Run())
}

// TestToASCII tests the ToASCII function using the test cases in testdata/toascii.json.
// The test cases are copied from the WHATWG URL Standard test suite.
func TestToASCII(t *testing.T) {
	tests, err := loadIdnaTestData("../testdata/toascii.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	var skipUnicodeVersion = "15.0.0"
	var shouldSkipKnownToFail = idna.UnicodeVersion == skipUnicodeVersion && !runSkipped
	var knownToFail = []int{59, 67, 74, 75, 76, 80, 81}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%03d: %s", i, tt.Comment), func(t *testing.T) {
			got, err := domainToASCII(tt.Input, false)
			if shouldSkipKnownToFail && slices.Contains(knownToFail, i) {
				t.Skipf("ToASCII(%v) = '%v', skipping test because golang.org/x/net/idna package unicode version is %s", tt.Input, got, skipUnicodeVersion)
			}

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

// TestIdna tests the ToASCII function using the test cases in testdata/IdnaTestV2.json.
// The test cases are copied from the WHATWG URL Standard test suite.
func TestIdnaV2(t *testing.T) {
	tests, err := loadIdnaTestData("../testdata/IdnaTestV2.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// skip tests that fail due to the golang.org/x/net/idna package not supporting the latest Unicode version
	var skipUnicodeVersion = "15.0.0"
	var shouldSkipKnownToFail = idna.UnicodeVersion == skipUnicodeVersion && !runSkipped
	var knownToFailUnicode = []int{
		64, 65, 355, 396, 397, 523, 524, 526, 530, 533, 536, 659, 660, 662,
		664, 667, 668, 672, 674, 949, 950, 954, 1070, 1071, 1170, 1171, 1278,
		1279, 1280, 1281, 1336, 1337, 1378, 1379, 1395, 1396, 1557, 1559, 1561,
		1712, 1713, 1758, 1759, 1771, 1773, 1778, 1791, 1792, 1823, 1826, 1827,
		1830, 1831, 1834, 1835, 1839, 1840, 1841, 1842, 1912, 1913, 1985, 1987,
		1989, 1991, 2027, 2029, 2333, 2334, 2335, 2336, 2431, 2433, 2524, 2525,
		2526, 2527, 2573, 2575, 2577, 2579, 2580, 2581, 2582, 2583,
	}

	// skip tests that fail due to the golang.org/x/net/idna package not implementing all validation rules
	var shouldSkipKnownToFailValidation = !runSkipped
	var knownToFailValidation = []int{147, 250, 351, 353, 1824, 1911}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%03d: %s", i, tt.Comment), func(t *testing.T) {
			got, err := domainToASCII(tt.Input, false)
			if shouldSkipKnownToFail && slices.Contains(knownToFailUnicode, i) {
				t.Skipf("ToASCII(%v) = '%v', skipping test because the golang.org/x/net/idna package unicode version is %s", tt.Input, got, skipUnicodeVersion)
			}
			if shouldSkipKnownToFailValidation && slices.Contains(knownToFailValidation, i) {
				t.Skipf("ToASCII(%v) = '%v', skipping test because the golang.org/x/net/idna package does not implement strict validation", tt.Input, got)
			}

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

func TestContainsOnlyASCIIOrMiscAndNoPunycode(t *testing.T) {
	tests := []struct {
		Input  string
		Output bool
	}{
		{"abc", true},
		{"xn--abc", false},
		{"abcxn--", true},
		{"abc.xn--", false},
		{"xnabc--", true},
		{"xn.--", true},
		{"ab\u2260c", true},
		{"ab\u2261c", false},
	}
	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			got := containsOnlyASCIIOrMiscAndNoPunycode(tt.Input)

			if got != tt.Output {
				t.Errorf("containsOnlyASCIIOrMiscAndNoPunycode(%v) = '%v', want '%v'", tt.Input, got, tt.Output)
			}
		})
	}
}
