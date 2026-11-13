package canonicalizer

import (
	"strings"
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestRewriteRules(t *testing.T) {
	c := New(
		RewriteHostname(func(_ *url.Url, host string) string {
			return strings.TrimPrefix(host, "www.")
		}),
		RewritePathname(func(_ *url.Url, path string) string {
			if path == "/" {
				return path
			}
			return strings.TrimSuffix(path, "/")
		}),
		RewriteQueryString(func(_ *url.Url, query string, hasQuery bool) (string, bool) {
			if query == "" {
				return query, hasQuery
			}
			return strings.ToLower(query), hasQuery
		}),
	)

	u, err := c.Parse("http://www.Example.com/path/?A=1&B=2")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got := u.String()
	want := "http://example.com/path?a=1&b=2"
	if got != want {
		t.Fatalf("Parse() got = %q, want %q", got, want)
	}
}

func TestRewriteCallbacksRunInOrder(t *testing.T) {
	c := New(
		RewriteHostname(
			func(_ *url.Url, host string) string { return strings.TrimPrefix(host, "www.") },
			func(_ *url.Url, host string) string { return strings.ToLower(host) },
		),
		RewritePathname(
			func(_ *url.Url, path string) string { return strings.TrimSuffix(path, "/") },
			func(_ *url.Url, path string) string {
				if path == "" {
					return "/"
				}
				return path
			},
		),
		RewriteQueryString(
			func(_ *url.Url, query string, hasQuery bool) (string, bool) {
				return strings.ReplaceAll(query, "A=1", "a=1"), hasQuery
			},
			func(_ *url.Url, query string, hasQuery bool) (string, bool) {
				return strings.ReplaceAll(query, "a=1&B=2", "a=1&b=2"), hasQuery
			},
		),
	)

	u, err := c.Parse("http://WWW.Example.com/path/?A=1&B=2")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got := u.String()
	want := "http://example.com/path?a=1&b=2"
	if got != want {
		t.Fatalf("Parse() got = %q, want %q", got, want)
	}
}

func TestRewriteRulesRunInOrder(t *testing.T) {
	c := New(
		RewriteQueryString(func(_ *url.Url, query string, hasQuery bool) (string, bool) {
			return strings.ReplaceAll(query, "A=1", "a=2"), hasQuery
		}),
		RewriteQueryString(func(_ *url.Url, query string, hasQuery bool) (string, bool) {
			return strings.ReplaceAll(query, "a=2&B=2", "a=2&b=3"), hasQuery
		}),
	)

	u, err := c.Parse("http://example.com/?A=1&B=2")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got := u.String()
	want := "http://example.com/?a=2&b=3"
	if got != want {
		t.Fatalf("Parse() got = %q, want %q", got, want)
	}
}
