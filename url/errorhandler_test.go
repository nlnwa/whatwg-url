package url

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationErrorCodes(t *testing.T) {
	tests := []struct {
		code ValidationErrorCode
		err  *ValidationError
	}{
		{CodeDomainToASCII, ErrDomainToASCII},
		{CodeDomainPercentEncoded, ErrDomainPercentEncoded},
		{CodeHostInvalidCodePoint, ErrHostInvalidCodePoint},
		{CodeIPv4EmptyPart, ErrIPv4EmptyPart},
		{CodeIPv4TooFewParts, ErrIPv4TooFewParts},
		{CodeIPv4TooManyParts, ErrIPv4TooManyParts},
		{CodeIPv4NonNumericPart, ErrIPv4NonNumericPart},
		{CodeIPv4NonDecimalPart, ErrIPv4NonDecimalPart},
		{CodeIPv4OutOfRangePart, ErrIPv4OutOfRangePart},
		{CodeIPv4NonASCIIInput, ErrIPv4NonASCIIInput},
		{CodeIPv6Unclosed, ErrIPv6Unclosed},
		{CodeIPv6InvalidCompression, ErrIPv6InvalidCompression},
		{CodeIPv6TooManyPieces, ErrIPv6TooManyPieces},
		{CodeIPv6MultipleCompression, ErrIPv6MultipleCompression},
		{CodeIPv6InvalidCodePoint, ErrIPv6InvalidCodePoint},
		{CodeIPv6TooFewPieces, ErrIPv6TooFewPieces},
		{CodeIPv6PieceLeadingZero, ErrIPv6PieceLeadingZero},
		{CodeIPv4InIPv6TooManyPieces, ErrIPv4InIPv6TooManyPieces},
		{CodeIPv4InIPv6InvalidCodePoint, ErrIPv4InIPv6InvalidCodePoint},
		{CodeIPv4InIPv6OutOfRangePart, ErrIPv4InIPv6OutOfRangePart},
		{CodeIPv4InIPv6TooFewParts, ErrIPv4InIPv6TooFewParts},
		{CodeInvalidURLUnit, ErrInvalidURLUnit},
		{CodeSpecialSchemeMissingFollowingSolidus, ErrSpecialSchemeMissingFollowingSolidus},
		{CodeMissingSchemeNonRelativeURL, ErrMissingSchemeNonRelativeURL},
		{CodeInvalidReverseSolidus, ErrInvalidReverseSolidus},
		{CodeInvalidCredentials, ErrInvalidCredentials},
		{CodeHostMissing, ErrHostMissing},
		{CodePortOutOfRange, ErrPortOutOfRange},
		{CodePortInvalid, ErrPortInvalid},
		{CodeFileInvalidWindowsDriveLetter, ErrFileInvalidWindowsDriveLetter},
		{CodeFileInvalidWindowsDriveLetterHost, ErrFileInvalidWindowsDriveLetterHost},
	}

	if len(tests) != 31 {
		t.Fatalf("got %d validation errors, want 31", len(tests))
	}
	seen := make(map[ValidationErrorCode]bool, len(tests))
	for _, tt := range tests {
		if tt.err.Code() != tt.code {
			t.Errorf("%T.Code() = %q, want %q", tt.err, tt.err.Code(), tt.code)
		}
		if tt.err.Detail() == "" || tt.err.Error() == "" {
			t.Errorf("validation error %q has no detail", tt.code)
		}
		if seen[tt.code] {
			t.Errorf("duplicate validation error code %q", tt.code)
		}
		seen[tt.code] = true
	}
}

func TestValidationError_ZeroValueMatchesNoSentinel(t *testing.T) {
	var zero ValidationError
	if zero.Code() != "" || zero.Detail() != "" || zero.Error() != "" || zero.Unwrap() != nil {
		t.Fatalf("unexpected non-empty zero value: %#v", zero)
	}
	if errors.Is(&zero, ErrInvalidURLUnit) || errors.Is(ErrInvalidURLUnit, &zero) {
		t.Fatal("zero-value ValidationError matched a sentinel")
	}
}

func TestValidationError_UnwrapsGenuineCause(t *testing.T) {
	cause := errors.New("IDNA failed")
	verr := validationErrorWithCause(ErrDomainToASCII, cause)
	if !errors.Is(verr, ErrDomainToASCII) {
		t.Fatal("wrapped occurrence does not match its sentinel")
	}
	if !errors.Is(verr, cause) {
		t.Fatal("wrapped occurrence does not expose its cause")
	}
	if got, want := verr.Detail(), ErrDomainToASCII.Detail(); got != want {
		t.Fatalf("Detail() = %q, want %q", got, want)
	}
}

func TestParseFailuresUseParseError(t *testing.T) {
	p := NewParser()
	bp := p.(BasicParsing)

	tests := []struct {
		name                 string
		call                 func() error
		wantValidationErrors int
	}{
		{
			name: "Parse",
			call: func() error {
				_, err := p.Parse("https://example.org:7z")
				return err
			},
			wantValidationErrors: 1,
		},
		{
			name: "ParseRef",
			call: func() error {
				_, err := p.ParseRef("https://example.org/", "https://example.org:7z")
				return err
			},
			wantValidationErrors: 1,
		},
		{
			name: "BasicParse",
			call: func() error {
				_, err := bp.BasicParse("1http", nil, newUrl(), StateSchemeStart)
				return err
			},
			wantValidationErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			if !errors.Is(err, ErrParseFailure) {
				t.Fatal("parser error does not match ErrParseFailure")
			}
			if got := len(parseErr.ValidationErrors()); got != tt.wantValidationErrors {
				t.Fatalf("len(ValidationErrors()) = %d, want %d", got, tt.wantValidationErrors)
			}
		})
	}
}

func TestParseError_MultiUnwrapAndDefensiveValidationErrors(t *testing.T) {
	_, err := Parse("https://example.org:7z")
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	if !errors.Is(err, ErrParseFailure) || !errors.Is(err, ErrPortInvalid) {
		t.Fatalf("error does not expose both failure and validation identities: %v", err)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.Code() != CodePortInvalid {
		t.Fatalf("error does not expose its validation occurrence through errors.As: %v", err)
	}

	errs := parseErr.ValidationErrors()
	errs[0] = nil
	if got := parseErr.ValidationErrors(); len(got) != 1 || got[0] == nil {
		t.Fatalf("ValidationErrors() was mutated through returned slice: %v", got)
	}

	var zero ParseError
	if !errors.Is(&zero, ErrParseFailure) {
		t.Fatal("zero-value ParseError does not match ErrParseFailure")
	}
}

func TestParseError_IncludesValidationErrorWhenReportingIsDisabled(t *testing.T) {
	p := NewParser(WithReportValidationErrors(false))
	_, err := p.Parse("https://example.org:7z")
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	if !errors.Is(parseErr, ErrPortInvalid) {
		t.Fatalf("error = %v, want %v", parseErr, ErrPortInvalid)
	}
	if got := parseErr.ValidationErrors(); len(got) != 1 || got[0].Code() != CodePortInvalid {
		t.Fatalf("ValidationErrors() = %v, want one port-invalid occurrence", got)
	}
}

func TestParseError_CanRepresentBareFailure(t *testing.T) {
	p := NewParser().(BasicParsing)
	_, err := p.BasicParse("1http", nil, newUrl(), StateSchemeStart)
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	if got := parseErr.ValidationErrors(); len(got) != 0 {
		t.Fatalf("bare failure contains validation errors: %v", got)
	}
	if !errors.Is(err, ErrParseFailure) {
		t.Fatal("bare failure does not match ErrParseFailure")
	}
}

func TestUrlValidationErrors_DefensiveAndPreservedByClone(t *testing.T) {
	u, err := Parse(" https://example.org ")
	if err != nil {
		t.Fatal(err)
	}

	errs := u.ValidationErrors()
	if len(errs) == 0 {
		t.Fatal("expected validation history")
	}
	errs[0] = nil
	if got := u.ValidationErrors(); got[0] == nil {
		t.Fatal("URL validation history was mutated through returned slice")
	}

	clone := u.Clone()
	cloneErrors := clone.ValidationErrors()
	if len(cloneErrors) != len(u.ValidationErrors()) {
		t.Fatalf("clone validation history length = %d, want %d", len(cloneErrors), len(u.ValidationErrors()))
	}
	cloneErrors[0] = nil
	if clone.ValidationErrors()[0] == nil || u.ValidationErrors()[0] == nil {
		t.Fatal("clone validation history was not defensively copied")
	}
}

func TestValidationErrorCodeStrings(t *testing.T) {
	if got, want := fmt.Sprint(CodeDomainToASCII), "domain-to-ASCII"; got != want {
		t.Fatalf("domain code = %q, want %q", got, want)
	}
	if got, want := fmt.Sprint(CodeFileInvalidWindowsDriveLetterHost), "file-invalid-Windows-drive-letter-host"; got != want {
		t.Fatalf("file host code = %q, want %q", got, want)
	}
}

func TestParse_RecordsValidationWarning(t *testing.T) {
	p := NewParser()

	u, err := p.Parse(" https://example.org ")
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("Parse returned nil URL")
	}

	verrs := u.ValidationErrors()
	if len(verrs) == 0 {
		t.Fatal("expected validation errors to be recorded")
	}

	var found bool
	for _, verr := range verrs {
		if errors.Is(verr, ErrInvalidURLUnit) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %v validation error to be recorded", ErrInvalidURLUnit)
	}
}

func TestBasicParse_BareStateOverrideFailure(t *testing.T) {
	p := NewParser()
	bp, ok := p.(BasicParsing)
	if !ok {
		t.Fatal("parser does not implement BasicParsing")
	}
	u := newUrl()
	ru, err := bp.BasicParse("1http", nil, u, StateSchemeStart)
	if err == nil {
		t.Fatal("expected parse failure")
	}
	if ru != nil {
		t.Fatal("expected nil URL on parse failure")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	if got := parseErr.ValidationErrors(); len(got) != 0 {
		t.Fatalf("bare failure contains validation errors: %v", got)
	}

	if len(u.ValidationErrors()) != 0 {
		t.Fatal("expected explicit failure path to not record validation error on URL")
	}
}

func TestParseError_ContainsAllEventsWhenReportingIsDisabled(t *testing.T) {
	p := NewParser(WithReportValidationErrors(false))
	tests := []struct {
		name      string
		input     string
		wantCodes []ValidationErrorCode
	}{
		{"preprocessing", " https://example.org:7z ", []ValidationErrorCode{CodeInvalidURLUnit, CodePortInvalid}},
		{"strict IDNA check", "https://a_b.example:7z", []ValidationErrorCode{CodeDomainToASCII, CodePortInvalid}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := p.Parse(tt.input)
			if u != nil {
				t.Fatalf("Parse returned partial URL %q on failure", u)
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			errs := parseErr.ValidationErrors()
			if len(errs) != len(tt.wantCodes) {
				t.Fatalf("ValidationErrors() = %v, want codes %v", errs, tt.wantCodes)
			}
			for i, code := range tt.wantCodes {
				if errs[i].Code() != code {
					t.Fatalf("ValidationErrors()[%d].Code() = %q, want %q", i, errs[i].Code(), code)
				}
			}
		})
	}
}

func TestBasicParse_AppendsEventsToExistingURLOnFailure(t *testing.T) {
	p := NewParser()
	bp := p.(BasicParsing)
	u := newUrl()

	// ASCII tab removal is a validation event; the resulting leading digit is
	// then a bare scheme-state-override failure.
	_, err := bp.BasicParse("1\thttp", nil, u, StateSchemeStart)
	if err == nil {
		t.Fatal("expected parse failure")
	}
	urlErrors := u.ValidationErrors()
	if len(urlErrors) != 1 || urlErrors[0].Code() != CodeInvalidURLUnit {
		t.Fatalf("existing URL validation history = %v, want one invalid-URL-unit occurrence", urlErrors)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T, want *ParseError", err)
	}
	failureErrors := parseErr.ValidationErrors()
	if len(failureErrors) != 1 || failureErrors[0].Code() != CodeInvalidURLUnit {
		t.Fatalf("failure validation history = %v, want the same detected occurrence", failureErrors)
	}
	if urlErrors[0] != failureErrors[0] {
		t.Fatal("URL and ParseError do not reference the same validation occurrence")
	}
}

func TestParseAndParseRef_DoNotExposePartialURL(t *testing.T) {
	p := NewParser()
	for name, call := range map[string]func() (*Url, error){
		"Parse": func() (*Url, error) {
			return p.Parse("https://example^example")
		},
		"ParseRef": func() (*Url, error) {
			return p.ParseRef("https://example.org/", "https://example^example")
		},
	} {
		t.Run(name, func(t *testing.T) {
			u, err := call()
			if err == nil {
				t.Fatal("expected parse failure")
			}
			if u != nil {
				t.Fatalf("returned partial URL %q", u)
			}
			if !errors.Is(err, ErrParseFailure) {
				t.Fatalf("error = %v, want ErrParseFailure", err)
			}
		})
	}
}
