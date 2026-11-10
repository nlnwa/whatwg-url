package url

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPercentDecode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"one pass", "%41%2f%7E", "A/~"},
		{"plus is literal", "a+b", "a+b"},
		{"malformed", "%2%GG%", "%2%GG%"},
		{"one pass only", "%2525", "%25"},
		{"legacy bytes", "%00%FC%ff", "\x00\xfc\xff"},
		{"mixed UTF8 and legacy bytes", "%C3%BC%FC", "ü\xfc"},
		{"existing replacement", "%EF%BF%BD", "\ufffd"},
		{"truncated UTF8", "%E2%82", "\xe2\x82"},
		{"nested legacy escape", "%25FC", "%FC"},
		{"encoded plus", "a+b%2B", "a+b+"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PercentDecode(test.input); got != test.want {
				t.Fatalf("PercentDecode(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}

	if got := PercentDecode([]byte("%00%ff")); got != string([]byte{0, 0xff}) {
		t.Fatalf("PercentDecode byte input = %q", got)
	} else if utf8.ValidString(got) {
		t.Fatal("decoded invalid UTF-8 unexpectedly became valid")
	}
}

func TestPercentDecodeRepeated(t *testing.T) {
	if got := PercentDecodeRepeated("%25252541%2B+"); got != "A++" {
		t.Fatalf("PercentDecodeRepeated() = %q, want %q", got, "A++")
	}
	if got := PercentDecodeRepeated([]byte("%25GG")); got != "%GG" {
		t.Fatalf("PercentDecodeRepeated() malformed result = %q", got)
	}
}

func TestSafeBrowsingStylePercentNormalizationVectors(t *testing.T) {
	escapeSet := NewPercentEncodeSet(0x20, ' ', '#', '%')
	tests := map[string]string{
		"/%25%32%35":          "/%25",
		"/%25%32%35%25%32%35": "/%25%25",
		"/%2525252525252525":  "/%25",
		"/asdf%25%32%35asd":   "/asdf%25asd",
		"/%%%25%32%35asd%%":   "/%25%25%25asd%25%25",
	}
	for input, want := range tests {
		got := PercentEncode(PercentDecodeRepeated(input), escapeSet)
		if got != want {
			t.Errorf("normalizing %q = %q, want %q", input, got, want)
		}
	}
}

func TestRewriteComponents(t *testing.T) {
	u, err := Parse("https://example.org:8443/A%2fb?x=1#frag")
	if err != nil {
		t.Fatal(err)
	}

	var calls []string
	u.RewriteHostname(
		func(got *Url, value string) string {
			if got != u {
				t.Fatal("rewriter received a different URL")
			}
			calls = append(calls, value)
			return "127.0.0.1"
		},
		nil,
		func(_ *Url, value string) string {
			calls = append(calls, value)
			return "[::1]"
		},
	)
	if strings.Join(calls, ",") != "example.org,127.0.0.1" {
		t.Fatalf("rewrite order = %q", calls)
	}
	u.RewriteHostname(func(_ *Url, _ string) string { return "127.0.0.1" })
	if !u.IsIPv4() || u.IsIPv6() || u.Host() != "127.0.0.1:8443" {
		t.Fatalf("rewritten IPv4 metadata: host=%q ipv4=%v ipv6=%v", u.Host(), u.IsIPv4(), u.IsIPv6())
	}
	u.RewriteHostname(func(_ *Url, _ string) string { return "[::1]" })
	if !u.IsIPv6() || u.IsIPv4() || u.Host() != "[::1]:8443" {
		t.Fatalf("rewritten host metadata: host=%q ipv4=%v ipv6=%v", u.Host(), u.IsIPv4(), u.IsIPv6())
	}

	u.RewritePathname(func(_ *Url, path string) string { return strings.ToLower(path) })
	u.RewriteQuery(func(_ *Url, raw string, present bool) (string, bool) {
		return raw + "&bare", present
	})
	u.RewriteFragment(func(_ *Url, _ string, _ bool) (string, bool) { return "", false })
	if got, want := u.String(), "https://[::1]:8443/a%2fb?x=1&bare"; got != want {
		t.Fatalf("rewritten URL = %q, want %q", got, want)
	}
}

func TestRewriteHostnameWithoutHostIsSkipped(t *testing.T) {
	u, err := Parse("mailto:test@example.org")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	u.RewriteHostname(func(_ *Url, host string) string {
		called = true
		return host
	})
	if called {
		t.Fatal("hostname rewriter called for hostless URL")
	}
}

func TestRewritePathnamePreservesOpaque(t *testing.T) {
	u, err := Parse("data:Text/Plain,HELLO")
	if err != nil {
		t.Fatal(err)
	}
	if !u.OpaquePath() {
		t.Fatal("test URL should have an opaque path")
	}
	u.RewritePathname(func(_ *Url, path string) string { return strings.ToLower(path) })
	if !u.OpaquePath() || u.Pathname() != "text/plain,hello" {
		t.Fatalf("opaque rewrite: opaque=%v path=%q", u.OpaquePath(), u.Pathname())
	}
}

func TestRewriteOptionalComponentPresence(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"absent", "https://example.org/", "https://example.org/"},
		{"empty", "https://example.org/?#", "https://example.org/?#"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u, err := Parse(test.url)
			if err != nil {
				t.Fatal(err)
			}
			u.RewriteQuery(func(_ *Url, raw string, present bool) (string, bool) {
				if raw != "" || present != (test.name == "empty") {
					t.Fatalf("query callback got raw=%q present=%v", raw, present)
				}
				return raw, present
			})
			u.RewriteFragment(func(_ *Url, raw string, present bool) (string, bool) {
				if raw != "" || present != (test.name == "empty") {
					t.Fatalf("fragment callback got raw=%q present=%v", raw, present)
				}
				return raw, present
			})
			if got := u.String(); got != test.want {
				t.Fatalf("URL = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRewriteQuerySynchronizesSearchParamsIdentity(t *testing.T) {
	u, err := Parse("https://example.org/?before=1")
	if err != nil {
		t.Fatal(err)
	}
	params := u.SearchParams()
	u.RewriteQuery(func(_ *Url, _ string, _ bool) (string, bool) {
		return "q=a+b&flag&space=%20", true
	})
	if u.SearchParams() != params {
		t.Fatal("RewriteQuery replaced the SearchParams object")
	}
	if got := params.Get("q"); got != "a b" {
		t.Fatalf("synchronized q = %q, want %q", got, "a b")
	}
	if !params.Has("flag") || params.Get("space") != " " {
		t.Fatalf("synchronized params = %q", params.String())
	}
}

func TestRewriteComponentsPreserveNonUTF8Bytes(t *testing.T) {
	check := func(name, input, want string, opaque bool) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			u, err := Parse(input)
			if err != nil {
				t.Fatal(err)
			}
			if u.String() != want || u.OpaquePath() != opaque {
				t.Fatalf("parsed URL = %q, opaque=%v; want %q, opaque=%v", u, u.OpaquePath(), want, opaque)
			}
			clone := u.Clone()
			params := u.SearchParams()
			_ = params.String() // Reading the form view must not rewrite the query.
			u.RewritePathname(func(_ *Url, raw string) string { return raw })
			u.RewriteQuery(func(_ *Url, raw string, present bool) (string, bool) { return raw, present })
			if u.String() != want || u.OpaquePath() != opaque || u.SearchParams() != params || clone.String() != want {
				t.Fatalf("identity rewrites changed URL, path kind, SearchParams identity, or clone: %q", u)
			}
			reparsed, err := Parse(u.String())
			if err != nil {
				t.Fatal(err)
			}
			if reparsed.String() != want || reparsed.OpaquePath() != opaque {
				t.Fatalf("reparse = %q, opaque=%v; want %q, opaque=%v", reparsed, reparsed.OpaquePath(), want, opaque)
			}
		})
	}
	for _, tc := range []struct{ name, input string }{
		{"legacy fields", "/%FC?%FC=%FC"},
		{"escaped delimiters", "/%FC%2F%5C%3F%23?a%26b=c%3Dd%26%23%3F%2F"},
		{"literal plus", "/a+b%2Bc?q=a+b%2Bc"},
		{"allowed punctuation", "/%FC;$&=:@,?q=%FC;$:@,/?"},
		{"escaped punctuation", "/%FC%3B%24%26%3D%3A%40%2C?q=%FC%3B%24%3A%40%2C%2F%3F"},
		{"escaped query names", "/%FC?a%26b%3Dc=%FC"},
		{"nested percent", "/%25FC?q=%25FC&deep=%2525FC"},
		{"query structure", "/%FC?&q=%FC&q=%FD&&flag&empty=&=v&"},
		{"absent query", "/%FC"},
		{"present empty query", "/%FC?"},
		{"fragment", "/%FC?q=%FC#%FC"},
		{"malformed escapes", "/%FC%GG%?q=%FC%2"},
		{"ASCII spelling", "/%fc%2f%7f?q=%00%09%20%25%2b%26%3d+"},
		{"different path and query sets", "/%FC%22%3C%3E%3F%5E%60%7B%7D?q=%FC%22%3C%3E%3F%5E%60%7B%7D"},
	} {
		input := "https://example.org" + tc.input
		check(tc.name, input, input, false)
	}
	// The scheme determines backslash handling and special-query escaping.
	check("special URL", "https://example.org/%FC'\\x?q=%FC'", "https://example.org/%FC'/x?q=%FC%27", false)
	input := "custom://example.org/%FC'\\x%5Cy%2Fz?q=%FC'%27%5C"
	check("non-special hierarchical URL", input, input, false)
	input = "custom:%FC/%2f?q=%FD+%2B#%FE"
	check("opaque path", input, input, true)
	input = "https://user%FC:pass%FD@example.org:8443/%FC?q=%FC#%FE"
	check("other components", input, input, false)
}

func TestPercentHelpersPreserveAllBytes(t *testing.T) {
	input := make([]byte, 256)
	for i := range input {
		input[i] = byte(i)
	}
	set := NewPercentEncodeSet(0x20, ' ', '#', '%')
	encoded := PercentEncode(input, set)
	if got := PercentDecode(encoded); got != string(input) {
		t.Fatalf("decoded bytes = %x, want %x", got, input)
	}
	if got := PercentDecodeRepeated(PercentEncode(input, nil)); got != string(input) {
		t.Fatalf("repeatedly decoded bytes = %x, want %x", got, input)
	}
	for _, test := range []struct{ input, want string }{
		{"%25FC", "\xfc"},
		{"%2525FD", "\xfd"},
		{"%FC%00+%GG%2%", "\xfc\x00+%GG%2%"},
	} {
		if got := PercentDecodeRepeated(test.input); got != test.want {
			t.Errorf("PercentDecodeRepeated(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
