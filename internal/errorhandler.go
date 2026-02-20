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

func isValidationFailure(code errors.Code) bool {
	switch code {
	case errors.DomainToASCII,
		errors.DomainInvalidCodePoint,
		errors.HostInvalidCodePoint,
		errors.IPv4TooManyParts,
		errors.IPv4NonNumericPart,
		errors.IPv4OutOfRangePart,
		errors.IPv6Unclosed,
		errors.IPv6InvalidCompression,
		errors.IPv6TooManyPieces,
		errors.IPv6MultipleCompression,
		errors.IPv6InvalidCodePoint,
		errors.IPv6TooFewPieces,
		errors.IPv4InIPv6TooManyPieces,
		errors.IPv4InIPv6InvalidCodePoint,
		errors.IPv4InIPv6OutOfRangePart,
		errors.IPv4InIPv6TooFewParts,
		errors.MissingSchemeNonRelativeURL,
		errors.HostMissing,
		errors.PortMissing,
		errors.PortOutOfRange,
		errors.PortInvalid,
		errors.InvalidEncoding:
		return true
	default:
		return false
	}
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
	failure := isValidationFailure(verr.Code())
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

// handleErrorAndFail records a validation error and returns a failure error regardless of validation severity.
// Use this for parser states that explicitly terminate with failure while still reporting validation details.
func (p *Parser) handleErrorAndFail(u *Url, err *errors.ValidationError) error {
	if handled := p.handleError(u, err); handled != nil {
		return handled
	}
	return err
}
