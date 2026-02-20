package internal

import (
	"errors"
	"testing"

	werrors "github.com/nationallibraryofnorway/whatwg-url/errors"
)

func TestParse_RecordsValidationWarning(t *testing.T) {
	p := NewParser()

	u, err := p.Parse(" https://example.org ")
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("Parse returned nil URL")
	}

	verrs := u.ValidationErrors()
	if len(verrs) == 0 {
		t.Fatal("expected validation errors to be recorded")
	}

	var found bool
	for _, verr := range verrs {
		if errors.Is(verr, werrors.ErrInvalidURLUnit) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %v validation error to be recorded", werrors.ErrInvalidURLUnit)
	}
}

func TestParse_ExplicitFailure_DoesNotRecordValidationError(t *testing.T) {
	p := NewParser()
	u := newUrl()
	u.parser = p

	_, err := p.parse("1http", nil, u, StateSchemeStart)
	if err == nil {
		t.Fatal("expected parse failure")
	}

	var v *werrors.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !errors.Is(v, werrors.ErrInvalidURLUnit) {
		t.Fatalf("expected %v, got %v", werrors.ErrInvalidURLUnit, v)
	}

	if len(u.ValidationErrors()) != 0 {
		t.Fatal("expected explicit failure path to not record validation error on URL")
	}
}
