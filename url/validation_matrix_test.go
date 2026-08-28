package url

import (
	"errors"
	"testing"
)

func TestWHATWGValidationErrorConformanceMatrix(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		base      string
		wantCodes []ValidationErrorCode
		wantURL   string
		wantOK    bool
	}{
		{
			name:      "domain-to-ASCII",
			input:     "https://exa%23mple.org",
			wantCodes: []ValidationErrorCode{CodeDomainPercentEncoded, CodeDomainToASCII},
		},
		{
			name:      "domain-percent-encoded",
			input:     "https://exam%70le.org",
			wantCodes: []ValidationErrorCode{CodeDomainPercentEncoded},
			wantURL:   "https://example.org/",
			wantOK:    true,
		},
		{
			name:      "host-invalid-code-point",
			input:     "foo://exa[mple.org",
			wantCodes: []ValidationErrorCode{CodeHostInvalidCodePoint},
		},
		{
			name:      "IPv4-empty-part",
			input:     "https://127.0.0.1./",
			wantCodes: []ValidationErrorCode{CodeIPv4EmptyPart},
			wantURL:   "https://127.0.0.1/",
			wantOK:    true,
		},
		{
			name:      "IPv4-too-few-parts",
			input:     "https://1.2.3/",
			wantCodes: []ValidationErrorCode{CodeIPv4TooFewParts},
			wantURL:   "https://1.2.0.3/",
			wantOK:    true,
		},
		{
			name:      "IPv4-too-many-parts",
			input:     "https://1.2.3.4.5/",
			wantCodes: []ValidationErrorCode{CodeIPv4TooManyParts},
		},
		{
			name:      "IPv4-non-numeric-part",
			input:     "https://test.42",
			wantCodes: []ValidationErrorCode{CodeIPv4TooFewParts, CodeIPv4NonNumericPart},
		},
		{
			name:      "IPv4-non-decimal-part",
			input:     "https://127.0.0x0.1",
			wantCodes: []ValidationErrorCode{CodeIPv4NonDecimalPart},
			wantURL:   "https://127.0.0.1/",
			wantOK:    true,
		},
		{
			name:      "IPv4-out-of-range-part",
			input:     "https://255.255.4000.1",
			wantCodes: []ValidationErrorCode{CodeIPv4OutOfRangePart},
		},
		{
			name:      "IPv4-non-ASCII-input",
			input:     "https://①.②.③.④",
			wantCodes: []ValidationErrorCode{CodeIPv4NonASCIIInput},
			wantURL:   "https://1.2.3.4/",
			wantOK:    true,
		},
		{
			name:      "IPv6-unclosed",
			input:     "https://[::1",
			wantCodes: []ValidationErrorCode{CodeIPv6Unclosed},
		},
		{
			name:      "IPv6-invalid-compression",
			input:     "https://[:1]",
			wantCodes: []ValidationErrorCode{CodeIPv6InvalidCompression},
		},
		{
			name:      "IPv6-too-many-pieces",
			input:     "https://[1:2:3:4:5:6:7:8:9]",
			wantCodes: []ValidationErrorCode{CodeIPv6TooManyPieces},
		},
		{
			name:      "IPv6-multiple-compression",
			input:     "https://[1::1::1]",
			wantCodes: []ValidationErrorCode{CodeIPv6MultipleCompression},
		},
		{
			name:      "IPv6-invalid-code-point",
			input:     "https://[1:2:3!:4]",
			wantCodes: []ValidationErrorCode{CodeIPv6InvalidCodePoint},
		},
		{
			name:      "IPv6-too-few-pieces",
			input:     "https://[1:2:3]",
			wantCodes: []ValidationErrorCode{CodeIPv6TooFewPieces},
		},
		{
			name:      "IPv6-piece-leading-zero",
			input:     "https://[::01]",
			wantCodes: []ValidationErrorCode{CodeIPv6PieceLeadingZero},
			wantURL:   "https://[::1]/",
			wantOK:    true,
		},
		{
			name:      "IPv4-in-IPv6-too-many-pieces",
			input:     "https://[1:1:1:1:1:1:1:127.0.0.1]",
			wantCodes: []ValidationErrorCode{CodeIPv4InIPv6TooManyPieces},
		},
		{
			name:      "IPv4-in-IPv6-invalid-code-point",
			input:     "https://[ffff::127.00.0.1]",
			wantCodes: []ValidationErrorCode{CodeIPv4InIPv6InvalidCodePoint},
		},
		{
			name:      "IPv4-in-IPv6-out-of-range-part",
			input:     "https://[ffff::127.0.0.4000]",
			wantCodes: []ValidationErrorCode{CodeIPv4InIPv6OutOfRangePart},
		},
		{
			name:      "IPv4-in-IPv6-too-few-parts",
			input:     "https://[ffff::127.0.0]",
			wantCodes: []ValidationErrorCode{CodeIPv4InIPv6TooFewParts},
		},
		{
			name:      "invalid-URL-unit",
			input:     "https://example.org/>",
			wantCodes: []ValidationErrorCode{CodeInvalidURLUnit},
			wantURL:   "https://example.org/%3E",
			wantOK:    true,
		},
		{
			name:      "special-scheme-missing-following-solidus",
			input:     "file:c:/my-secret-folder",
			wantCodes: []ValidationErrorCode{CodeSpecialSchemeMissingFollowingSolidus},
			wantURL:   "file:///c:/my-secret-folder",
			wantOK:    true,
		},
		{
			name:      "missing-scheme-non-relative-URL",
			input:     "💩",
			wantCodes: []ValidationErrorCode{CodeMissingSchemeNonRelativeURL},
		},
		{
			name:      "invalid-reverse-solidus",
			input:     `https://example.org\path\to\file`,
			wantCodes: []ValidationErrorCode{CodeInvalidReverseSolidus, CodeInvalidReverseSolidus, CodeInvalidReverseSolidus},
			wantURL:   "https://example.org/path/to/file",
			wantOK:    true,
		},
		{
			name:      "invalid-credentials",
			input:     "https://user@example.org",
			wantCodes: []ValidationErrorCode{CodeInvalidCredentials},
			wantURL:   "https://user@example.org/",
			wantOK:    true,
		},
		{
			name:      "host-missing",
			input:     "https://#fragment",
			wantCodes: []ValidationErrorCode{CodeHostMissing},
		},
		{
			name:      "port-out-of-range",
			input:     "https://example.org:70000",
			wantCodes: []ValidationErrorCode{CodePortOutOfRange},
		},
		{
			name:      "port-invalid",
			input:     "https://example.org:7z",
			wantCodes: []ValidationErrorCode{CodePortInvalid},
		},
		{
			name:      "file-invalid-Windows-drive-letter",
			input:     "c|/path/to/file",
			base:      "file:///c:/",
			wantCodes: []ValidationErrorCode{CodeFileInvalidWindowsDriveLetter, CodeInvalidURLUnit},
			wantURL:   "file:///c:/path/to/file",
			wantOK:    true,
		},
		{
			name:      "file-invalid-Windows-drive-letter-host",
			input:     "file://c:",
			wantCodes: []ValidationErrorCode{CodeFileInvalidWindowsDriveLetterHost},
			wantURL:   "file:///c:",
			wantOK:    true,
		},
	}

	if got, want := len(tests), 31; got != want {
		t.Fatalf("matrix contains %d cases, want %d", got, want)
	}

	covered := make(map[ValidationErrorCode]bool, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, code := range tt.wantCodes {
				covered[code] = true
			}

			var u *Url
			var err error
			if tt.base == "" {
				u, err = Parse(tt.input)
			} else {
				u, err = ParseRef(tt.base, tt.input)
			}

			if tt.wantOK {
				if err != nil {
					t.Fatalf("parse returned an unexpected error: %v", err)
				}
				if u == nil {
					t.Fatal("parse returned a nil URL on success")
				}
				if got := u.Href(false); got != tt.wantURL {
					t.Fatalf("serialized URL = %q, want %q", got, tt.wantURL)
				}
				requireValidationCodes(t, u.ValidationErrors(), tt.wantCodes...)
				return
			}

			if err == nil {
				t.Fatalf("parse returned URL %q, want failure", u)
			}
			if u != nil {
				t.Fatalf("parse returned partial URL %q on failure", u)
			}
			parseErr, ok := errors.AsType[*ParseError](err)
			if !ok {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			if !errors.Is(err, ErrParseFailure) {
				t.Fatalf("error = %v, want ErrParseFailure", err)
			}
			requireValidationCodes(t, parseErr.ValidationErrors(), tt.wantCodes...)
		})
	}

	for _, code := range []ValidationErrorCode{
		CodeDomainToASCII,
		CodeDomainPercentEncoded,
		CodeHostInvalidCodePoint,
		CodeIPv4EmptyPart,
		CodeIPv4TooFewParts,
		CodeIPv4TooManyParts,
		CodeIPv4NonNumericPart,
		CodeIPv4NonDecimalPart,
		CodeIPv4OutOfRangePart,
		CodeIPv4NonASCIIInput,
		CodeIPv6Unclosed,
		CodeIPv6InvalidCompression,
		CodeIPv6TooManyPieces,
		CodeIPv6MultipleCompression,
		CodeIPv6InvalidCodePoint,
		CodeIPv6TooFewPieces,
		CodeIPv6PieceLeadingZero,
		CodeIPv4InIPv6TooManyPieces,
		CodeIPv4InIPv6InvalidCodePoint,
		CodeIPv4InIPv6OutOfRangePart,
		CodeIPv4InIPv6TooFewParts,
		CodeInvalidURLUnit,
		CodeSpecialSchemeMissingFollowingSolidus,
		CodeMissingSchemeNonRelativeURL,
		CodeInvalidReverseSolidus,
		CodeInvalidCredentials,
		CodeHostMissing,
		CodePortOutOfRange,
		CodePortInvalid,
		CodeFileInvalidWindowsDriveLetter,
		CodeFileInvalidWindowsDriveLetterHost,
	} {
		if !covered[code] {
			t.Errorf("matrix does not cover validation error %q", code)
		}
	}
}
