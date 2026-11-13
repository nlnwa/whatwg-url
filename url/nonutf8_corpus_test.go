//go:build long

package url_test

import (
	"bufio"
	neturl "net/url"
	"os"
	"strings"
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestNonUTF8Corpus(t *testing.T) {
	file, err := os.Open("../testdata/non-utf8/non_utf8_urls_cc-main-2017-30.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		input := scanner.Text()
		if input == "" {
			t.Fatalf("line %d: empty URL", line)
		}
		u, err := url.Parse(input)
		if err != nil {
			t.Fatalf("line %d: Parse(%q): %v", line, input, err)
		}
		path, query := nonUTF8Components(t, line, input)
		if path == "" {
			path = "/" // Parsing normalizes an empty HTTP path.
		}
		for _, component := range []struct{ name, original, serialized string }{
			{"path", path, u.Pathname()},
			{"query", query, u.Query()},
		} {
			want := unescapeNonUTF8Component(t, line, component.original)
			got := unescapeNonUTF8Component(t, line, component.serialized)
			if got != want {
				t.Fatalf("line %d: %s bytes = %q, want %q", line, component.name, got, want)
			}
		}
		serialized := u.String()
		again, err := url.Parse(serialized)
		if err != nil {
			t.Fatalf("line %d: reparse %q: %v", line, serialized, err)
		}
		if got := again.String(); got != serialized {
			t.Fatalf("line %d: reparse = %q, want %q", line, got, serialized)
		}
		if got := url.ParseQuery(u.Query()).String(); got != u.Query() {
			t.Fatalf("line %d: Query round-trip = %q, want %q", line, got, u.Query())
		}
		params := u.SearchParams()
		_ = params.Get("q")
		_ = params.GetAll("q")
		_ = params.Has("q")
		_ = params.String()
		if got := u.String(); got != serialized {
			t.Fatalf("line %d: reading SearchParams changed URL to %q, want %q", line, got, serialized)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read corpus after line %d: %v", line, err)
	}
	if line == 0 {
		t.Fatal("empty non-UTF-8 corpus")
	}
}

// Split the fixture's absolute HTTP(S) URLs without parsing or normalizing them.
func nonUTF8Components(t *testing.T, line int, raw string) (path, query string) {
	t.Helper()
	_, rest, found := strings.Cut(raw, "://")
	if !found {
		t.Fatalf("line %d: expected an absolute HTTP(S) URL, got %q", line, raw)
	}
	start := strings.IndexAny(rest, "/?#")
	if start < 0 {
		return "", ""
	}
	rest, _, _ = strings.Cut(rest[start:], "#")
	path, query, _ = strings.Cut(rest, "?")
	return
}

func unescapeNonUTF8Component(t *testing.T, line int, raw string) string {
	t.Helper()
	// Decode URL bytes without applying form decoding. PathUnescape preserves
	// literal '+' in both path and query, as ordinary WHATWG URL parsing does.
	// Only form decoding, such as SearchParams, converts '+' to a space.
	decoded, err := neturl.PathUnescape(raw)
	if err != nil {
		t.Fatalf("line %d: percent-decode %q: %v", line, raw, err)
	}
	return decoded
}
