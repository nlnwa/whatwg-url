package surt

import "testing"

func TestSSURTString(t *testing.T) {
	tests := []struct {
		name          string
		u             string
		includeScheme bool
		want          string
		wantErr       bool
	}{
		{"1", "http://www.example.com", true, "com,example,www,//:http:/", false},
		{"2", "http://www.example.com:80", true, "com,example,www,//:http:/", false},
		{"3", "http://www.example.com/foo/bar", true, "com,example,www,//:http:/foo/bar", false},
		{"4", "http://127.0.0.1/foo/bar", true, "127.0.0.1//:http:/foo/bar", false},
		{"5", "http://[::1]/foo/bar", true, "[::1]//:http:/foo/bar", false},
		{"6", "http://example.com/foo/bar?query#fragment", true, "com,example,//:http:/foo/bar?query#fragment", false},
		{"7", "http://example.com:8080/foo/bar?query#fragment", true, "com,example,//8080:http:/foo/bar?query#fragment", false},
		{"8", "http://user:pass@foo.example.org:81/path?query#frag", true, "org,example,foo,//81:http@user:pass:/path?query#frag", false},
		{"9", "http://user@foo.example.org:81/path?query#frag", true, "org,example,foo,//81:http@user:/path?query#frag", false},
		{"10", "scheme://foo.example.org/path?query#frag", true, "org,example,foo,//:scheme:/path?query#frag", false},
		{"11", "http://archive.org/goo/?a=2&b&a=1", true, "org,archive,//:http:/goo/?a=2&b&a=1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SSURTString(tt.u, tt.includeScheme)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SSURTString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("SSURTString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnSSURT(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		isErr bool
	}{
		{"domain", "org,example,foo,//81:http@user:pass:/path?query#frag", "http://user:pass@foo.example.org:81/path?query#frag", false},
		{"ipv4", "127.0.0.1//:http:/foo", "http://127.0.0.1/foo", false},
		{"passthrough", "mailto:foo@example.com", "mailto:foo@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnSSURT(tt.in)
			if (err != nil) != tt.isErr {
				t.Fatalf("UnSSURT() error = %v, wantErr %v", err, tt.isErr)
			}
			if got != tt.want {
				t.Fatalf("UnSSURT() got = %v, want %v", got, tt.want)
			}
		})
	}
}
