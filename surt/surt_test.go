package surt

import (
	"strings"
	"testing"

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
		{"default scheme fallback", "www.example.com/Path", nil, "com,example)/path"},
		{"trimmed default scheme fallback", "  www.example.com/Path  ", nil, "com,example)/path"},
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
		{"custom replaces default", []Option{WithFilteredQueryParams("keep")}, "org,archive)/?jsessionid=" + id},
		{"disabled", []Option{WithQueryParamFilter(nil)}, "org,archive)/?jsessionid=" + id + "&keep=1"},
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

func TestURLDoesNotMutateInput(t *testing.T) {
	u, err := whatwgurl.Parse("https://User:Pass@WWW.Example.ORG:80/Path/?B&A#Frag")
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
