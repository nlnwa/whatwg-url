package url

import (
	"errors"
	"testing"
)

func TestLaxHostParsingIPv6Recovery(t *testing.T) {
	for _, tc := range []struct {
		input string
		codes []ValidationErrorCode
	}{
		{"http://[::1/", []ValidationErrorCode{CodeIPv6Unclosed}},
		{"custom://[bogus]/", []ValidationErrorCode{CodeIPv6InvalidCodePoint}},
		{"http://[BOGUS]/", []ValidationErrorCode{CodeIPv6InvalidCodePoint}},
		{"http://[1::2::3]:81/path?x#y", []ValidationErrorCode{CodeIPv6MultipleCompression}},
		{"http://[1:2]/", []ValidationErrorCode{CodeIPv6TooFewPieces}},
		{"http://[:1]/", []ValidationErrorCode{CodeIPv6InvalidCompression}},
		{"http://[1:2:3:4:5:6:7:8:9]/", []ValidationErrorCode{CodeIPv6TooManyPieces}},
		{"http://[::ffff:256.1.1.1]/", []ValidationErrorCode{CodeIPv4InIPv6OutOfRangePart}},
		{"http://[01::2::3]/", []ValidationErrorCode{CodeIPv6PieceLeadingZero, CodeIPv6MultipleCompression}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			for _, report := range []bool{true, false} {
				u, err := NewParser(WithLaxHostParsing(), WithReportValidationErrors(report)).Parse(tc.input)
				if err != nil || u.String() != tc.input || u.IsIPv6() || u.IsIPv4() {
					t.Fatalf("lax parse = %v, %v; want original spelling without IP flags", u, err)
				}
				want := tc.codes
				if !report {
					want = nil
				}
				requireValidationCodes(t, u.ValidationErrors(), want...)
				for _, candidate := range []*Url{u, u.Clone()} {
					candidate.RewriteHostname(func(_ *Url, host string) string { return host })
					if candidate.String() != tc.input || candidate.IsIPv6() || candidate.IsIPv4() {
						t.Fatalf("clone/rewrite misclassified recovered host: %v", candidate)
					}
					requireValidationCodes(t, candidate.ValidationErrors(), want...)
				}
				_, err = NewParser(WithReportValidationErrors(report)).Parse(tc.input)
				failure, ok := errors.AsType[*ParseError](err)
				if !ok {
					t.Fatalf("strict parse = %v; want ParseError", err)
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
					t.Fatalf("fail-fast parse = %v, %v", u, err)
				}
				requireValidationCodes(t, failure.ValidationErrors(), tc.codes[0])
			}
		})
	}
}

func TestLaxIPv6MetadataTransitions(t *testing.T) {
	p := NewParser(WithLaxHostParsing())
	u, err := p.Parse("http://[2001:0DB8::1]/base/")
	if err != nil || u.Hostname() != "[2001:db8::1]" || !u.IsIPv6() {
		t.Fatalf("valid IPv6 = %v, %v", u, err)
	}
	for _, tc := range []struct {
		host       string
		ipv4, ipv6 bool
	}{
		{"[1::2::3]", false, false},
		{"[::1]", false, true},
		{"example.org", false, false},
		{"127.0.0.1", true, false},
		{"[unclosed", false, false},
	} {
		for _, setter := range []func(string){u.SetHostname, u.SetHost} {
			setter(tc.host)
			if u.Hostname() != tc.host || u.IsIPv4() != tc.ipv4 || u.IsIPv6() != tc.ipv6 {
				t.Fatalf("setter(%q) = %v, IPv4=%v, IPv6=%v", tc.host, u, u.IsIPv4(), u.IsIPv6())
			}
		}
		resolved, err := p.ParseRef(u.String(), "child")
		if err != nil || resolved.Hostname() != tc.host || resolved.IsIPv4() != tc.ipv4 || resolved.IsIPv6() != tc.ipv6 {
			t.Fatalf("relative resolution = %v, %v", resolved, err)
		}
	}
	u, err = Parse("http://[::1]/")
	if err != nil {
		t.Fatal(err)
	}
	u.SetHostname("[bad]")
	if u.Hostname() != "[::1]" || !u.IsIPv6() {
		t.Fatalf("failed setter changed host or metadata: %v", u)
	}
	u.RewriteHostname(func(_ *Url, _ string) string { return "[bad]" })
	if u.IsIPv6() {
		t.Fatal("brackets alone classified as IPv6")
	}
}
