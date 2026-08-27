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
	stderrors "errors"
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
	switch verr.kind {
	case errDomainToASCII,
		errDomainInvalidCodePoint,
		errHostInvalidCodePoint,
		errIPv4TooManyParts,
		errIPv4NonNumericPart,
		errIPv4OutOfRangePart,
		errIPv6Unclosed,
		errIPv6InvalidCompression,
		errIPv6TooManyPieces,
		errIPv6MultipleCompression,
		errIPv6InvalidCodePoint,
		errIPv6TooFewPieces,
		errIPv4InIPv6TooManyPieces,
		errIPv4InIPv6InvalidCodePoint,
		errIPv4InIPv6OutOfRangePart,
		errIPv4InIPv6TooFewParts,
		errMissingSchemeNonRelativeURL,
		errHostMissing,
		errPortMissing,
		errPortOutOfRange,
		errPortInvalid:
		return true
	default:
		return false
	}
}

// handleError handles an error according to the options set for the parser
func (p *Parser) handleError(u *Url, err error) error {
	verr, ok := err.(*ValidationError)
	if !ok {
		return err
	}
	if u != nil && p.ReportValidationErrors {
		u.validationErrors = append(u.validationErrors, err)
	}
	failure := isValidationFailure(verr)
	if failure || p.FailOnValidationError {
		return err
	}
	return nil
}

// handleErrorWithDescription handles an error according to the options set for the parser
func (p *Parser) handleErrorWithDescription(u *Url, err *ValidationError, description string) error {
	return p.handleWrappedError(u, err, stderrors.New(description))
}

// handleWrappedError handles an error according to the options set for the parser
func (p *Parser) handleWrappedError(u *Url, err *ValidationError, cause error) error {
	err = err.Wrap(cause)
	return p.handleError(u, err)
}

// handleErrorAndFail returns a failure error for parser states that explicitly terminate.
// Unlike handleError, it intentionally does not record validation state on URL.
func (p *Parser) handleErrorAndFail(_ *Url, err *ValidationError) error {
	return err
}
