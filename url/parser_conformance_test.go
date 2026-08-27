package url

import (
	"errors"
	"strings"
	"testing"
)

func validationCodes(errs []*ValidationError) []ValidationErrorCode {
	codes := make([]ValidationErrorCode, len(errs))
	for i, err := range errs {
		codes[i] = err.Code()
	}
	return codes
}

func requireValidationCodes(t *testing.T, got []*ValidationError, want ...ValidationErrorCode) {
	t.Helper()
	gotCodes := validationCodes(got)
	if len(gotCodes) != len(want) {
		t.Fatalf("validation codes = %v, want %v", gotCodes, want)
	}
	for i := range want {
		if gotCodes[i] != want[i] {
			t.Fatalf("validation codes = %v, want %v", gotCodes, want)
		}
	}
}

func TestIPv4ValidationAndRangeFailures(t *testing.T) {
	t.Run("too few parts", func(t *testing.T) {
		u, err := Parse("https://1.2.3/")
		if err != nil {
			t.Fatal(err)
		}
		if got, want := u.Href(false), "https://1.2.0.3/"; got != want {
			t.Fatalf("Href() = %q, want %q", got, want)
		}
		requireValidationCodes(t, u.ValidationErrors(), CodeIPv4TooFewParts)
	})

	t.Run("trailing empty part is removed first", func(t *testing.T) {
		u, err := Parse("https://1.2.3./")
		if err != nil {
			t.Fatal(err)
		}
		requireValidationCodes(t, u.ValidationErrors(), CodeDomainToASCII, CodeIPv4EmptyPart, CodeIPv4TooFewParts)
	})

	t.Run("range event occurs once", func(t *testing.T) {
		u, err := Parse("https://256.257.1.1/")
		if err == nil || u != nil {
			t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
		}
		var parseErr *ParseError
		if !errors.As(err, &parseErr) {
			t.Fatalf("error = %T, want *ParseError", err)
		}
		requireValidationCodes(t, parseErr.ValidationErrors(), CodeIPv4OutOfRangePart)
	})

	t.Run("machine integer overflow is range not syntax", func(t *testing.T) {
		u, err := Parse("https://" + strings.Repeat("9", 30) + "/")
		if err == nil || u != nil {
			t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
		}
		var parseErr *ParseError
		if !errors.As(err, &parseErr) {
			t.Fatalf("error = %T, want *ParseError", err)
		}
		requireValidationCodes(t, parseErr.ValidationErrors(), CodeIPv4TooFewParts, CodeIPv4OutOfRangePart)
		if errors.Is(err, ErrIPv4NonNumericPart) {
			t.Fatal("numeric overflow was classified as a non-numeric IPv4 part")
		}
	})
}

func TestIPv6ClassificationsAndLeadingZeros(t *testing.T) {
	for _, tt := range []struct {
		input string
		code  ValidationErrorCode
	}{
		{"https://[:]/", CodeIPv6InvalidCompression},
		{"https://[g]/", CodeIPv6InvalidCodePoint},
		{"https://[1]/", CodeIPv6TooFewPieces},
	} {
		t.Run(tt.input, func(t *testing.T) {
			u, err := Parse(tt.input)
			if err == nil || u != nil {
				t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			requireValidationCodes(t, parseErr.ValidationErrors(), tt.code)
		})
	}

	u, err := Parse("https://[::01]/")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Href(false), "https://[::1]/"; got != want {
		t.Fatalf("Href() = %q, want %q", got, want)
	}
	requireValidationCodes(t, u.ValidationErrors(), CodeIPv6PieceLeadingZero)

	u, err = Parse("https://[01::0002]/")
	if err != nil {
		t.Fatal(err)
	}
	requireValidationCodes(t, u.ValidationErrors(), CodeIPv6PieceLeadingZero, CodeIPv6PieceLeadingZero)
}

func TestOpaquePathReportsEveryInternalSpace(t *testing.T) {
	u, err := Parse("opaque:a b c")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Href(false), "opaque:a b c"; got != want {
		t.Fatalf("Href() = %q, want %q", got, want)
	}
	requireValidationCodes(t, u.ValidationErrors(), CodeInvalidURLUnit, CodeInvalidURLUnit)
}

func TestIDNAFailureHasOneOccurrenceWithCause(t *testing.T) {
	for _, input := range []string{
		"https://ä.\u200d.example/", // strict and relaxed conversion both fail
		"https://example^example/",  // fast-path forbidden-code-point failure
	} {
		t.Run(input, func(t *testing.T) {
			u, err := Parse(input)
			if err == nil || u != nil {
				t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			verrs := parseErr.ValidationErrors()
			requireValidationCodes(t, verrs, CodeDomainToASCII)
			if verrs[0].Unwrap() == nil {
				t.Fatal("domain-to-ASCII occurrence did not preserve the strict IDNA cause")
			}
		})
	}
}

func TestInvalidUTF8DomainFailsDuringDomainToASCII(t *testing.T) {
	u, err := Parse("https://example.com%80/")
	if err == nil || u != nil {
		t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	requireValidationCodes(
		t,
		parseErr.ValidationErrors(),
		CodeDomainPercentEncoded,
		CodeDomainToASCII,
	)
}

func TestEmptyIDNAOutputIsBareFailure(t *testing.T) {
	u, err := Parse("https://\ufeff/")
	if err == nil || u != nil {
		t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	// Strict processing still reports its one standard occurrence. The empty
	// output failure itself must not fabricate a second occurrence.
	requireValidationCodes(t, parseErr.ValidationErrors(), CodeDomainToASCII)
}

func TestStateOverrideFailuresAreBare(t *testing.T) {
	p := NewParser()
	bp := p.(BasicParsing)
	internalParser := p.(*parser)

	tests := []struct {
		name  string
		input string
		state State
		url   func(t *testing.T) *Url
	}{
		{"scheme start", "1http:", StateSchemeStart, func(t *testing.T) *Url { return internalParser.NewUrl() }},
		{"scheme", "!", StateScheme, func(t *testing.T) *Url { return internalParser.NewUrl() }},
		{"hostname", "other.example:80", StateHostname, func(t *testing.T) *Url {
			u, err := p.Parse("https://example.test/")
			if err != nil {
				t.Fatal(err)
			}
			return u
		}},
		{"host", "", StateHost, func(t *testing.T) *Url {
			u, err := p.Parse("opaque://user@example.test:99/path")
			if err != nil {
				t.Fatal(err)
			}
			return u
		}},
		{"empty port", "", StatePort, func(t *testing.T) *Url {
			u, err := p.Parse("https://example.test:99/")
			if err != nil {
				t.Fatal(err)
			}
			return u
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := tt.url(t)
			got, err := bp.BasicParse(tt.input, nil, u, tt.state)
			if err == nil || got != nil {
				t.Fatalf("BasicParse() = (%v, %v), want nil URL and failure", got, err)
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			if len(parseErr.ValidationErrors()) != 0 {
				t.Fatalf("bare failure contains validation errors: %v", parseErr.ValidationErrors())
			}
		})
	}
}
