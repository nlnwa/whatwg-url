package surt

import (
	"errors"
	"strings"
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	whatwgurl "github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestSURTCanonicalization(t *testing.T) {
	tests := []struct {
		name string
		in   string
		opts []Option
		want string
	}{
		{"repeated percent normalization", "http://host/%25%32%35", nil, "host)/%25"},
		{"credentials fragment case slash", "HTTP://User:Pass@WWW.Example.COM:80/Some/Path/#Fragment", nil, "com,example)/some/path"},
		{"non-default port", "https://www.example.com:80/path", nil, "com,example:80)/path"},
		{"scheme port comma", "https://www.example.com:80/path", []Option{WithIncludeScheme(true), WithTrailingComma(true)}, "https://(com,example:80,)/path"},
		{"reverse ipv4", "http://192.168.1.254/info/", nil, "254,1,168,192)/info"},
		{"keep ipv4", "http://192.168.1.254/info/", []Option{WithReverseIPAddr(false)}, "192.168.1.254)/info"},
		{"ipv6", "http://[2001:db8::1]/Path/", nil, "[2001:db8::1])/path"},
		{"lossless raw terms", "http://example.com/?b&empty=&a=2&&a=1&q=a+b&space=%20&slash=%2f", nil, "com,example)/?&a=1&a=2&b&empty=&q=a+b&slash=/&space=%20"},
		{"escaped query-name separator", "http://example.com/?CFID%253DX=1&cfid=removed", nil, "com,example)/?cfid%3dx=1"},
		{"default scheme fallback", "www.example.com/Path", nil, "com,example)/path"},
		{"trimmed default scheme fallback", "  www.example.com/Path  ", nil, "com,example)/path"},
		{"literal invalid bytes", "http://www.example.com/\xFC?q=\x91", nil, "com,example)/%fc?q=%91"},
		{"lax IPv4 hostname", "http://09/path", nil, "09)/path"},
		{"lax malformed IPv6", "http://[1::2::3]/Path/", nil, "[1::2::3])/path"},
		{"lax unclosed IPv6", "http://[::1/Path/", nil, "[::1)/path"},
		{"above-root parent", "https://domain.com/../path", nil, "com,domain)/path"},
		{"parent after segment", "https://domain.com/a/../path", nil, "com,domain)/path"},
		{"parent beyond root", "https://domain.com/a/../../path", nil, "com,domain)/path"},
		{"encoded above-root parent", "https://domain.com/%2e%2e/path", nil, "com,domain)/path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := String(test.in, test.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestQueryFilterOptions(t *testing.T) {
	id := "0123456789abcdefghijklemopqrstuv"
	tests := []struct {
		name string
		opts []Option
		want string
	}{
		{"default", nil, "org,archive)/?keep=1"},
		{"custom replaces default", []Option{WithParser(canonicalizer.IA(canonicalizer.WithQueryFilter(func(name, _ string, _ bool) bool { return name != "keep" })))}, "org,archive)/?jsessionid=" + id},
		{"disabled", []Option{WithParser(canonicalizer.IA(canonicalizer.WithQueryFilter(nil)))}, "org,archive)/?jsessionid=" + id + "&keep=1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := String("http://archive.org/?jsessionid="+id+"&keep=1", test.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDefaultQuerySessionFilters(t *testing.T) {
	id := "0123456789abcdefghijklemopqrstuv"
	tests := []string{
		"jsessionid=" + id,
		"phpsessid=" + id,
		"sid=" + id,
		"aspsessionidABCDEFGH=ABCDEFGHIJKLMNOPQRSTUVWX",
		"CFID=1169580",
		"CFTOKEN=48630702",
	}
	for _, filtered := range tests {
		got, err := String("http://example.org/?keep=1&" + filtered)
		if err != nil {
			t.Fatal(err)
		}
		if got != "org,example)/?keep=1" {
			t.Errorf("filtering %q produced %q", filtered, got)
		}
	}
}

func TestCanonicalizationThenURLDoesNotMutateInput(t *testing.T) {
	u, err := canonicalizer.IA().Parse("https://User:Pass@WWW.Example.ORG:80/Path/?B&A#Frag")
	if err != nil {
		t.Fatal(err)
	}
	before := u.String()
	got, err := URL(u, WithIncludeScheme(true))
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://(org,example:80)/path?a&b" {
		t.Fatalf("URL() = %q", got)
	}
	if u.String() != before {
		t.Fatalf("URL mutated input: got %q, want %q", u, before)
	}
}

func TestURLNil(t *testing.T) {
	got, err := URL(nil)
	if err != nil || got != "" {
		t.Fatalf("URL(nil) = %q, %v", got, err)
	}
}

func TestURLFormatsWithoutCanonicalization(t *testing.T) {
	u, err := whatwgurl.Parse("https://User:Pass@WWW.Example.ORG:80/Path/?B&A#Frag")
	if err != nil {
		t.Fatal(err)
	}
	before := u.String()
	params := u.SearchParams()
	parser := &countingParser{Parser: canonicalizer.IA()}
	got, err := URL(u, WithIncludeScheme(true), WithParser(parser))
	const want = "https://(org,example,www:80)/Path/?B&A"
	if err != nil || got != want {
		t.Fatalf("URL() = %q, %v; want %q", got, err, want)
	}
	if u.String() != before || u.SearchParams() != params || parser.calls != 0 {
		t.Fatal("formatter changed the URL or invoked a parser")
	}
	opaque, err := whatwgurl.Parse("custom:opaque?b=2&a=1#f")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := URL(opaque); err != nil || got != opaque.String() {
		t.Fatalf("hostless formatting = %q, %v", got, err)
	}
}

type countingParser struct {
	whatwgurl.Parser
	calls int
	err   error
}

func (p *countingParser) Parse(raw string) (*whatwgurl.Url, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return p.Parser.Parse(raw)
}

func TestStringUsesSelectedParserOnce(t *testing.T) {
	rules := 0
	profile := canonicalizer.IA(canonicalizer.WithHostMassage(false)).Then(func(u *whatwgurl.Url) {
		rules++
		u.RewritePathname(func(_ *whatwgurl.Url, path string) string { return path + "/rule" })
	})
	parser := &countingParser{Parser: profile}
	got, err := String("http://www.example.org/Path/?B=2&A=1", WithParser(parser))
	if err != nil || got != "org,example,www)/path/rule?a=1&b=2" || parser.calls != 1 || rules != 1 {
		t.Fatalf("String() = %q, %v; parser calls=%d, rule calls=%d", got, err, parser.calls, rules)
	}
	const input = "http://www.example.org/Path/?B=2&A=1"
	got, err = String(input, WithParser(whatwgurl.NewParser()))
	if err != nil || got != "org,example,www)/Path/?B=2&A=1" {
		t.Fatalf("plain parser was canonicalized: %q, %v", got, err)
	}
	got, err = String(input, WithParser(profile), WithParser(nil))
	if err != nil || got != "org,example)/path?a=1&b=2" {
		t.Fatalf("nil did not restore default parser: %q, %v", got, err)
	}
	if _, err := String("example.org/path", WithParser(whatwgurl.NewParser())); !errors.Is(err, whatwgurl.ErrMissingSchemeNonRelativeURL) {
		t.Fatalf("SURT applied its own missing-scheme fallback: %v", err)
	}
}

func TestStringPreservesParserErrorsAndFormattingConventions(t *testing.T) {
	sentinel := errors.New("custom parser failure")
	parser := &countingParser{err: sentinel}
	if got, err := String("http://example.org/", WithParser(parser)); got != "" || err != sentinel || parser.calls != 1 {
		t.Fatalf("parser failure changed: %q, %v; calls=%d", got, err, parser.calls)
	}
	for _, tc := range []struct{ input, want string }{
		{"", "-"},
		{"warcinfo:raw input", "warcinfo:raw input"},
	} {
		got, err := String(tc.input, WithParser(parser))
		if err != nil || got != tc.want || parser.calls != 1 {
			t.Fatalf("formatting convention invoked parser: %q, %v; calls=%d", got, err, parser.calls)
		}
	}
}

func TestSharedIAProfileCanonicalizesOnce(t *testing.T) {
	// Host massage removes one label. Applying the profile twice would also
	// remove www2 and change this key.
	got, err := String("http://www.www2.example.org/Path")
	if err != nil || got != "org,example,www2)/path" {
		t.Fatalf("String() = %q, %v; canonicalization must run once", got, err)
	}
}

func TestSSURTUsesSharedParserWithoutCanonicalizing(t *testing.T) {
	got, err := SSURTString("http://User:Pass@WWW.Example.org/Path/?B=2&A=1#Frag", true)
	const want = "org,example,www,//http@User:Pass:/Path/?B=2&A=1#Frag"
	if err != nil || got != want {
		t.Fatalf("SSURTString() = %q, %v; want %q", got, err, want)
	}
}

func TestUnSURT(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"empty", "-", "", ""},
		{"default scheme", "org,archive)/goo?a=1&a=2&b", "http://archive.org/goo?a=1&a=2&b", ""},
		{"scheme port comma", "https://(com,example:80,)/path", "https://example.com:80/path", ""},
		{"reversed ipv4", "254,1,168,192)/info", "http://192.168.1.254/info", ""},
		{"ipv6", "https://([2001:db8::1]:8443)/", "https://[2001:db8::1]:8443/", ""},
		{"passthrough", "mailto:foo@example.com", "mailto:foo@example.com", ""},
		{"missing delimiter", "com,example/path", "", "missing ')'"},
		{"empty label", "com,,example)/", "", "empty host label"},
		{"bad scheme", "https://com,example)/", "", "malformed"},
		{"invalid scheme", "1http://(com,example)/", "", "malformed scheme"},
		{"bad port", "https://(com,example:99999)/", "", "malformed port"},
		{"query-only path", "com,example)?q=1", "http://example.com/?q=1", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnSURT(test.in)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("UnSURT() error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("UnSURT() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestNormalizedSURTRoundTrip(t *testing.T) {
	first, err := String("HTTPS://WWW.Example.com:80/Path/?b=2&a=1#frag", WithIncludeScheme(true))
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := UnSURT(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := String(reversed, WithIncludeScheme(true))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("normalized round trip: first=%q reversed=%q second=%q", first, reversed, second)
	}
}
