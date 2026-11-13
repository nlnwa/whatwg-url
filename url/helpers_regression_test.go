package url

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"slices"
	"strings"
	"testing"
)

// A custom set for helper tests and benchmarks, independent of canonicalizer
// policy. Construct it once so allocations are outside measured operations.
var helperPercentEncodeSet = C0OrSpacePercentEncodeSet.Set('#', '%', '&', '=')

func referenceQueryTerm(p QueryParameter) string {
	if p.HasEquals {
		return p.Name + "=" + p.Value
	}
	return p.Name
}

func TestQuerySortReference(t *testing.T) {
	var terms Query
	for _, name := range []string{"", "a", "a!", "a+", "a%2f", "a=", "a?", "aa", "ab", "b", "\x00", "\xff", "\xffa"} {
		for _, value := range []string{"", "a", "z", "=", "&", "\x00\xff", "+%2f"} {
			for _, hasEquals := range []bool{false, true} {
				terms = append(terms, QueryParameter{name, value, hasEquals})
			}
		}
	}
	// Include arbitrary bytes and identical serialized terms with different
	// fields, so stable ordering and ignored bare-term values are observable.
	random := rand.New(rand.NewSource(42))
	for range 300 {
		name, value := make([]byte, random.Intn(8)), make([]byte, random.Intn(8))
		random.Read(name)
		random.Read(value)
		terms = append(terms, QueryParameter{string(name), string(value), random.Intn(2) == 0})
	}
	terms = append(terms, terms...)
	random.Shuffle(len(terms), func(i, j int) { terms[i], terms[j] = terms[j], terms[i] })
	for _, absolute := range []bool{false, true} {
		got, want := slices.Clone(terms), slices.Clone(terms)
		slices.SortStableFunc(want, func(a, b QueryParameter) int {
			if absolute {
				return strings.Compare(referenceQueryTerm(a), referenceQueryTerm(b))
			}
			return strings.Compare(a.Name, b.Name)
		})
		if absolute {
			got.SortAbsolute()
		} else {
			got.Sort()
		}
		if !slices.Equal(got, want) {
			t.Fatalf("sort absolute=%v differs from serialized reference", absolute)
		}
	}
	var serialized []string
	for _, term := range terms {
		serialized = append(serialized, referenceQueryTerm(term))
	}
	if got, want := terms.String(), strings.Join(serialized, "&"); got != want {
		t.Fatal("serialization differs from reference")
	}
}

// These deliberately allocating references preserve the previous helpers'
// byte semantics independently of the optimized scans and in-place decoder.
func referencePercentDecode(raw string, repeated bool) string {
	for {
		var out strings.Builder
		for i := 0; i < len(raw); i++ {
			if raw[i] == '%' && i+2 < len(raw) {
				if decoded, err := hex.DecodeString(raw[i+1 : i+3]); err == nil {
					out.Write(decoded)
					i += 2
					continue
				}
			}
			out.WriteByte(raw[i])
		}
		decoded := out.String()
		if !repeated || decoded == raw {
			return decoded
		}
		raw = decoded
	}
}

func referencePercentEncode(raw string, set *PercentEncodeSet) string {
	var out strings.Builder
	for i := 0; i < len(raw); i++ {
		if set == nil || set.ByteShouldBeEncoded(raw[i]) {
			out.WriteByte('%')
			out.WriteString(strings.ToUpper(hex.EncodeToString([]byte{raw[i]})))
		} else {
			out.WriteByte(raw[i])
		}
	}
	return out.String()
}

func TestPercentHelpersReference(t *testing.T) {
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
	}
	inputs := []string{string(allBytes), referencePercentEncode(string(allBytes), nil)}
	tokens := []string{"%", "%25", "%32", "%34", "%31", "4", "1", "%GG", "%ff", "%2", "A", "+"}
	for _, a := range tokens {
		for _, b := range tokens {
			for _, c := range tokens {
				inputs = append(inputs, a+b+c)
			}
		}
	}
	for _, raw := range inputs {
		for _, repeated := range []bool{false, true} {
			want := referencePercentDecode(raw, repeated)
			got := PercentDecode(raw)
			if repeated {
				got = PercentDecodeRepeated(raw)
			}
			if got != want {
				t.Fatalf("decode repeated=%v input=%q: got %q, want %q", repeated, raw, got, want)
			}
		}
		for _, set := range []*PercentEncodeSet{nil, NewPercentEncodeSet(0), PathPercentEncodeSet, helperPercentEncodeSet} {
			if got, want := PercentEncode(raw, set), referencePercentEncode(raw, set); got != want {
				t.Fatalf("encode %q: got %q, want %q", raw, got, want)
			}
		}
	}
}

func TestPercentHelpersOwnByteInputs(t *testing.T) {
	type namedBytes []byte
	type namedString string
	for _, raw := range []string{"plain+%GG", "%41%FF+", "%25252541", "%%341", "", "\x00\xff"} {
		for _, helper := range []func([]byte) string{
			func(b []byte) string { return PercentDecode(namedBytes(b)) },
			func(b []byte) string { return PercentDecodeRepeated(namedBytes(b)) },
			func(b []byte) string { return PercentEncode(namedBytes(b), PathPercentEncodeSet) },
			func(b []byte) string { return PercentEncode(namedBytes(b), nil) },
		} {
			input := []byte(raw)
			got := helper(input)
			if !bytes.Equal(input, []byte(raw)) {
				t.Fatalf("helper mutated input %q", raw)
			}
			want := strings.Clone(got)
			for i := range input {
				input[i] = 'X'
			}
			if got != want {
				t.Fatalf("result aliases mutable input %q", raw)
			}
		}
		if PercentDecodeRepeated(namedString(raw)) != referencePercentDecode(raw, true) {
			t.Fatal("named string input changed decoding")
		}
	}
}

func TestHelperAllocationBudgets(t *testing.T) {
	query := benchmarkQuery(97)
	for _, absolute := range []bool{false, true} {
		if allocs := testing.AllocsPerRun(100, func() {
			slices.Reverse(query)
			if absolute {
				query.SortAbsolute()
			} else {
				query.Sort()
			}
		}); allocs != 0 {
			t.Errorf("sorting absolute=%v allocated %v times", absolute, allocs)
		}
	}
	if allocs := testing.AllocsPerRun(100, func() { helperStringSink = query.String() }); allocs > 1 {
		t.Errorf("serialization allocated %v times", allocs)
	}
	for _, helper := range []func() string{
		func() string { return PercentEncode("unchanged+%GG", PathPercentEncodeSet) },
		func() string { return PercentDecode("unchanged+%GG") },
		func() string { return PercentDecodeRepeated("unchanged+%GG") },
	} {
		if allocs := testing.AllocsPerRun(100, func() { helperStringSink = helper() }); allocs != 0 {
			t.Errorf("unchanged string allocated %v times", allocs)
		}
	}
	for _, depth := range []int{1, 8, 64, 512} {
		for _, suffix := range []string{"41", "25", "FF"} {
			input := "%" + strings.Repeat("25", depth) + suffix
			if allocs := testing.AllocsPerRun(100, func() {
				helperStringSink = PercentEncode(PercentDecodeRepeated(input), helperPercentEncodeSet)
			}); allocs > 3 {
				t.Errorf("nested depth=%d allocated %v times", depth, allocs)
			}
		}
	}
}
