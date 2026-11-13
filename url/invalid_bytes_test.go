package url

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAcceptInvalidCodepointsPreservesComponentBytes(t *testing.T) {
	components := []struct{ name, prefix string }{
		{"path", "https://example.org/"},
		{"opaque path", "custom:"},
		{"special query", "https://example.org/?x="},
		{"non-special query", "custom:base?x="},
		{"special fragment", "https://example.org/#"},
		{"non-special fragment", "custom:base#"},
	}
	for _, component := range components {
		t.Run(component.name, func(t *testing.T) {
			for _, tc := range []struct {
				input, want string
				invalid     int
			}{
				{"\x91%91", "%91%91", 1},
				{"\xFC%FC", "%FC%FC", 1},
				{"\xC3(\x80", "%C3(%80", 2},
				{"\xC0\xAF", "%C0%AF", 2},
				{"\xED\xA0\x80", "%ED%A0%80", 3},
				{"ü", "%C3%BC", 0},
				{"\u0091", "%C2%91", 0},
				{"\uFFFD", "%EF%BF%BD", 0},
			} {
				t.Run(fmt.Sprintf("%q", tc.input), func(t *testing.T) {
					input, want := component.prefix+tc.input, component.prefix+tc.want
					p := NewParser(WithAcceptInvalidCodepoints())
					u, err := p.Parse(input)
					if err != nil || u.String() != want {
						t.Fatalf("Parse(%q) = %v, %v; want %q", input, u, err, want)
					}
					invalid := 0
					for _, diagnostic := range u.ValidationErrors() {
						if strings.Contains(diagnostic.Detail(), "invalid UTF-8 byte") {
							invalid++
							if diagnostic.Code() != CodeInvalidURLUnit {
								t.Fatalf("invalid byte diagnostic code = %v", diagnostic.Code())
							}
						}
					}
					if invalid != tc.invalid {
						t.Fatalf("invalid byte diagnostics = %d, want %d", invalid, tc.invalid)
					}
					again, err := p.Parse(want)
					if err != nil || again.String() != want {
						t.Fatalf("reparse = %v, %v; want %q", again, err, want)
					}
				})
			}
			for _, options := range [][]ParserOption{
				{WithAcceptInvalidCodepoints(), WithFailOnValidationError()},
				{WithFailOnValidationError(), WithAcceptInvalidCodepoints(), WithReportValidationErrors(false)},
			} {
				u, err := NewParser(options...).Parse(component.prefix + "\xFC")
				if u != nil || !errors.Is(err, ErrInvalidURLUnit) {
					t.Fatalf("fail-fast parsing = %v, %v; want invalid-URL-unit failure", u, err)
				}
				failure, ok := errors.AsType[*ParseError](err)
				if !ok {
					t.Fatalf("error type = %T, want *ParseError", err)
				}
				requireValidationCodes(t, failure.ValidationErrors(), CodeInvalidURLUnit)
			}
			u, err := NewParser(WithAcceptInvalidCodepoints(), WithReportValidationErrors(false)).Parse(component.prefix + "\xFC")
			if err != nil || u.String() != component.prefix+"%FC" || len(u.ValidationErrors()) != 0 {
				t.Fatalf("recovery without reporting = %v, %v", u, err)
			}
		})
	}
}

func TestAcceptInvalidCodepointsRelativeResolutionAndSetters(t *testing.T) {
	p := NewParser(WithAcceptInvalidCodepoints())
	u, err := p.ParseRef("https://example.org/dir/", "\xFC?q=\x80#\x91")
	if err != nil || u.String() != "https://example.org/dir/%FC?q=%80#%91" {
		t.Fatalf("ParseRef = %v, %v", u, err)
	}
	params := u.SearchParams()
	u.SetPathname("/\x91")
	u.SetSearch("?q=\xFC")
	u.SetHash("#\x80")
	if got, want := u.String(), "https://example.org/%91?q=%FC#%80"; got != want {
		t.Fatalf("after setters = %q, want %q", got, want)
	}
	if u.SearchParams() != params || params.Get("q") != "\uFFFD" {
		t.Fatal("SearchParams did not refresh after the query setter")
	}
}
