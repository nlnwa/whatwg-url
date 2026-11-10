package url

import (
	"slices"
	"testing"
)

func TestRawQueryRoundTrip(t *testing.T) {
	tests := []string{
		"",
		"q=a+b",
		"space=%20&lower=%2f",
		"flag&empty=",
		"a=b=c",
		"&&a=1&&",
		"?literal-leading-question",
		"&&%FC=%FD&flag&empty=&a+b=%2B&%FC=%FE&",
		"x=%GG%2%&nested=%2525FD",
	}
	for _, raw := range tests {
		if got := ParseRawQuery(raw).String(); got != raw {
			t.Errorf("ParseRawQuery(%q).String() = %q", raw, got)
		}
	}
}

func TestRawQuerySort(t *testing.T) {
	query := ParseRawQuery("b=first&a=z&b=second&a=a")
	query.Sort()
	if got, want := query.String(), "a=z&a=a&b=first&b=second"; got != want {
		t.Fatalf("Sort() = %q, want %q", got, want)
	}

	query = ParseRawQuery("b=first&a=z&b=second&a=a&b")
	query.SortAbsolute()
	if got, want := query.String(), "a=a&a=z&b&b=first&b=second"; got != want {
		t.Fatalf("SortAbsolute() = %q, want %q", got, want)
	}
}

func TestRawQueryFilter(t *testing.T) {
	query := ParseRawQuery("keep=1&&drop=2&flag&keep=3")
	var seen []RawQueryParameter
	query.Filter(func(name, value string, hasEquals bool) bool {
		seen = append(seen, RawQueryParameter{name, value, hasEquals})
		return name != "drop" && name != ""
	})
	if got, want := query.String(), "keep=1&flag&keep=3"; got != want {
		t.Fatalf("Filter() = %q, want %q", got, want)
	}
	if !slices.Equal(seen, ParseRawQuery("keep=1&&drop=2&flag&keep=3")) {
		t.Fatalf("Filter callback saw %#v", seen)
	}
}

func TestRawQuerySortPreservesNonUTF8Bytes(t *testing.T) {
	query := ParseRawQuery("b=%FC&a=%FF&a=%FE&flag&empty=&")
	query.Sort()
	if got, want := query.String(), "&a=%FF&a=%FE&b=%FC&empty=&flag"; got != want {
		t.Fatalf("Sort() = %q, want %q", got, want)
	}
	query.SortAbsolute()
	if got, want := query.String(), "&a=%FE&a=%FF&b=%FC&empty=&flag"; got != want {
		t.Fatalf("SortAbsolute() = %q, want %q", got, want)
	}
}
