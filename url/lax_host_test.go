package url

import (
	"errors"
	"testing"
)

func TestLaxHostParsingDomainRecovery(t *testing.T) {
	for _, tc := range []struct {
		input string
		codes []ValidationErrorCode
	}{
		{"http://HOST%23.Example/", []ValidationErrorCode{CodeDomainPercentEncoded, CodeDomainToASCII}},
		{"http://%20Example.com/", []ValidationErrorCode{CodeDomainPercentEncoded, CodeDomainToASCII}},
		{"http:// Example.com/", []ValidationErrorCode{CodeDomainToASCII}},
		{"http://%EF%BB%BF/", []ValidationErrorCode{CodeDomainPercentEncoded, CodeDomainToASCII}},
		// Exercise the general domain-parser path as well as the ASCII path.
		{"http://ä%23.Example/", []ValidationErrorCode{CodeDomainPercentEncoded, CodeDomainToASCII}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			for _, report := range []bool{true, false} {
				u, err := NewParser(WithLaxHostParsing(), WithReportValidationErrors(report)).Parse(tc.input)
				if err != nil {
					t.Fatal(err)
				}
				if u.String() != tc.input {
					t.Fatalf("recovery = %q, want original spelling %q", u, tc.input)
				}
				want := tc.codes
				if !report {
					want = nil
				}
				requireValidationCodes(t, u.ValidationErrors(), want...)

				u, err = NewParser(WithReportValidationErrors(report)).Parse(tc.input)
				parseErr, ok := errors.AsType[*ParseError](err)
				if u != nil || !ok {
					t.Fatalf("default parsing = %v, %v; want parse failure", u, err)
				}
				requireValidationCodes(t, parseErr.ValidationErrors(), tc.codes...)
			}

			for _, opts := range [][]ParserOption{
				{WithLaxHostParsing(), WithFailOnValidationError()},
				{WithFailOnValidationError(), WithLaxHostParsing(), WithReportValidationErrors(false)},
			} {
				u, err := NewParser(opts...).Parse(tc.input)
				parseErr, ok := errors.AsType[*ParseError](err)
				if u != nil || !ok {
					t.Fatalf("fail-fast parsing = %v, %v; want parse failure", u, err)
				}
				requireValidationCodes(t, parseErr.ValidationErrors(), tc.codes[0])
			}
		})
	}
}

func TestLaxHostParsingDoesNotMutateBorrowedInput(t *testing.T) {
	for _, host := range []string{"HOST%23.Example", " Example.COM"} {
		input := []byte(host)
		p := newDefaultParser()
		p.LaxHostParsing = true
		got, err := p.parseHost(&Url{}, input, false)
		if err != nil || got != host || string(input) != host {
			t.Fatalf("parseHost(%q) = %q, %v; input after parsing = %q", host, got, err, input)
		}
		for i := range input {
			input[i] = 'x'
		}
		if got != host {
			t.Fatalf("recovered host aliases input: %q", got)
		}
	}
}

func TestLaxHostParsingIPv4Recovery(t *testing.T) {
	for _, tc := range []struct {
		input, host string
		codes       []ValidationErrorCode
	}{
		{"http://09/", "09", []ValidationErrorCode{CodeIPv4TooFewParts, CodeIPv4NonNumericPart}},
		{"http://0xGG.1/", "0xgg.1", []ValidationErrorCode{CodeIPv4TooFewParts, CodeIPv4NonNumericPart}},
		{"http://1.2.3.999/", "1.2.3.999", []ValidationErrorCode{CodeIPv4OutOfRangePart}},
		{"http://256.1.1.1/", "256.1.1.1", []ValidationErrorCode{CodeIPv4OutOfRangePart}},
		{"http://1.2.3.4.5/", "1.2.3.4.5", []ValidationErrorCode{CodeIPv4TooManyParts}},
		{"http://example.42/", "example.42", []ValidationErrorCode{CodeIPv4TooFewParts, CodeIPv4NonNumericPart}},
		{"http://%30%39/", "09", []ValidationErrorCode{CodeDomainPercentEncoded, CodeIPv4TooFewParts, CodeIPv4NonNumericPart}},
		{"http://①.②.③.⑨⑨⑨/", "1.2.3.999", []ValidationErrorCode{CodeIPv4NonASCIIInput, CodeIPv4OutOfRangePart}},
		{"http://99999999999999999999999999999/", "99999999999999999999999999999", []ValidationErrorCode{CodeIPv4TooFewParts, CodeIPv4OutOfRangePart}},
		{"http://0xFFFFFFFFF/", "0xfffffffff", []ValidationErrorCode{CodeIPv4TooFewParts, CodeIPv4NonDecimalPart, CodeIPv4OutOfRangePart}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			for _, report := range []bool{true, false} {
				u, err := NewParser(WithLaxHostParsing(), WithReportValidationErrors(report)).Parse(tc.input)
				if err != nil || u.String() != "http://"+tc.host+"/" || u.IsIPv4() || u.IsIPv6() {
					t.Fatalf("recovery = %v, %v; want hostname %q without IP metadata", u, err, tc.host)
				}
				want := tc.codes
				if !report {
					want = nil
				}
				requireValidationCodes(t, u.ValidationErrors(), want...)
			}
			for _, options := range [][]ParserOption{
				nil,
				{WithReportValidationErrors(false)},
			} {
				u, err := NewParser(options...).Parse(tc.input)
				failure, ok := errors.AsType[*ParseError](err)
				if u != nil || !ok {
					t.Fatalf("without IPv4 recovery = %v, %v; want failure", u, err)
				}
				requireValidationCodes(t, failure.ValidationErrors(), tc.codes...)
			}
			for _, options := range [][]ParserOption{
				{WithLaxHostParsing(), WithFailOnValidationError()},
				{WithFailOnValidationError(), WithLaxHostParsing(), WithReportValidationErrors(false)},
			} {
				u, err := NewParser(options...).Parse(tc.input)
				failure, ok := errors.AsType[*ParseError](err)
				if u != nil || !ok {
					t.Fatalf("fail-fast parsing = %v, %v; want failure", u, err)
				}
				requireValidationCodes(t, failure.ValidationErrors(), tc.codes[0])
			}
		})
	}
}

func TestLaxHostParsingIPv4MetadataAndResolution(t *testing.T) {
	p := NewParser(WithLaxHostParsing())
	u, err := p.Parse("http://0x7f.1/")
	if err != nil || u.String() != "http://127.0.0.1/" || !u.IsIPv4() {
		t.Fatalf("valid IPv4 normalization = %v, %v", u, err)
	}
	u.SetHostname("09")
	if u.String() != "http://09/" || u.IsIPv4() || u.IsIPv6() {
		t.Fatalf("hostname setter retained stale IP metadata: %q", u)
	}
	u, err = p.ParseRef("http://09/dir/", "child")
	if err != nil || u.String() != "http://09/dir/child" || u.IsIPv4() {
		t.Fatalf("relative resolution = %v, %v", u, err)
	}
}
