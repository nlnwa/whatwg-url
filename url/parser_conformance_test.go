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
		requireValidationCodes(t, u.ValidationErrors(), CodeIPv4EmptyPart, CodeIPv4TooFewParts)
	})

	t.Run("range event occurs once", func(t *testing.T) {
		u, err := Parse("https://256.257.1.1/")
		if err == nil || u != nil {
			t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
		}
		parseErr, ok := errors.AsType[*ParseError](err)
		if !ok {
			t.Fatalf("error = %T, want *ParseError", err)
		}
		requireValidationCodes(t, parseErr.ValidationErrors(), CodeIPv4OutOfRangePart)
	})

	t.Run("machine integer overflow is range not syntax", func(t *testing.T) {
		u, err := Parse("https://" + strings.Repeat("9", 30) + "/")
		if err == nil || u != nil {
			t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
		}
		parseErr, ok := errors.AsType[*ParseError](err)
		if !ok {
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
			parseErr, ok := errors.AsType[*ParseError](err)
			if !ok {
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

func TestOpaqueHostFusedValidationOrdering(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCodes []ValidationErrorCode
	}{
		{
			name:      "invalid URL unit before forbidden code point",
			input:     "foo://{^/",
			wantCodes: []ValidationErrorCode{CodeInvalidURLUnit, CodeHostInvalidCodePoint},
		},
		{
			name:      "malformed percent before forbidden code point",
			input:     "foo://%z^/",
			wantCodes: []ValidationErrorCode{CodeInvalidURLUnit, CodeHostInvalidCodePoint},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := Parse(tt.input)
			if err == nil || u != nil {
				t.Fatalf("Parse(%q) = (%v, %v), want nil URL and failure", tt.input, u, err)
			}
			parseErr, ok := errors.AsType[*ParseError](err)
			if !ok {
				t.Fatalf("Parse(%q) error = %T, want *ParseError", tt.input, err)
			}
			requireValidationCodes(t, parseErr.ValidationErrors(), tt.wantCodes...)
		})
	}
}

func TestOpaqueHostFusedFailFastOrdering(t *testing.T) {
	u, err := NewParser(WithFailOnValidationError()).Parse("foo://{^/")
	if err == nil || u != nil {
		t.Fatalf("Parse() = (%v, %v), want nil URL and failure", u, err)
	}
	parseErr, ok := errors.AsType[*ParseError](err)
	if !ok {
		t.Fatalf("Parse() error = %T, want *ParseError", err)
	}
	requireValidationCodes(t, parseErr.ValidationErrors(), CodeInvalidURLUnit)
	if errors.Is(err, ErrHostInvalidCodePoint) {
		t.Fatalf("fail-fast reached the later forbidden code point: %v", err)
	}
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
			parseErr, ok := errors.AsType[*ParseError](err)
			if !ok {
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
	parseErr, ok := errors.AsType[*ParseError](err)
	if !ok {
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
	parseErr, ok := errors.AsType[*ParseError](err)
	if !ok {
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
			parseErr, ok := errors.AsType[*ParseError](err)
			if !ok {
				t.Fatalf("error = %T, want *ParseError", err)
			}
			if len(parseErr.ValidationErrors()) != 0 {
				t.Fatalf("bare failure contains validation errors: %v", parseErr.ValidationErrors())
			}
		})
	}
}

func TestParsePreservesNonUTF8Bytes(t *testing.T) {
	for _, v := range []struct{ Name, Input, Parsed string }{
		{"latin1", "http://onlinestreet.de/strassen/in-M%FCnchen.html", "http://onlinestreet.de/strassen/in-M%FCnchen.html"},
		{"commoncrawl_path", "http://www.aluroba.com/tags/%C3%CE%CA%C7%D1%E5%C7.htm", "http://www.aluroba.com/tags/%C3%CE%CA%C7%D1%E5%C7.htm"},
		{"commoncrawl_query", "https://1kr.ua/newslist.html?tag=%E4%EE%F8%EA%EE%EB%FC%ED%EE%E5", "https://1kr.ua/newslist.html?tag=%E4%EE%F8%EA%EE%EB%FC%ED%EE%E5"},
		{"corpus_line_210", "http://www.tubeplus.ag/info/643723/Ya_se_qui%E9n_eres_(te_he_estado_observando)_(1971)/", "http://www.tubeplus.ag/info/643723/Ya_se_qui%E9n_eres_(te_he_estado_observando)_(1971)/"},
		{"corpus_line_212", "https://www.tubeplus.ag/player/157000/Dude%2C_Where%92s_My_Car%3F/", "https://www.tubeplus.ag/player/157000/Dude%2C_Where%92s_My_Car%3F/"},
		{"corpus_line_24487", "http://www.algerie-forum.com/forum/forum-g%C3%A9n%C3%A9ral/parlons-en/123381-un-expert-militaire-alg%E9rien-%ABla-d%E9cision-d?abattre-l?avion-russe-est-intentionnelle%BB", "http://www.algerie-forum.com/forum/forum-g%C3%A9n%C3%A9ral/parlons-en/123381-un-expert-militaire-alg%E9rien-%ABla-d%E9cision-d?abattre-l?avion-russe-est-intentionnelle%BB"},
		{"corpus_line_46175", "http://cncloisirs.com/Construction/T%EAteDeFraisage?from=Construction.D%C3%A9fonceuse", "http://cncloisirs.com/Construction/T%EAteDeFraisage?from=Construction.D%C3%A9fonceuse"},
		{"iso8859_2", "http://www.nerosty.cz/nerosty/m%EC%EF/3360_m%EC%EF", "http://www.nerosty.cz/nerosty/m%EC%EF/3360_m%EC%EF"},
		{"windows1251", "http://io.ua/c14/z/Osen/%CE%F1%E5%ED%FC", "http://io.ua/c14/z/Osen/%CE%F1%E5%ED%FC"},
		{"utf8", "https://example.org/M%C3%BCnchen?q=%C3%BC", "https://example.org/M%C3%BCnchen?q=%C3%BC"},
		{"replacement", "https://example.org/%EF%BF%BD?q=%EF%BF%BD", "https://example.org/%EF%BF%BD?q=%EF%BF%BD"},
		{"nested", "https://example.org/%25FC?q=%2525FD", "https://example.org/%25FC?q=%2525FD"},
		{"malformed", "https://example.org/%FC%GG%?q=%FD%2", "https://example.org/%FC%GG%?q=%FD%2"},
		{"names_values_duplicates", "https://example.org/%FC?%FD=%FE&b=%FF&b=%FC&flag&empty=", "https://example.org/%FC?%FD=%FE&b=%FF&b=%FC&flag&empty="},
		{"mixed", "https://example.org/%C3%BC%FC?q=%E2%82", "https://example.org/%C3%BC%FC?q=%E2%82"},
	} {
		t.Run(v.Name, func(t *testing.T) {
			u, err := Parse(v.Input)
			if err != nil {
				t.Fatal(err)
			}
			if u.String() != v.Parsed {
				t.Errorf("got %q, want %q", u.String(), v.Parsed)
			}
			reparsed, err := Parse(u.String())
			if err != nil {
				t.Fatal(err)
			}
			if reparsed.String() != v.Parsed {
				t.Errorf("reparse = %q, want %q", reparsed, v.Parsed)
			}
		})
	}
}

func TestParseNonUTF8HostBoundary(t *testing.T) {
	for _, input := range []string{"https://example%FC.org/", "https://example.org%80/"} {
		if u, err := Parse(input); err == nil || u != nil {
			t.Errorf("%q should fail WHATWG host processing", input)
		}
	}
	if u, err := Parse("https://example.org/%FC?q=%80"); err != nil || u == nil {
		t.Errorf("path/query byte escapes should parse: %v", err)
	}
}

func TestParseExclamationValidation(t *testing.T) {
	// https://url.spec.whatwg.org/#url-code-points
	// https://url.spec.whatwg.org/#path-state
	for _, input := range []string{
		"https://example.org/!",
		"https://example.org/?q=!",
		"https://example.org/#!",
		"data:!",
	} {
		t.Run(input, func(t *testing.T) {
			u, err := Parse(input)
			if err != nil {
				t.Fatal(err)
			}
			if u.String() != input {
				t.Errorf("serialization = %q, want %q", u.String(), input)
			}
			if len(u.ValidationErrors()) != 0 {
				t.Errorf("permitted '!' produced validation diagnostics: %v", u.ValidationErrors())
			}
			if _, err := NewParser(WithFailOnValidationError()).Parse(input); err != nil {
				t.Errorf("strict policy rejected permitted '!': %v", err)
			}
		})
	}
	t.Run("malformed_escape_still_rejected", func(t *testing.T) {
		if _, err := NewParser(WithFailOnValidationError()).Parse("https://example.org/%GG"); err == nil {
			t.Error("strict policy must still reject a malformed percent escape")
		}
	})
}
