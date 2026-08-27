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

package url

import (
	"errors"
	"fmt"
)

func codePointDescription(r rune) string {
	return fmt.Sprintf("U+%04X (%q)", r, r)
}

func byteCodePointDescription(b byte) string {
	return codePointDescription(rune(b))
}

func invalidUTF8ByteDescription(b byte) string {
	return fmt.Sprintf("invalid UTF-8 byte 0x%02X", b)
}

func isValidationFailure(verr *ValidationError) bool {
	switch verr.Code() {
	case CodeDomainToASCII,
		CodeHostInvalidCodePoint,
		CodeIPv4TooManyParts,
		CodeIPv4NonNumericPart,
		CodeIPv6Unclosed,
		CodeIPv6InvalidCompression,
		CodeIPv6TooManyPieces,
		CodeIPv6MultipleCompression,
		CodeIPv6InvalidCodePoint,
		CodeIPv6TooFewPieces,
		CodeIPv4InIPv6TooManyPieces,
		CodeIPv4InIPv6InvalidCodePoint,
		CodeIPv4InIPv6OutOfRangePart,
		CodeIPv4InIPv6TooFewParts,
		CodeMissingSchemeNonRelativeURL,
		CodeHostMissing,
		CodePortOutOfRange,
		CodePortInvalid:
		return true
	default:
		return false
	}
}

// handleError handles an error according to the options set for the parser
func (p *parser) handleError(u *Url, err error) error {
	verr, ok := err.(*ValidationError)
	if !ok {
		return err
	}
	verr = cloneValidationError(verr)
	if u != nil && p.ReportValidationErrors {
		u.validationErrors = append(u.validationErrors, verr)
	}
	failure := isValidationFailure(verr)
	if failure || p.FailOnValidationError {
		return verr
	}
	return nil
}

// handleValidationWarning handles a validation error that does not make URL
// parsing fail unless FailOnValidationError is enabled.
func (p *parser) handleValidationWarning(u *Url, err error) error {
	verr, ok := err.(*ValidationError)
	if !ok {
		return err
	}
	verr = cloneValidationError(verr)
	if u != nil && p.ReportValidationErrors {
		u.validationErrors = append(u.validationErrors, verr)
	}
	if p.FailOnValidationError {
		return verr
	}
	return nil
}

// handleErrorWithDescription handles an error according to the options set for the parser
func (p *parser) handleErrorWithDescription(u *Url, err *ValidationError, description string) error {
	return p.handleWrappedError(u, err, errors.New(description))
}

// handleWrappedError handles an error according to the options set for the parser
func (p *parser) handleWrappedError(u *Url, err *ValidationError, cause error) error {
	err = validationErrorWithCause(err, cause)
	return p.handleError(u, err)
}

// handleErrorAndFail returns a failure error for parser states that explicitly terminate.
// Unlike handleError, it intentionally does not record validation state on URL.
func (p *parser) handleErrorAndFail(_ *Url, err *ValidationError) error {
	return cloneValidationError(err)
}
