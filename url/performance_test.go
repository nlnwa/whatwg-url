/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package url

import (
	"errors"
	"testing"
)

var (
	benchmarkParseURLSink        *Url
	benchmarkHostSink            string
	benchmarkParseErrorSink      error
	benchmarkParseErrorTypeSink  *ParseError
	benchmarkValidationErrorSink *ValidationError
)

func BenchmarkParseOpaqueHost(b *testing.B) {
	benchmarks := []struct {
		name        string
		input       []byte
		configure   func(*parser)
		want        string
		wantFailure bool
	}{
		{name: "clean-ascii", input: []byte("www.example.com"), want: "www.example.com"},
		{name: "unicode-encoding", input: []byte("faß.example"), want: "fa%C3%9F.example"},
		{name: "malformed-percent-warning", input: []byte("bad%hostname"), want: "bad%hostname"},
		{
			name:  "invalid-utf8-recovery",
			input: []byte{'b', 'a', 'd', 0xff, '.', 'e', 'x', 'a', 'm', 'p', 'l', 'e'},
			configure: func(p *parser) {
				p.AcceptInvalidCodepoints = true
			},
			want: "bad%FF.example",
		},
		{name: "forbidden-code-point-failure", input: []byte("bad^hostname"), wantFailure: true},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			p := newDefaultParser()
			if benchmark.configure != nil {
				benchmark.configure(p)
			}
			got, err := p.parseHost(nil, benchmark.input, true)
			if (err != nil) != benchmark.wantFailure {
				b.Fatalf("parseHost(%q) error = %v, want failure %t", benchmark.input, err, benchmark.wantFailure)
			}
			if !benchmark.wantFailure && got != benchmark.want {
				b.Fatalf("parseHost(%q) = %q, want %q", benchmark.input, got, benchmark.want)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkHostSink, benchmarkParseErrorSink = p.parseHost(nil, benchmark.input, true)
			}
		})
	}
}

func BenchmarkParsePerformance(b *testing.B) {
	benchmarks := []struct {
		name            string
		input           string
		wantFailure     bool
		wantValidations int
	}{
		{name: "valid-lowercase-ascii", input: "https://www.example.com/path"},
		{name: "valid-mixed-case-ascii", input: "https://WWW.Example.COM/path"},
		{name: "valid-idn", input: "https://www.Googlé.com/path"},
		{name: "ascii-idna-warning", input: "https://a.b-.c/", wantValidations: 1},
		{name: "unicode-idna-warning", input: "https://ä.b-.c/", wantValidations: 1},
		{name: "multiple-warnings", input: " https://a.b-.c/ ", wantValidations: 2},
		{name: "ipv4-normalization", input: "https://127.1/", wantValidations: 1},
		{name: "invalid-port", input: "https://example.com:7z/", wantFailure: true, wantValidations: 1},
		{name: "invalid-ipv6", input: "https://[1::2::3]/", wantFailure: true, wantValidations: 1},
		{name: "warning-then-failure", input: " https://example.org:7z ", wantFailure: true, wantValidations: 2},
	}
	reportingModes := []struct {
		name    string
		enabled bool
	}{
		{name: "reporting-enabled", enabled: true},
		{name: "reporting-disabled", enabled: false},
	}

	for _, benchmark := range benchmarks {
		for _, reporting := range reportingModes {
			b.Run(benchmark.name+"/"+reporting.name, func(b *testing.B) {
				p := NewParser(WithReportValidationErrors(reporting.enabled))
				assertParseBenchmarkResult(b, p, benchmark.input, benchmark.wantFailure, benchmark.wantValidations, reporting.enabled)

				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					benchmarkParseURLSink, benchmarkParseErrorSink = p.Parse(benchmark.input)
				}
			})
		}
	}
}

func BenchmarkParserErrorDiscovery(b *testing.B) {
	_, parserFailure := NewParser().Parse("https://example.com:7z/")
	_, ipv6Failure := parseIPv6([]byte("1::2::3"), nil)
	_, idnaWarning, _ := domainParser("ä.b-.c", false)

	for _, benchmark := range []struct {
		name       string
		err        error
		parseError bool
	}{
		{name: "parse-error", err: parserFailure, parseError: true},
		{name: "ipv6-validation", err: ipv6Failure},
		{name: "idna-validation", err: idnaWarning},
	} {
		b.Run(benchmark.name, func(b *testing.B) {
			if benchmark.parseError {
				b.Run("errors.As", func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						var target *ParseError
						if !errors.As(benchmark.err, &target) {
							b.Fatal("ParseError not found")
						}
						benchmarkParseErrorTypeSink = target
					}
				})
				b.Run("errors.AsType", func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						target, ok := errors.AsType[*ParseError](benchmark.err)
						if !ok {
							b.Fatal("ParseError not found")
						}
						benchmarkParseErrorTypeSink = target
					}
				})
				return
			}

			b.Run("errors.As", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var target *ValidationError
					if !errors.As(benchmark.err, &target) {
						b.Fatal("ValidationError not found")
					}
					benchmarkValidationErrorSink = target
				}
			})
			b.Run("errors.AsType", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					target, ok := errors.AsType[*ValidationError](benchmark.err)
					if !ok {
						b.Fatal("ValidationError not found")
					}
					benchmarkValidationErrorSink = target
				}
			})
		})
	}
}

func assertParseBenchmarkResult(b *testing.B, p Parser, input string, wantFailure bool, wantValidations int, reporting bool) {
	b.Helper()

	u, err := p.Parse(input)
	if (err != nil) != wantFailure {
		b.Fatalf("Parse(%q) error = %v, want failure %t", input, err, wantFailure)
	}

	if wantFailure {
		parseErr, ok := err.(*ParseError)
		if !ok {
			b.Fatalf("Parse(%q) error = %T, want *ParseError", input, err)
		}
		if got := len(parseErr.ValidationErrors()); got != wantValidations {
			b.Fatalf("Parse(%q) retained %d validation errors, want %d", input, got, wantValidations)
		}
		return
	}

	wantRetained := wantValidations
	if !reporting {
		wantRetained = 0
	}
	if got := len(u.ValidationErrors()); got != wantRetained {
		b.Fatalf("Parse(%q) retained %d validation errors, want %d", input, got, wantRetained)
	}
}
