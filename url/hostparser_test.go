package url

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
)

var (
	endsInANumberSink          bool
	strictASCIIValidSink       bool
	endsInANumberBytesInput    = []byte("example.0x10.")
	strictASCIIValidBytesInput = []byte("www.example.com")
)

func Test_parser_parseHost(t *testing.T) {
	type args struct {
		input    string
		isOpaque bool
	}
	tests := []struct {
		name                  string
		args                  args
		failOnValidationError bool
		want                  string
		wantErr               bool
	}{
		{"1-1", args{input: "EXAMPLE.COM", isOpaque: false}, false, "example.com", false},
		{"1-2", args{input: "EXAMPLE.COM", isOpaque: true}, false, "EXAMPLE.COM", false},
		{"2-1", args{input: "example%2Ecom", isOpaque: false}, false, "example.com", false},
		{"2-2", args{input: "example%2Ecom", isOpaque: true}, false, "example%2Ecom", false},
		{"3-1", args{input: "faß.example", isOpaque: false}, false, "xn--fa-hia.example", false},
		{"3-2", args{input: "faß.example", isOpaque: true}, false, "fa%C3%9F.example", false},
		{"4-1", args{input: "0", isOpaque: false}, false, "0.0.0.0", false},
		{"4-2", args{input: "0", isOpaque: true}, false, "0", false},
		{"5-1", args{input: "%30", isOpaque: false}, false, "0.0.0.0", false},
		{"5-2", args{input: "%30", isOpaque: true}, false, "%30", false},
		{"6-1", args{input: "0x", isOpaque: false}, false, "0.0.0.0", false},
		{"6-2", args{input: "0x", isOpaque: true}, false, "0x", false},
		{"7-1", args{input: "0xffffffff", isOpaque: false}, false, "255.255.255.255", false},
		{"7-2", args{input: "0xffffffff", isOpaque: true}, false, "0xffffffff", false},
		{"8-1", args{input: "[0:0::1]", isOpaque: false}, false, "[::1]", false},
		{"8-2", args{input: "[0:0::1]", isOpaque: true}, false, "[::1]", false},
		{"9-1", args{input: "[0:0::1%5d]", isOpaque: false}, false, "", true},
		{"9-2", args{input: "[0:0::1%5d]", isOpaque: true}, false, "", true},
		{"10-1", args{input: "[0:0::1%31]", isOpaque: false}, false, "", true},
		{"10-2", args{input: "[0:0::1%31]", isOpaque: true}, false, "", true},
		{"11-1", args{input: "09", isOpaque: false}, false, "09", true},
		{"11-2", args{input: "09", isOpaque: true}, false, "09", false},
		{"12-1", args{input: "example.255", isOpaque: false}, false, "example.255", true},
		{"12-2", args{input: "example.255", isOpaque: true}, false, "example.255", false},
		{"13-1", args{input: "example^example", isOpaque: false}, false, "", true},
		{"13-2", args{input: "example^example", isOpaque: true}, false, "", true},
		{"14-1", args{input: "bad%hostname", isOpaque: false}, true, "", true},
		{"14-2", args{input: "bad%hostname", isOpaque: true}, true, "", true},
		{"15-1", args{input: "bad\\:hostname", isOpaque: false}, true, "", true},
		{"15-2", args{input: "bad\\:hostname", isOpaque: true}, true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{FailOnValidationError: tt.failOnValidationError}

			got, err := p.parseHost(&Url{}, []byte(tt.args.input), tt.args.isOpaque)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseHost() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHostCopyOnWriteAndReturnedStringOwnership(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "borrowed lowercase ASCII", input: "example.com", want: "example.com"},
		{name: "copied mixed-case ASCII", input: "EXAMPLE.COM", want: "example.com"},
		{name: "copied percent-encoded ASCII", input: "example%2Ecom", want: "example.com"},
		{name: "borrowed IDN input", input: "faß.example", want: "xn--fa-hia.example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.input)
			original := append([]byte(nil), input...)

			host, err := newDefaultParser().parseHost(&Url{}, input, false)
			if err != nil {
				t.Fatal(err)
			}
			if host != tt.want {
				t.Fatalf("parseHost(%q) = %q, want %q", tt.input, host, tt.want)
			}
			if string(input) != string(original) {
				t.Fatalf("parseHost mutated input: got %q, want %q", input, original)
			}

			for i := range input {
				input[i] = 'x'
			}
			if host != tt.want {
				t.Fatalf("returned host changed with input buffer: got %q, want %q", host, tt.want)
			}
		})
	}
}

func TestParseHost_IDNAValidation(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		report     bool
		fail       bool
		want       string
		wantErr    bool
		wantErrors int
	}{
		{
			name:       "ASCII strict warning relaxed success",
			input:      "a.b-.c",
			report:     true,
			want:       "a.b-.c",
			wantErrors: 1,
		},
		{
			name:       "ASCII STD3 warning relaxed success",
			input:      "a_b.c",
			report:     true,
			want:       "a_b.c",
			wantErrors: 1,
		},
		{
			name:       "ASCII empty label warning relaxed success",
			input:      "a..b",
			report:     true,
			want:       "a..b",
			wantErrors: 1,
		},
		{
			name:       "ASCII DNS length warning relaxed success",
			input:      strings.Repeat("a", 64) + ".example",
			report:     true,
			want:       strings.Repeat("a", 64) + ".example",
			wantErrors: 1,
		},
		{
			name:   "valid ASCII",
			input:  "example.com",
			report: true,
			want:   "example.com",
		},
		{
			name:       "non-ASCII strict warning relaxed success",
			input:      "ä.b-.c",
			report:     true,
			want:       "xn--4ca.b-.c",
			wantErrors: 1,
		},
		{
			name:   "valid non-ASCII",
			input:  "faß.example",
			report: true,
			want:   "xn--fa-hia.example",
		},
		{
			name:       "non-ASCII strict and relaxed failure",
			input:      "ä.\u200d.example",
			report:     true,
			wantErr:    true,
			wantErrors: 1,
		},
		{
			name:   "reporting disabled",
			input:  "a.b-.c",
			want:   "a.b-.c",
			report: false,
		},
		{
			name:    "fail on strict warning",
			input:   "a.b-.c",
			fail:    true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{
				ReportValidationErrors: tt.report,
				FailOnValidationError:  tt.fail,
			}
			u := &Url{}

			got, err := p.parseHost(u, []byte(tt.input), false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHost() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseHost() = %q, want %q", got, tt.want)
			}
			if len(u.ValidationErrors()) != tt.wantErrors {
				t.Fatalf("ValidationErrors() = %v, want %d error(s)", u.ValidationErrors(), tt.wantErrors)
			}
			for _, err := range u.ValidationErrors() {
				if !errors.Is(err, ErrDomainToASCII) {
					t.Errorf("ValidationErrors() contains %v, want %v", err, ErrDomainToASCII)
				}
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrDomainToASCII) {
				t.Errorf("parseHost() error = %v, want %v", err, ErrDomainToASCII)
			}
		})
	}
}

func TestIsStrictASCIIValid(t *testing.T) {
	label63 := strings.Repeat("a", 63)
	domain253 := strings.Join([]string{label63, label63, label63, strings.Repeat("a", 61)}, ".")

	tests := []struct {
		input string
		want  bool
	}{
		{"a", true},
		{"example.com", true},
		{"a-b.example", true},
		{label63, true},
		{domain253, true},
		{"", false},
		{".", false},
		{".example", false},
		{"example.", true},
		{"example..com", false},
		{"-example", false},
		{"example-", false},
		{"ab--c", false},
		{"a_b.example", false},
		{"a!b.example", false},
		{strings.Repeat("a", 64), false},
		{domain253 + "a", false},
		{domain253 + ".", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isStrictASCIIValid([]byte(tt.input)); got != tt.want {
				t.Errorf("isStrictASCIIValid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsStrictASCIIValid_MatchesIDNA(t *testing.T) {
	check := func(t *testing.T, domain string) {
		t.Helper()
		if !isASCIIWithoutACEPrefix([]byte(domain)) {
			return
		}
		_, err := domainParserToASCII(domain, true)
		want := err == nil
		if got := isStrictASCIIValid([]byte(domain)); got != want {
			t.Fatalf("isStrictASCIIValid(%q) = %v, domainParserToASCII error = %v", domain, got, err)
		}
	}

	// Cover every non-forbidden ASCII code point as a one-byte label.
	for b := byte(0); b < 128; b++ {
		if forbiddenDomainASCII[b] {
			continue
		}
		lower := b
		if 'A' <= lower && lower <= 'Z' {
			lower |= 0x20
		}
		check(t, string([]byte{lower}))
	}

	// Exercise combinations and length boundaries deterministically.
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789-._!\"$&'()*+,;=`{}~"
	rng := rand.New(rand.NewSource(1))
	for range 10_000 {
		buf := make([]byte, rng.Intn(280))
		for i := range buf {
			buf[i] = alphabet[rng.Intn(len(alphabet))]
		}
		check(t, string(buf))
	}
}

func TestDomainParserToASCII_PreservesTrailingRootLabel(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"example.", "example."},
		{"example\u3002", "example."},
		{"example\uff0e", "example."},
		{"example\uff61", "example."},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := domainParserToASCII(tt.input, true)
			if err != nil {
				t.Fatalf("domainParserToASCII(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("domainParserToASCII(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsStrictASCIIValid_ZeroAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		strictASCIIValidSink = isStrictASCIIValid(strictASCIIValidBytesInput)
	})
	if allocs != 0 {
		t.Fatalf("isStrictASCIIValid allocated %v times, want 0", allocs)
	}
}

func TestEndsInANumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"example.com", false},
		{"example.com.", false},
		{"", false},
		{".", false},
		{"example..", false},
		{"1..", false},
		{"example.1", true},
		{"example.1.", true},
		{".1", true},
		{"255", true},
		{"256", true},
		{"077", true},
		{"0x10", true},
		{"0x", true},
		{"09", true},
		{"example.0xg", false},
		{"example.08x", false},
		{"example.+1", false},
		{strings.Repeat("9", 100), true},
		{"0x" + strings.Repeat("f", 100), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := endsInANumber(tt.input); got != tt.want {
				t.Errorf("endsInANumber(string(%q)) = %v, want %v", tt.input, got, tt.want)
			}
			if got := endsInANumber([]byte(tt.input)); got != tt.want {
				t.Errorf("endsInANumber([]byte(%q)) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEndsInANumber_ZeroAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		endsInANumberSink = endsInANumber("example.0x10.")
		endsInANumberSink = endsInANumber(endsInANumberBytesInput)
	})
	if allocs != 0 {
		t.Fatalf("endsInANumber allocated %v times, want 0", allocs)
	}
}

func BenchmarkParseHostIDNA(b *testing.B) {
	benchmarks := []struct {
		name   string
		domain string
		report bool
	}{
		{"ASCII/reporting", "www.example.com", true},
		{"ASCII/no-reporting", "www.example.com", false},
		{"ASCII-warning/reporting", "a.b-.c", true},
		{"ASCII-warning/no-reporting", "a.b-.c", false},
		{"IDN/reporting", "www.Googlé.com", true},
		{"IDN/no-reporting", "www.Googlé.com", false},
		{"IDN-warning/reporting", "ä.b-.c", true},
		{"IDN-warning/no-reporting", "ä.b-.c", false},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			p := &parser{ReportValidationErrors: bm.report}
			input := []byte(bm.domain)
			for b.Loop() {
				if _, err := p.parseHost(nil, input, false); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
