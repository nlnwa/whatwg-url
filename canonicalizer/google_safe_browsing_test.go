package canonicalizer

import (
	"strings"
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// Canonicalization examples from Google Safe Browsing API v4:
// https://developers.google.com/safe-browsing/v4/urls-hashing#canonicalization
// Retrieved 2026-11-12; the page was last updated 2024-09-18.
// Google Developers code samples are licensed under Apache 2.0:
// https://developers.google.com/terms/site-policies
//
// Inputs and outputs are copied from the code block in the original HTML,
// concatenating syntax-highlighting spans without inserting spaces. All 33
// published expectations are asserted directly, with no overrides or skips.
func TestGoogleSafeBrowsingPublishedExamples(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"1", "http://host/%25%32%35", "http://host/%25"},
		{"2", "http://host/%25%32%35%25%32%35", "http://host/%25%25"},
		{"3", "http://host/%2525252525252525", "http://host/%25"},
		{"4", "http://host/asdf%25%32%35asd", "http://host/asdf%25asd"},
		{"5", "http://host/%%%25%32%35asd%%", "http://host/%25%25%25asd%25%25"},
		{"6", "http://www.google.com/", "http://www.google.com/"},
		{"7", "http://%31%36%38%2e%31%38%38%2e%39%39%2e%32%36/%2E%73%65%63%75%72%65/%77%77%77%2E%65%62%61%79%2E%63%6F%6D/", "http://168.188.99.26/.secure/www.ebay.com/"},
		{"8", "http://195.127.0.11/uploads/%20%20%20%20/.verify/.eBaysecure=updateuserdataxplimnbqmn-xplmvalidateinfoswqpcmlx=hgplmcx/", "http://195.127.0.11/uploads/%20%20%20%20/.verify/.eBaysecure=updateuserdataxplimnbqmn-xplmvalidateinfoswqpcmlx=hgplmcx/"},
		{"9", "http://host%23.com/%257Ea%2521b%2540c%2523d%2524e%25f%255E00%252611%252A22%252833%252944_55%252B", "http://host%23.com/~a!b@c%23d$e%25f^00&11*22(33)44_55+"},
		{"10", "http://3279880203/blah", "http://195.127.0.11/blah"},
		{"11", "http://www.google.com/blah/..", "http://www.google.com/"},
		{"12", "www.google.com/", "http://www.google.com/"},
		{"13", "www.google.com", "http://www.google.com/"},
		{"14", "http://www.evil.com/blah#frag", "http://www.evil.com/blah"},
		{"15", "http://www.GOOgle.com/", "http://www.google.com/"},
		{"16", "http://www.google.com.../", "http://www.google.com/"},
		{"17", "http://www.google.com/foo\tbar\rbaz\n2", "http://www.google.com/foobarbaz2"},
		{"18", "http://www.google.com/q?", "http://www.google.com/q?"},
		{"19", "http://www.google.com/q?r?", "http://www.google.com/q?r?"},
		{"20", "http://www.google.com/q?r?s", "http://www.google.com/q?r?s"},
		{"21", "http://evil.com/foo#bar#baz", "http://evil.com/foo"},
		{"22", "http://evil.com/foo;", "http://evil.com/foo;"},
		{"23", "http://evil.com/foo?bar;", "http://evil.com/foo?bar;"},
		{"24", "http://\x01\x80.com/", "http://%01%80.com/"},
		{"25", "http://notrailingslash.com", "http://notrailingslash.com/"},
		{"26", "http://www.gotaport.com:1234/", "http://www.gotaport.com/"},
		{"27", "  http://www.google.com/  ", "http://www.google.com/"},
		{"28", "http:// leadingspace.com/", "http://%20leadingspace.com/"},
		{"29", "http://%20leadingspace.com/", "http://%20leadingspace.com/"},
		{"30", "%20leadingspace.com/", "http://%20leadingspace.com/"},
		{"31", "https://www.securesite.com/", "https://www.securesite.com/"},
		{"32", "http://host.com/ab%23cd", "http://host.com/ab%23cd"},
		{"33", "http://host.com//twoslashes?more//slashes", "http://host.com/twoslashes?more//slashes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GoogleSafeBrowsing().Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.String() != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Exercise every written normalization step, including interactions absent
// from the 33 published examples. These expectations follow the rules, rather
// than treating the old profile's output as the reference implementation.
func TestGoogleSafeBrowsingRules(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"literal controls", "http://Ex\tAmple.\rorg/a\nb?x=\ty\r\n#gone", "http://example.org/ab?x=y"},
		{"escaped controls", "http://example.org/%2509%250d%250a?q=%2509%250d%250a", "http://example.org/%09%0D%0A?q=%09%0D%0A"},
		{"fragment before decoding", "http://example.org/a%2523b?q=%2523#gone%2525", "http://example.org/a%23b?q=%23"},
		{"IDNA", "http://BÜCHER.example", "http://xn--bcher-kva.example/"},
		{"nested IDNA", "http://b%25C3%25BCcher.example/", "http://xn--bcher-kva.example/"},
		{"IDNA before dot cleanup", "http://%25E3%2580%2582EXAMPLE%25E3%2580%2582%25E3%2580%2582COM%25E3%2580%2582/", "http://example.com/"},
		{"IDNA before IPv4", "http://%25E3%2580%25820x7f%25E3%2580%2582%25E3%2580%25821%25E3%2580%2582/", "http://127.0.0.1/"},
		{"host dots", "http://..ExAmple...org.../", "http://example.org/"},
		{"IPv4 after host dots", "http://.0x7f..1.../", "http://127.0.0.1/"},
		{"octal IPv4", "http://0177.1/", "http://127.0.0.1/"},
		{"hex IPv4", "http://0x7f000001/", "http://127.0.0.1/"},
		{"integer IPv4", "http://2130706433/", "http://127.0.0.1/"},
		{"short IPv4", "http://127.1/", "http://127.0.0.1/"},
		{"nested IPv4", "http://%2530%2578%2537%2566.1/", "http://127.0.0.1/"},
		{"lowercase after decoding", "http://%2545XAMPLE.COM/", "http://example.com/"},
		{"lowercase recovered host", "http://EXAMPLE%23.COM/", "http://example%23.com/"},
		{"host punctuation", "http://!foo_bar~.org/", "http://!foo_bar~.org/"},
		{"decoded parent", "http://example.org/a/%252e%252e/b", "http://example.org/b"},
		{"mixed parents", "http://example.org/a/%252e%252e/../b", "http://example.org/b"},
		{"decoded dot", "http://example.org/a/%252e/b", "http://example.org/a/b"},
		{"decoded trailing parent", "http://example.org/a/%252e%252e", "http://example.org/"},
		{"decoded slashes", "http://example.org/a%252f%252fb", "http://example.org/a/b"},
		{"dots before slash collapse", "http://example.org/a//../b", "http://example.org/a/b"},
		{"decoded slash before parent", "http://example.org/a/%252f../b", "http://example.org/a/b"},
		{"query path unchanged", "http://example.org/a//b?q=/./a//../b", "http://example.org/a/b?q=/./a//../b"},
		{"decoded query path unchanged", "http://example.org/?q=%252f.%252f.%252e%252f%252f", "http://example.org/?q=/./..//"},
		{"decoded query separators", "http://example.org/?q=%2526%253d%252b", "http://example.org/?q=&=+"},
		{"literal plus", "http://example.org/a+b?q=a+b%252B", "http://example.org/a+b?q=a+b+"},
		{"spaces", "http://example.org/a b?q=a b", "http://example.org/a%20b?q=a%20b"},
		{"arbitrary bytes", "http://example.org/%2500%257f%2580%25ff?q=%25ff", "http://example.org/%00%7F%80%FF?q=%FF"},
		{"malformed escapes", "http://example.org/%GG%?q=%GG%2", "http://example.org/%25GG%25?q=%25GG%252"},
		{"present empty query", "http://example.org/?#gone", "http://example.org/?"},
		{"remove port", "http://example.org:8443/", "http://example.org/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := GoogleSafeBrowsing()
			u, err := p.Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if u.String() != tt.want {
				t.Fatalf("Parse(%q) = %q, want %q", tt.input, u, tt.want)
			}
			p.Canonicalize(u)
			if u.String() != tt.want {
				t.Fatalf("second canonicalization = %q, want %q", u, tt.want)
			}
		})
	}
}

func TestGoogleSafeBrowsingMinimalEscaping(t *testing.T) {
	// Unlike general URL normalization, the matching representation exposes
	// decoded delimiters and does not form-decode '+' or protect '&' and '='.
	p := GoogleSafeBrowsing()
	u, err := p.Parse("http://example%252F.org/a%253Fb%2523c%255Cd?q=%2526%253D")
	if err != nil {
		t.Fatal(err)
	}
	if u.Hostname() != "example/.org" || u.Pathname() != "/a?b%23c\\d" || u.Query() != "q=&=" {
		t.Fatalf("matching components: host=%q path=%q query=%q", u.Hostname(), u.Pathname(), u.Query())
	}
	var raw strings.Builder
	var want strings.Builder
	const hex = "0123456789ABCDEF"
	for i := range 256 {
		b := byte(i)
		raw.WriteByte('%')
		raw.WriteByte(hex[b>>4])
		raw.WriteByte(hex[b&15])
		if b <= 32 || b >= 127 || b == '#' || b == '%' {
			want.WriteByte('%')
			want.WriteByte(hex[b>>4])
			want.WriteByte(hex[b&15])
		} else {
			want.WriteByte(b)
		}
	}
	u, err = p.Parse("http://example.org/?" + raw.String())
	if err != nil {
		t.Fatal(err)
	}
	if u.Query() != want.String() {
		t.Fatalf("escaped query bytes = %q, want %q", u.Query(), want.String())
	}
}

func TestGoogleSafeBrowsingExistingURL(t *testing.T) {
	p := GoogleSafeBrowsing()
	u, err := p.Parser().Parse("http://.0x7f..1.../a/%252e%252e/b?q=%2526%253D#fragment")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	p.Canonicalize(u)
	if u.String() != "http://127.0.0.1/b?q=&=" || !u.IsIPv4() || u.IsIPv6() || u.SearchParams() != params || params.Get("q") != "" || !params.Has("") {
		t.Fatalf("canonical URL/form/IP metadata: %v, %v", u, params)
	}
	u, err = p.ParseRef("http://example.org/a/%252e%252e/dir/", "../b")
	if err != nil || u.String() != "http://example.org/b" {
		t.Fatalf("ParseRef: %v, %v", u, err)
	}
	// Ordinary URL parsing still performs its WHATWG path normalization.
	u, err = url.Parse("http://example.org/a//../b")
	if err != nil || u.Pathname() != "/a/b" {
		t.Fatalf("standard parser changed: %v, %v", u, err)
	}
}
