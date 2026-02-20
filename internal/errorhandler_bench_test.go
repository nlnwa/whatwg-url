package internal

import (
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/errors"
)

func BenchmarkHandleError_NoReport_NoFail(b *testing.B) {
	p := &Parser{}
	u := &Url{}
	err := errors.ErrInvalidURLUnit

	b.ResetTimer()
	for b.Loop() {
		if got := p.handleError(u, err); got != nil {
			b.Fatal(got)
		}
	}
}

func BenchmarkHandleError_Report_NoFail(b *testing.B) {
	p := &Parser{ReportValidationErrors: true}
	err := errors.ErrInvalidURLUnit

	b.ResetTimer()
	for b.Loop() {
		u := &Url{}
		if got := p.handleError(u, err); got != nil {
			b.Fatal(got)
		}
	}
}

func BenchmarkHandleError_FailOnValidation(b *testing.B) {
	p := &Parser{FailOnValidationError: true}
	u := &Url{}
	err := errors.ErrInvalidURLUnit

	b.ResetTimer()
	for b.Loop() {
		if got := p.handleError(u, err); got == nil {
			b.Fatal("expected validation error")
		}
	}
}

func BenchmarkHandleErrorWithDescription_NoReport_NoFail(b *testing.B) {
	p := &Parser{}
	u := &Url{}

	b.ResetTimer()
	for b.Loop() {
		if got := p.handleErrorWithDescription(u, errors.ErrInvalidURLUnit, "%"); got != nil {
			b.Fatal(got)
		}
	}
}

func BenchmarkParse_InvalidHost_NoReport(b *testing.B) {
	p := NewParser()
	input := "http://exa[mple.com"

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.Parse(input)
	}
}

func BenchmarkParse_InvalidHost_Report(b *testing.B) {
	p := NewParser()
	p.ReportValidationErrors = true
	input := "http://exa[mple.com"

	b.ResetTimer()
	for b.Loop() {
		_, _ = p.Parse(input)
	}
}
