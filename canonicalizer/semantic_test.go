package canonicalizer_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	canon "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func readSemanticFixture(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile("../testdata/iipc-urlcanon/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

// Fixture strings represent bytes: each JSON code point becomes one byte.
// This preserves invalid UTF-8 rather than transcoding it during JSON decoding.
func fixtureBytes(t *testing.T, s string) string {
	t.Helper()
	bytes := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 255 {
			t.Fatalf("fixture code point %U does not represent one byte", r)
		}
		bytes = append(bytes, byte(r))
	}
	return string(bytes)
}

func TestSemanticURLCanonFixtures(t *testing.T) {
	var vectors map[string]string
	readSemanticFixture(t, "semantic_precise.json", &vectors)
	for input, expected := range vectors {
		t.Run(input, func(t *testing.T) {
			want := fixtureBytes(t, expected)
			// Semantic removes fragments; semantic_precise retains them.
			want, _, _ = strings.Cut(want, "#")
			checkProfile(t, canon.Semantic(), fixtureBytes(t, input), want)
		})
	}
}

func TestSemanticHostNormalization(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"file:", "file:///"},
		{"file:///path?", "file:///path?"},
		{"custom://?", "custom://?"},
		{"custom:///path", "custom:///path"},
		{"http://.../path", "http://0.0.0.0/path"},
		{"custom://.../path", "custom://0.0.0.0/path"},
		{"custom://!example_~.org/path", "custom://!example_~.org/path"},
		{"custom://%2521example%255F%257E.org/path", "custom://!example_~.org/path"},
		{"custom://%255Bbad%255D/path", "custom://%5Bbad%5D/path"},
		{"custom://example%252F%253A%2540%255B%255C%255D%253F%2523%253C%253E%255E%257C.org/path", "custom://example%2F%3A%40%5B\\%5D%3F%23<>^|.org/path"},
		{"custom://[2001:db8::1]/path", "custom://[2001:db8::1]/path"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			u := checkProfile(t, canon.Semantic(), tc.input, tc.want)
			canon.Semantic().Canonicalize(u)
			if got := u.String(); got != tc.want {
				t.Fatalf("second canonicalization = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSemanticPreservesLiteralInvalidBytes(t *testing.T) {
	checkProfile(t, canon.Semantic(),
		"http://example.org/\x91\xFC?b=\x80&a=%FC&b=%91",
		"http://example.org/%91%FC?a=%FC&b=%80&b=%91")
}

func TestSemanticForbiddenOpaqueHostOption(t *testing.T) {
	const input = "custom://a^b/"
	checkProfile(t, canon.Semantic(), input, input)
	checkProfile(t, canon.Semantic(canon.WithAllowForbiddenOpaqueHost(false)), input, "custom://a%5Eb/")
	checkProfile(t, canon.Semantic(), input, input)
	for _, profile := range []canon.Profile{
		canon.Semantic().WithParser(url.NewParser()),
		canon.Semantic().WithParser(url.NewParser(url.WithLaxHostParsing(), url.WithFailOnValidationError())),
	} {
		if _, err := profile.Parse(input); err == nil {
			t.Fatal("canonicalization bypassed host validation")
		}
	}
	withoutHistory := canon.Semantic().WithParser(url.NewParser(url.WithLaxHostParsing(), url.WithReportValidationErrors(false)))
	u := checkProfile(t, withoutHistory, input, input)
	if len(u.ValidationErrors()) != 0 {
		t.Fatal("canonicalization restored disabled validation history")
	}
	custom := canon.New(canon.RepeatedPercentDecoding()).WithParser(url.NewParser(url.WithLaxHostParsing()))
	checkProfile(t, custom, input, "custom://a%5Eb/")
	checkProfile(t, canon.New(canon.RepeatedPercentDecoding(canon.WithAllowForbiddenOpaqueHost(true))).WithParser(custom.Parser()), input, input)
}

func TestSemanticIPv4RecoveryUsesConfiguredParser(t *testing.T) {
	u := checkProfile(t, canon.Semantic(), "http://09/", "http://09/")
	if u.IsIPv4() || u.IsIPv6() || len(u.ValidationErrors()) == 0 {
		t.Fatal("recovery must retain validation and classify the result as a hostname")
	}
	for _, parser := range []url.Parser{
		url.NewParser(),
		url.NewParser(url.WithLaxHostParsing(), url.WithFailOnValidationError()),
	} {
		if _, err := canon.Semantic().WithParser(parser).Parse("http://09/"); err == nil {
			t.Fatal("profile bypassed parser rejection")
		}
	}
}
