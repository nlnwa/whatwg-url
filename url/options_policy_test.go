package url

import (
	"errors"
	"testing"
)

func TestValidationPolicy_DefaultAndReporting(t *testing.T) {
	const input = " https://example.org "

	tests := []struct {
		name        string
		options     []ParserOption
		wantHistory bool
	}{
		{name: "default", wantHistory: true},
		{
			name:    "reporting disabled",
			options: []ParserOption{WithReportValidationErrors(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewParser(tt.options...).Parse(input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got, want := u.String(), "https://example.org/"; got != want {
				t.Fatalf("Parse() = %q, want %q", got, want)
			}

			errs := u.ValidationErrors()
			if tt.wantHistory {
				requireValidationCodes(t, errs, CodeInvalidURLUnit)
			} else if len(errs) != 0 {
				t.Fatalf("ValidationErrors() = %v, want none", errs)
			}
		})
	}
}

func TestValidationPolicy_FailOnFirstOccurrence(t *testing.T) {
	p := NewParser(
		WithReportValidationErrors(false),
		WithFailOnValidationError(),
	)

	u, err := p.Parse(" https://example.org:7z ")
	if u != nil {
		t.Fatalf("Parse() URL = %q, want nil", u)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("Parse() error = %T, want *ParseError", err)
	}
	requireValidationCodes(t, parseErr.ValidationErrors(), CodeInvalidURLUnit)
	if !errors.Is(err, ErrInvalidURLUnit) {
		t.Fatalf("Parse() error = %v, want ErrInvalidURLUnit", err)
	}
	if errors.Is(err, ErrPortInvalid) {
		t.Fatalf("Parse() reached the invalid port after fail-fast: %v", err)
	}
}

func TestValidationPolicy_ExtensionPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		extension  ParserOption
		want       string
		wantCode   ValidationErrorCode
		wantTarget error
	}{
		{
			name:       "lax opaque host",
			input:      "foo://example^example/",
			extension:  WithLaxHostParsing(),
			want:       "foo://example^example/",
			wantCode:   CodeHostInvalidCodePoint,
			wantTarget: ErrHostInvalidCodePoint,
		},
		{
			name:       "lax IDNA host",
			input:      "https://ä.\u200d.example/",
			extension:  WithLaxHostParsing(),
			want:       "https://ä.\u200d.example/",
			wantCode:   CodeDomainToASCII,
			wantTarget: ErrDomainToASCII,
		},
		{
			name:       "invalid UTF-8 acceptance",
			input:      "foo://\xff/",
			extension:  WithAcceptInvalidCodepoints(),
			want:       "foo://%FF/",
			wantCode:   CodeInvalidURLUnit,
			wantTarget: ErrInvalidURLUnit,
		},
		{
			name:       "single percent recovery",
			input:      "https://example.org/a%z",
			extension:  WithPercentEncodeSinglePercentSign(),
			want:       "https://example.org/a%25z",
			wantCode:   CodeInvalidURLUnit,
			wantTarget: ErrInvalidURLUnit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" recovers after detection", func(t *testing.T) {
			u, err := NewParser(tt.extension).Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got := u.String(); got != tt.want {
				t.Fatalf("Parse() = %q, want %q", got, tt.want)
			}
			requireValidationCodes(t, u.ValidationErrors(), tt.wantCode)
		})

		t.Run(tt.name+" fail-fast prevents recovery", func(t *testing.T) {
			orders := []struct {
				name    string
				options []ParserOption
			}{
				{"extension then strict", []ParserOption{tt.extension, WithFailOnValidationError()}},
				{"strict then extension", []ParserOption{WithFailOnValidationError(), tt.extension}},
			}
			for _, order := range orders {
				t.Run(order.name, func(t *testing.T) {
					u, err := NewParser(order.options...).Parse(tt.input)
					if u != nil {
						t.Fatalf("Parse() URL = %q, want nil", u)
					}
					var parseErr *ParseError
					if !errors.As(err, &parseErr) {
						t.Fatalf("Parse() error = %T, want *ParseError", err)
					}
					requireValidationCodes(t, parseErr.ValidationErrors(), tt.wantCode)
					if !errors.Is(err, tt.wantTarget) {
						t.Fatalf("Parse() error = %v, want %v", err, tt.wantTarget)
					}
				})
			}
		})
	}
}

func TestValidationPolicy_CrossProduct(t *testing.T) {
	const input = "foo://example^example/"
	tests := []struct {
		name      string
		options   []ParserOption
		wantOK    bool
		wantURL   string
		wantCodes []ValidationErrorCode
	}{
		{
			name:      "default",
			wantCodes: []ValidationErrorCode{CodeHostInvalidCodePoint},
		},
		{
			name:      "reporting disabled",
			options:   []ParserOption{WithReportValidationErrors(false)},
			wantCodes: []ValidationErrorCode{CodeHostInvalidCodePoint},
		},
		{
			name:      "fail on validation",
			options:   []ParserOption{WithFailOnValidationError()},
			wantCodes: []ValidationErrorCode{CodeHostInvalidCodePoint},
		},
		{
			name:      "lax recovery",
			options:   []ParserOption{WithLaxHostParsing()},
			wantOK:    true,
			wantURL:   input,
			wantCodes: []ValidationErrorCode{CodeHostInvalidCodePoint},
		},
		{
			name: "fail on validation with lax recovery",
			options: []ParserOption{
				WithLaxHostParsing(),
				WithFailOnValidationError(),
			},
			wantCodes: []ValidationErrorCode{CodeHostInvalidCodePoint},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewParser(tt.options...).Parse(input)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				if got := u.Href(false); got != tt.wantURL {
					t.Fatalf("Parse() = %q, want %q", got, tt.wantURL)
				}
				requireValidationCodes(t, u.ValidationErrors(), tt.wantCodes...)
				return
			}

			if u != nil {
				t.Fatalf("Parse() URL = %q, want nil", u)
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("Parse() error = %T, want *ParseError", err)
			}
			requireValidationCodes(t, parseErr.ValidationErrors(), tt.wantCodes...)
		})
	}
}

func TestValidationPolicy_FailFastBeforeDomainRecovery(t *testing.T) {
	tests := []struct {
		name string
		url  string
		code ValidationErrorCode
		err  error
	}{
		{
			name: "percent-encoded domain",
			url:  "https://exam%70le.org/",
			code: CodeDomainPercentEncoded,
			err:  ErrDomainPercentEncoded,
		},
		{
			name: "IDNA-derived IPv4",
			url:  "https://①.②.③.④/",
			code: CodeIPv4NonASCIIInput,
			err:  ErrIPv4NonASCIIInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewParser(WithFailOnValidationError()).Parse(tt.url)
			if u != nil {
				t.Fatalf("Parse() URL = %q, want nil", u)
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("Parse() error = %T, want *ParseError", err)
			}
			requireValidationCodes(t, parseErr.ValidationErrors(), tt.code)
			if !errors.Is(err, tt.err) {
				t.Fatalf("Parse() error = %v, want %v", err, tt.err)
			}
		})
	}
}
