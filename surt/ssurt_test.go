package surt

import (
	"strings"
	"testing"
)

func TestSSURTString(t *testing.T) {
	tests := []struct {
		in            string
		includeScheme bool
		want          string
	}{
		{"http://www.example.com", true, "com,example,www,//http:/"},
		{"http://127.0.0.1/foo/bar", true, "127.0.0.1//http:/foo/bar"},
		{"http://[::1]/foo/bar", true, "[::1]//http:/foo/bar"},
		{"http://example.com:8080/foo?query#fragment", true, "com,example,//8080:http:/foo?query#fragment"},
		{"http://user:pass@foo.example.org:81/path", true, "org,example,foo,//81:http@user:pass:/path"},
		{"http://archive.org/goo/?a=2&b&a=1", false, "org,archive,///goo/?a=2&b&a=1"},
	}
	for _, test := range tests {
		got, err := SSURTString(test.in, test.includeScheme)
		if err != nil || got != test.want {
			t.Errorf("SSURTString(%q) = %q, %v; want %q", test.in, got, err, test.want)
		}
	}
}

func TestUnSSURT(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"domain", "org,example,foo,//81:http@user:pass:/path?query#frag", "http://user:pass@foo.example.org:81/path?query#frag", ""},
		{"ipv4", "127.0.0.1//:http:/foo", "http://127.0.0.1/foo", ""},
		{"passthrough", "mailto:foo@example.com", "mailto:foo@example.com", ""},
		{"missing delimiter", "com,example/path", "", "missing '//'"},
		{"malformed metadata", "com,example,//http/", "", "malformed metadata"},
		{"bad port", "com,example,//99999:http:/", "", "malformed port"},
		{"bad scheme", "com,example,//:1http:/", "", "malformed scheme"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnSSURT(test.in)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("UnSSURT() error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("UnSSURT() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestUnSSURTAcceptsLegacyEmptyPortSeparator(t *testing.T) {
	got, err := UnSSURT("com,example,//:http:/path")
	if err != nil || got != "http://example.com/path" {
		t.Fatalf("UnSSURT() = %q, %v", got, err)
	}
}

func TestNormalizedSSURTRoundTrip(t *testing.T) {
	first, err := SSURTString("http://user:pass@example.org:81/path?query#frag", true)
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := UnSSURT(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SSURTString(reversed, true)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("round trip: first=%q reversed=%q second=%q", first, reversed, second)
	}
}
