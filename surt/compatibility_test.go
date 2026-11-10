package surt

import "testing"

// These vectors were converted from the Internet Archive Python test suite:
// https://github.com/internetarchive/surt/blob/master/tests/test_surt.py
// Scheme-inclusive expectations follow upstream output, including the opening
// parenthesis after "scheme://".
func TestUpstreamSURTCompatibility(t *testing.T) {
	tests := []struct {
		name string
		in   string
		opts []Option
		want string
	}{
		{"empty", "", nil, "-"},
		{"filedesc", "filedesc:foo.arc.gz", nil, "filedesc:foo.arc.gz"},
		{"filedesc slash", "filedesc:/foo.arc.gz", nil, "filedesc:/foo.arc.gz"},
		{"filedesc authority", "filedesc://foo.arc.gz", nil, "filedesc://foo.arc.gz"},
		{"warcinfo", "warcinfo:foo.warc.gz", nil, "warcinfo:foo.warc.gz"},
		{"dns", "dns:archive.org", nil, "dns:archive.org"},
		{"mailto", "mailto:foo@example.com", nil, "mailto:foo@example.com"},
		{"www massage", "http://www.archive.org/", nil, "org,archive)/"},
		{"plain host", "http://archive.org/", nil, "org,archive)/"},
		{"trailing slash", "http://archive.org/goo/", nil, "org,archive)/goo"},
		{"empty query", "http://archive.org/goo/?", nil, "org,archive)/goo"},
		{"bare query", "http://archive.org/goo/?b&a", nil, "org,archive)/goo?a&b"},
		{"query ordering", "http://archive.org/goo/?a=2&b&a=1", nil, "org,archive)/goo?a=1&a=2&b"},
		{"trailing comma", "http://archive.org/goo/?a=2&b&a=1", []Option{WithTrailingComma(true)}, "org,archive,)/goo?a=1&a=2&b"},
		{"php session", "http://archive.org/index.php?PHPSESSID=0123456789abcdefghijklemopqrstuv&action=profile;u=4221", nil, "org,archive)/index.php?action=profile;u=4221"},
		{"whois", "whois://whois.isoc.org.il/shaveh.co.il", nil, "il,org,isoc,whois)/shaveh.co.il"},
		{"scheme", "http://www.example.com/", []Option{WithIncludeScheme(true)}, "http://(com,example)/"},
		{"scheme and comma", "https://www.example.com/", []Option{WithIncludeScheme(true), WithTrailingComma(true)}, "https://(com,example,)/"},
		{"scheme no massage", "ftp://www.example.com/", []Option{WithIncludeScheme(true), WithTrailingComma(true), WithHostMassage(false)}, "ftp://(com,example,www,)/"},
		{"whois scheme", "whois://whois.isoc.org.il/shaveh.co.il", []Option{WithIncludeScheme(true)}, "whois://(il,org,isoc,whois)/shaveh.co.il"},
		{"encoded unicode", "http://example.com/app?item=Wroc%C5%82aw", nil, "com,example)/app?item=wroc%c5%82aw"},
		{"path session", "http://archive.org/(S(4hqa0555fwsecu455xqckv45))/Mileg.aspx", nil, "org,archive)/mileg.aspx"},
		{"http default port", "http://www.archive.org:80/", nil, "org,archive)/"},
		{"https non-default port", "https://www.archive.org:80/", nil, "org,archive:80)/"},
		{"https default port", "https://www.archive.org:443/", nil, "org,archive)/"},
		{
			"yahoo web bug",
			"http://visit.webhosting.yahoo.com/visit.gif?&r=http%3A//web.archive.org/web/20090517140029/http%3A//anthonystewarthead.electric-chi.com/&b=Netscape%205.0%20%28Windows%3B%20en-US%29&s=1366x768&o=Win32&c=24&j=true&v=1.2",
			nil,
			"com,yahoo,webhosting,visit)/visit.gif?&b=netscape%205.0%20(windows;%20en-us)&c=24&j=true&o=win32&r=http://web.archive.org/web/20090517140029/http://anthonystewarthead.electric-chi.com/&s=1366x768&v=1.2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := String(test.in, test.opts...)
			if err != nil {
				t.Fatalf("String(%q): %v", test.in, err)
			}
			if got != test.want {
				t.Fatalf("String(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestUpstreamHostToSURTCompatibility(t *testing.T) {
	tests := []struct {
		host          string
		reverseIPAddr bool
		want          string
	}{
		{"www.archive.org", true, "org,archive,www"},
		{"123.123.net", true, "net,123,123"},
		{"100.100.100.100.org", true, "org,100,100,100,100"},
		{"123.45.167.89", true, "89,167,45,123"},
		{"123.45.167.89", false, "123.45.167.89"},
		{"10.162.1024.3", false, "3,1024,162,10"},
		{"990.991.992.993", true, "993,992,991,990"},
		{"990.991.992.993", false, "990.991.992.993"},
	}
	for _, test := range tests {
		if got := hostToSURTKey(test.host, test.reverseIPAddr); got != test.want {
			t.Errorf("hostToSURTKey(%q, %v) = %q, want %q", test.host, test.reverseIPAddr, got, test.want)
		}
	}
}

func TestUpstreamHostMassageCompatibility(t *testing.T) {
	tests := map[string]string{
		"foo.com":              "foo.com",
		"www.foo.com":          "foo.com",
		"www12.foo.com":        "foo.com",
		"www2foo.com":          "www2foo.com",
		"www2.www2foo.com":     "www2foo.com",
		"www.www2.example.com": "www2.example.com",
	}
	for input, want := range tests {
		if got := massageHost(input); got != want {
			t.Errorf("massageHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUpstreamPathSessionCompatibility(t *testing.T) {
	tests := map[string]string{
		"/(S(4hqa0555fwsecu455xqckv45))/mileg.aspx":                                                       "/mileg.aspx",
		"/(4hqa0555fwsecu455xqckv45)/mileg.aspx":                                                          "/mileg.aspx",
		"/(a(4hqa0555fwsecu455xqckv45)S(4hqa0555fwsecu455xqckv45)f(4hqa0555fwsecu455xqckv45))/mileg.aspx": "/mileg.aspx",
		"/photos/36050182@N05/":                                                                           "/photos/36050182@N05/",
	}
	for input, want := range tests {
		if got := stripPathSessionID(input); got != want {
			t.Errorf("stripPathSessionID(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestInternetArchiveDocumentedQueryOrdering(t *testing.T) {
	got, err := String("http://archive.org/goo/?a=2&b&a=1")
	if err != nil {
		t.Fatal(err)
	}
	if want := "org,archive)/goo?a=1&a=2&b"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

// The four data pairs from internetarchive/surt PR #30 at head revision
// 9a6717a64e374c4690cf6e28ae7d3504b3bda6cc:
// https://github.com/internetarchive/surt/pull/30
func TestUpstreamSURTNonUTF8Compatibility(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"http://example.com/city-of-M%C3%BCnchen.html", "com,example)/city-of-m%c3%bcnchen.html"},
		{"http://example.com/city-of-M%FCnchen.html", "com,example)/city-of-m%fcnchen.html"},
		{"http://www.aluroba.com/tags/%C3%CE%CA%C7%D1%E5%C7.htm", "com,aluroba)/tags/%c3%ce%ca%c7%d1%e5%c7.htm"},
		{"https://1kr.ua/newslist.html?tag=%E4%EE%F8%EA%EE%EB%FC%ED%EE%E5", "ua,1kr)/newslist.html?tag=%e4%ee%f8%ea%ee%eb%fc%ed%ee%e5"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := String(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("String(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
