package url

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

// TestToASCII tests the ToASCII function using the test cases in testdata/toascii.json.
// The test cases are copied from the WHATWG URL Standard test suite.
func TestToASCII(t *testing.T) {
	type args struct {
		Input   string
		Output  *string
		Comment string
	}

	var tests []args

	jsonFile, err := os.Open("../testdata/toascii.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := io.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	empty := args{}
	var testNum int
	for _, tt := range tests {
		if tt != empty {
			testNum++

			t.Run(fmt.Sprintf("%d: %s", testNum, tt.Comment), func(t *testing.T) {
				got, err := defaultParser.(*parser).ToASCII(tt.Input, false)

				if tt.Output != nil {
					if err != nil {
						t.Errorf("\nInput '%s'\nOutput '%s'\ngot '%s'\nerr %v\n\n", tt.Input, *tt.Output, got, err)
						t.Errorf("ToASCII(%v) = '%v', error = '%v', wantErr %v", tt.Input, got, err, tt.Output == nil)
						return
					}

					if got != *tt.Output {
						t.Errorf("ToASCII(%v) = '%v', want '%s'", tt.Input, got, *tt.Output)
					}
				} else {
					if err == nil {
						t.Errorf("ToASCII(%v) = '%v', error = '%v', wantErr %v", tt.Input, got, err, tt.Output == nil)
					}
				}
			})
		}
	}
}

// TestToASCII tests the ToASCII function using the test cases in testdata/toascii.json.
// The test cases are copied from the WHATWG URL Standard test suite.
func TestIdnaV2(t *testing.T) {
	type args struct {
		Input   string
		Output  *string
		Comment string
	}

	var tests []args

	jsonFile, err := os.Open("../testdata/IdnaTestV2.json")
	if err != nil {
		t.Error(err)
	}
	defer jsonFile.Close()
	data, _ := io.ReadAll(jsonFile)
	json.Unmarshal(data, &tests)

	empty := args{}
	var testNum int
	for _, tt := range tests {
		if tt != empty {
			testNum++

			t.Run(fmt.Sprintf("%d: %s", testNum, tt.Comment), func(t *testing.T) {
				got, err := defaultParser.(*parser).ToASCII(tt.Input, false)

				if tt.Output != nil {
					if err != nil {
						t.Errorf("\nInput '%s'\nOutput '%s'\ngot '%s'\nerr %v\n\n", tt.Input, *tt.Output, got, err)
						t.Errorf("ToASCII(%v) = '%v', error = '%v', wantErr %v", tt.Input, got, err, tt.Output == nil)
						return
					}

					if got != *tt.Output {
						t.Errorf("ToASCII(%v) = '%v', want '%s'", tt.Input, got, *tt.Output)
					}
				} else {
					if err == nil {
						t.Errorf("ToASCII(%+q) = '%v', error = '%v', wantErr %v", tt.Input, got, err, tt.Output == nil)
					}
				}
			})
		}
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
