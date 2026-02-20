package surt

import "testing"

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		u       string
		opts    []Option
		want    string
		wantErr bool
	}{
		{"empty", "", nil, "-", false},
		{"dns-pass-through", "dns:archive.org", nil, "dns:archive.org", false},
		{"mailto-pass-through", "mailto:foo@example.com", nil, "mailto:foo@example.com", false},
		{"ia-default", "http://archive.org/goo/?a=2&b&a=1", nil, "org,archive)/goo?a=1&a=2&b", false},
		{"ia-trailing", "http://archive.org/goo/?a=2&b&a=1", []Option{WithTrailingComma(true)}, "org,archive,)/goo?a=1&a=2&b", false},
		{"ia-no-reverse-ip", "http://123.45.78.90/goo/?a=2&b&a=1", []Option{WithReverseIPAddr(false)}, "123.45.78.90)/goo?a=1&a=2&b", false},
		{"host-massage-default", "http://www.example.com/", nil, "com,example)/", false},
		{"host-massage-off", "http://www.example.com/", []Option{WithHostMassage(false)}, "com,example,www)/", false},
		{"include-scheme", "http://www.example.com/", []Option{WithIncludeScheme(true), WithTrailingComma(true)}, "http://com,example,)/", false},
		{"include-scheme-no-massage", "http://www.example.com/", []Option{WithIncludeScheme(true), WithTrailingComma(true), WithHostMassage(false)}, "http://com,example,www,)/", false},
		{"custom-filter-by-name", "http://archive.org/?foo=1&bar=2", []Option{WithFilteredQueryParams("foo")}, "org,archive)/?bar=2", false},
		{"disable-query-filter", "http://archive.org/?jsessionid=0123456789abcdefghijklemopqrstuv", []Option{WithQueryParamFilter(nil)}, "org,archive)/?jsessionid=0123456789abcdefghijklemopqrstuv", false},
		{"whois", "whois://whois.isoc.org.il/shaveh.co.il", nil, "il,org,isoc,whois)/shaveh.co.il", false},
		{"whois-trailing-scheme", "whois://whois.isoc.org.il/shaveh.co.il", []Option{WithTrailingComma(true), WithIncludeScheme(true)}, "whois://il,org,isoc,whois,)/shaveh.co.il", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := String(tt.u, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("String() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("String() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnSURT(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		isErr bool
	}{
		{"default-http", "org,archive)/goo?a=1&a=2&b", "http://archive.org/goo?a=1&a=2&b", false},
		{"with-scheme", "https://com,example,)/", "https://example.com/", false},
		{"passthrough", "dns:archive.org", "dns:archive.org", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnSURT(tt.in)
			if (err != nil) != tt.isErr {
				t.Fatalf("UnSURT() error = %v, wantErr %v", err, tt.isErr)
			}
			if got != tt.want {
				t.Fatalf("UnSURT() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCompatibilitySurtVectors validates SURT outputs against external compatibility vectors.
func TestCompatibilitySurtVectors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		opts []Option
		want string
	}{
		{"none-like-empty", "", nil, "-"},
		{"filedesc", "filedesc:foo.arc.gz", nil, "filedesc:foo.arc.gz"},
		{"filedesc-slash", "filedesc:/foo.arc.gz", nil, "filedesc:/foo.arc.gz"},
		{"filedesc-slash2", "filedesc://foo.arc.gz", nil, "filedesc://foo.arc.gz"},
		{"warcinfo", "warcinfo:foo.warc.gz", nil, "warcinfo:foo.warc.gz"},
		{"dns", "dns:archive.org", nil, "dns:archive.org"},
		{"http-www", "http://www.archive.org/", nil, "org,archive)/"},
		{"http-no-www", "http://archive.org/", nil, "org,archive)/"},
		{"path-trailing-slash", "http://archive.org/goo/", nil, "org,archive)/goo"},
		{"query-empty", "http://archive.org/goo/?", nil, "org,archive)/goo"},
		{"query-sort-1", "http://archive.org/goo/?b&a", nil, "org,archive)/goo?a&b"},
		{"query-sort-2", "http://archive.org/goo/?a=2&b&a=1", nil, "org,archive)/goo?a=1&a=2&b"},
		{"trailing-comma", "http://archive.org/goo/?a=2&b&a=1", []Option{WithTrailingComma(true)}, "org,archive,)/goo?a=1&a=2&b"},
		{"whois", "whois://whois.isoc.org.il/shaveh.co.il", nil, "il,org,isoc,whois)/shaveh.co.il"},
		{"with-scheme", "http://www.example.com/", []Option{WithIncludeScheme(true)}, "http://com,example)/"},
		{"with-scheme-no-massage", "http://www.example.com/", []Option{WithIncludeScheme(true), WithHostMassage(false)}, "http://com,example,www)/"},
		{"mailto", "mailto:foo@example.com", nil, "mailto:foo@example.com"},
		{"reverse-ip-default", "http://192.168.1.254/info/", nil, "254,1,168,192)/info"},
		{"reverse-ip-off", "http://192.168.1.254/info/", []Option{WithReverseIPAddr(false)}, "192.168.1.254)/info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := String(tt.in, tt.opts...)
			if err != nil {
				t.Fatalf("String() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("String() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompatibilityHostToSURTVectors validates host-to-key conversion behavior.
func TestCompatibilityHostToSURTVectors(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		reverseIP bool
		want      string
	}{
		{"domain", "www.archive.org", true, "org,archive,www"},
		{"numeric-domain", "123.123.net", true, "net,123,123"},
		{"numeric-domain-long", "100.100.100.100.org", true, "org,100,100,100,100"},
		{"ipv4-reverse", "123.45.167.89", true, "89,167,45,123"},
		{"ipv4-no-reverse", "123.45.167.89", false, "123.45.167.89"},
		{"ip-like-out-of-range", "990.991.992.993", true, "993,992,991,990"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostToSURTKey(tt.host, false, tt.reverseIP)
			if got != tt.want {
				t.Fatalf("hostToSURTKey() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompatibilityMassageHostVectors validates host massaging behavior.
func TestCompatibilityMassageHostVectors(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"foo.com", "foo.com"},
		{"www.foo.com", "foo.com"},
		{"www12.foo.com", "foo.com"},
		{"www2foo.com", "www2foo.com"},
		{"www2.www2foo.com", "www2foo.com"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := massageHost(tt.in)
			if got != tt.want {
				t.Fatalf("massageHost() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompatibilityQueryReorderVectors validates query ordering through the public SURT API.
func TestCompatibilityQueryReorderVectors(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"http://archive.org/?a", "org,archive)/?a"},
		{"http://archive.org/?a=1", "org,archive)/?a=1"},
		{"http://archive.org/?b=1&a=1", "org,archive)/?a=1&b=1"},
		{"http://archive.org/?a=b&a=a", "org,archive)/?a=a&a=b"},
		{"http://archive.org/?b=b&a=b&b=a&a=a", "org,archive)/?a=a&a=b&b=a&b=b"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := String(tt.in)
			if err != nil {
				t.Fatalf("String() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("String() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompatibilityQueryParamFilterVectors validates canonical query-parameter filtering vectors.
func TestCompatibilityQueryParamFilterVectors(t *testing.T) {
	str32id := "0123456789abcdefghijklemopqrstuv"
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"jsessionid-only", "?jsessionid=" + str32id, "?"},
		{"jsessionid-too-long", "?jsessionid=" + str32id + "0", "?jsessionid=0123456789abcdefghijklemopqrstuv0"},
		{"jsessionid-with-next", "?jsessionid=" + str32id + "&x=y", "?x=y"},
		{"jsessionid-middle", "?one=two&jsessionid=" + str32id + "&x=y", "?one=two&x=y"},
		{"jsessionid-last", "?one=two&jsessionid=" + str32id, "?one=two&"},
		{"aspsessionid", "?aspsessionidABCDEFGH=ABCDEFGHIJKLMNOPQRSTUVWX&x=y", "?x=y"},
		{"phpsessid", "?phpsessid=" + str32id + "&x=y", "?x=y"},
		{"phpsessid-middle", "?one=two&phpsessid=" + str32id + "&x=y", "?one=two&x=y"},
		{"phpsessid-last", "?one=two&phpsessid=" + str32id, "?one=two&"},
		{"sid", "?sid=9682993c8daa2c5497996114facdc805&x=y", "?x=y"},
		{"sid-and-jsessionid", "?sid=9682993c8daa2c5497996114facdc805&jsessionid=" + str32id, "?"},
		{"cfid-cftoken-dtstamp", "?CFID=1169580&CFTOKEN=48630702&dtstamp=22%2F08%2F2006%7C06%3A58%3A11", "?dtstamp=22%2F08%2F2006%7C06%3A58%3A11"},
		{"cfid-cftoken-dt", "?CFID=12412453&CFTOKEN=15501799&dt=19_08_2006_22_39_28", "?dt=19_08_2006_22_39_28"},
		{"cfid-cftoken-r-m", "?CFID=14475712&CFTOKEN=2D89F5AF-3048-2957-DA4EE4B6B13661AB&r=468710288378&m=forgotten", "?r=468710288378&m=forgotten"},
		{"cfid-cftoken-only", "?CFID=16603925&CFTOKEN=2AE13EEE-3048-85B0-56CEDAAB0ACA44B8", "?"},
		{"cfid-cftoken-requestid", "?CFID=4308017&CFTOKEN=63914124&requestID=200608200458360%2E39414378", "?requestID=200608200458360%2E39414378"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterQueryParameters(tt.in, shouldFilterCanonicalParam)
			if got != tt.want {
				t.Fatalf("filterQueryParameters() got = %q, want %q", got, tt.want)
			}
		})
	}
}
