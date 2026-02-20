/*
 * Copyright 2020 National Library of Norway.
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

package internal

import (
	gerrors "errors"

	"github.com/nationallibraryofnorway/whatwg-url/errors"
)

type errorUnwrapper interface {
	Unwrap() error
}

func isValidationFailure(verr *errors.ValidationError) bool {
	return verr.Is(errors.ErrDomainToASCII) ||
		verr.Is(errors.ErrDomainInvalidCodePoint) ||
		verr.Is(errors.ErrHostInvalidCodePoint) ||
		verr.Is(errors.ErrIPv4TooManyParts) ||
		verr.Is(errors.ErrIPv4NonNumericPart) ||
		verr.Is(errors.ErrIPv4OutOfRangePart) ||
		verr.Is(errors.ErrIPv6Unclosed) ||
		verr.Is(errors.ErrIPv6InvalidCompression) ||
		verr.Is(errors.ErrIPv6TooManyPieces) ||
		verr.Is(errors.ErrIPv6MultipleCompression) ||
		verr.Is(errors.ErrIPv6InvalidCodePoint) ||
		verr.Is(errors.ErrIPv6TooFewPieces) ||
		verr.Is(errors.ErrIPv4InIPv6TooManyPieces) ||
		verr.Is(errors.ErrIPv4InIPv6InvalidCodePoint) ||
		verr.Is(errors.ErrIPv4InIPv6OutOfRangePart) ||
		verr.Is(errors.ErrIPv4InIPv6TooFewParts) ||
		verr.Is(errors.ErrMissingSchemeNonRelativeURL) ||
		verr.Is(errors.ErrHostMissing) ||
		verr.Is(errors.ErrPortMissing) ||
		verr.Is(errors.ErrPortOutOfRange) ||
		verr.Is(errors.ErrPortInvalid) ||
		verr.Is(errors.ErrInvalidEncoding)
}

func asValidationError(err error) (*errors.ValidationError, bool) {
	if err == nil {
		return nil, false
	}
	if verr, ok := err.(*errors.ValidationError); ok {
		return verr, true
	}
	original := err
	for {
		u, ok := err.(errorUnwrapper)
		if !ok {
			return nil, false
		}
		err = u.Unwrap()
		if err == nil {
			return nil, false
		}
		if verr, ok := err.(*errors.ValidationError); ok {
			return verr, true
		}
		if err == original {
			return nil, false
		}
	}
}

// handleError handles an error according to the options set for the parser
func (p *Parser) handleError(u *Url, err error) error {
	verr, ok := asValidationError(err)
	if !ok {
		return err
	}
	if u != nil {
		u.validationErrors = append(u.validationErrors, verr)
	}
	failure := isValidationFailure(verr)
	if failure || p.FailOnValidationError {
		return verr
	}
	return nil
}

// handleErrorWithDescription handles an error according to the options set for the parser
func (p *Parser) handleErrorWithDescription(u *Url, err *errors.ValidationError, description string) error {
	return p.handleWrappedError(u, err, gerrors.New(description))
}

// handleWrappedError handles an error according to the options set for the parser
func (p *Parser) handleWrappedError(u *Url, err *errors.ValidationError, cause error) error {
	err = err.Wrap(cause)
	return p.handleError(u, err)
}

// handleErrorAndFail returns a failure error for parser states that explicitly terminate.
// Unlike handleError, it intentionally does not record validation state on URL.
func (p *Parser) handleErrorAndFail(_ *Url, err *errors.ValidationError) error {
	return err
}
