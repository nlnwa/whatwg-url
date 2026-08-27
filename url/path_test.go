package url

import "testing"

var pathStringSink string

func TestPathShortenSerialized(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		scheme string
		want   string
	}{
		{name: "empty", value: "", scheme: "http", want: ""},
		{name: "root", value: "/", scheme: "http", want: ""},
		{name: "segment", value: "/a", scheme: "http", want: ""},
		{name: "trailing empty segment", value: "/a/", scheme: "http", want: "/a"},
		{name: "multiple segments", value: "/a/b", scheme: "http", want: "/a"},
		{name: "leading empty segment", value: "//host", scheme: "web+demo", want: "/"},
		{name: "file drive", value: "/C:", scheme: "file", want: "/C:"},
		{name: "file drive child", value: "/C:/a", scheme: "file", want: "/C:"},
		{name: "unnormalized file drive", value: "/C|", scheme: "file", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &path{value: tt.value}
			p.shortenPath(tt.scheme)
			if got := p.String(); got != tt.want {
				t.Fatalf("shortenPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathAppendAndCollapse(t *testing.T) {
	p := &path{}
	p.addSegmentBytes([]byte("a"), false)
	p.addSegmentBytes(nil, false)
	p.addSegmentBytes([]byte("b"), false)
	if got := p.String(); got != "/a//b" {
		t.Fatalf("serialized path = %q, want %q", got, "/a//b")
	}

	p.init()
	p.addSegmentBytes(nil, false)
	p.addSegmentBytes(nil, true)
	p.addSegmentBytes([]byte("a"), true)
	p.addSegmentBytes(nil, false)
	p.addSegmentBytes([]byte("b"), true)
	if got := p.String(); got != "/a/b" {
		t.Fatalf("collapsed path = %q, want %q", got, "/a/b")
	}
}

func TestPathCloneUsesCopyOnWrite(t *testing.T) {
	original := &path{value: "/a/b"}
	clone := original.clone()
	clone.addSegmentBytes([]byte("c"), false)

	if got := original.String(); got != "/a/b" {
		t.Fatalf("original path changed to %q", got)
	}
	if got := clone.String(); got != "/a/b/c" {
		t.Fatalf("clone path = %q, want %q", got, "/a/b/c")
	}
}

func TestPathSerializedProperties(t *testing.T) {
	tests := []struct {
		value              string
		disambiguation     bool
		normalizedFileRoot bool
	}{
		{value: ""},
		{value: "/"},
		{value: "//", disambiguation: true},
		{value: "//not-a-host", disambiguation: true},
		{value: "/C:", normalizedFileRoot: true},
		{value: "/C:/path", normalizedFileRoot: true},
		{value: "/C|/path"},
		{value: "/long:/path"},
	}

	for _, tt := range tests {
		p := &path{value: tt.value}
		if got := p.needsDisambiguation(); got != tt.disambiguation {
			t.Errorf("path %q needsDisambiguation() = %t, want %t", tt.value, got, tt.disambiguation)
		}
		if got := p.firstSegmentIsNormalizedWindowsDriveLetter(); got != tt.normalizedFileRoot {
			t.Errorf("path %q firstSegmentIsNormalizedWindowsDriveLetter() = %t, want %t", tt.value, got, tt.normalizedFileRoot)
		}
	}
}

func TestPathnameDoesNotAllocateAfterParse(t *testing.T) {
	for _, input := range []string{
		"https://example.com/a/b/c/",
		"data:text/plain,opaque-path",
	} {
		u, err := testParser.Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", input, err)
		}

		allocs := testing.AllocsPerRun(1000, func() {
			pathStringSink = u.Pathname()
		})
		if allocs != 0 {
			t.Errorf("Pathname() for %q allocated %v times, want 0", input, allocs)
		}
	}
}

func TestSerializedPathRelativeResolutionAndClone(t *testing.T) {
	base, err := testParser.Parse("https://example.com/a/b/c")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := base.Parse("../d//e/")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resolved.Pathname(), "/a/d//e/"; got != want {
		t.Fatalf("resolved pathname = %q, want %q", got, want)
	}
	if got, want := base.Pathname(), "/a/b/c"; got != want {
		t.Fatalf("base pathname changed to %q, want %q", got, want)
	}

	clone := resolved.Clone()
	clone.SetPathname("/x/y")
	if got, want := clone.Pathname(), "/x/y"; got != want {
		t.Fatalf("clone pathname = %q, want %q", got, want)
	}
	if got, want := resolved.Pathname(), "/a/d//e/"; got != want {
		t.Fatalf("resolved pathname changed to %q, want %q", got, want)
	}
}
