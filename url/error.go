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

import "errors"

// ValidationErrorCode is a stable WHATWG validation-error identifier.
type ValidationErrorCode string

// WHATWG validation-error codes.
const (
	CodeDomainToASCII                        ValidationErrorCode = "domain-to-ASCII"
	CodeDomainPercentEncoded                 ValidationErrorCode = "domain-percent-encoded"
	CodeHostInvalidCodePoint                 ValidationErrorCode = "host-invalid-code-point"
	CodeIPv4EmptyPart                        ValidationErrorCode = "IPv4-empty-part"
	CodeIPv4TooFewParts                      ValidationErrorCode = "IPv4-too-few-parts"
	CodeIPv4TooManyParts                     ValidationErrorCode = "IPv4-too-many-parts"
	CodeIPv4NonNumericPart                   ValidationErrorCode = "IPv4-non-numeric-part"
	CodeIPv4NonDecimalPart                   ValidationErrorCode = "IPv4-non-decimal-part"
	CodeIPv4OutOfRangePart                   ValidationErrorCode = "IPv4-out-of-range-part"
	CodeIPv4NonASCIIInput                    ValidationErrorCode = "IPv4-non-ASCII-input"
	CodeIPv6Unclosed                         ValidationErrorCode = "IPv6-unclosed"
	CodeIPv6InvalidCompression               ValidationErrorCode = "IPv6-invalid-compression"
	CodeIPv6TooManyPieces                    ValidationErrorCode = "IPv6-too-many-pieces"
	CodeIPv6MultipleCompression              ValidationErrorCode = "IPv6-multiple-compression"
	CodeIPv6InvalidCodePoint                 ValidationErrorCode = "IPv6-invalid-code-point"
	CodeIPv6TooFewPieces                     ValidationErrorCode = "IPv6-too-few-pieces"
	CodeIPv6PieceLeadingZero                 ValidationErrorCode = "IPv6-piece-leading-zero"
	CodeIPv4InIPv6TooManyPieces              ValidationErrorCode = "IPv4-in-IPv6-too-many-pieces"
	CodeIPv4InIPv6InvalidCodePoint           ValidationErrorCode = "IPv4-in-IPv6-invalid-code-point"
	CodeIPv4InIPv6OutOfRangePart             ValidationErrorCode = "IPv4-in-IPv6-out-of-range-part"
	CodeIPv4InIPv6TooFewParts                ValidationErrorCode = "IPv4-in-IPv6-too-few-parts"
	CodeInvalidURLUnit                       ValidationErrorCode = "invalid-URL-unit"
	CodeSpecialSchemeMissingFollowingSolidus ValidationErrorCode = "special-scheme-missing-following-solidus"
	CodeMissingSchemeNonRelativeURL          ValidationErrorCode = "missing-scheme-non-relative-URL"
	CodeInvalidReverseSolidus                ValidationErrorCode = "invalid-reverse-solidus"
	CodeInvalidCredentials                   ValidationErrorCode = "invalid-credentials"
	CodeHostMissing                          ValidationErrorCode = "host-missing"
	CodePortOutOfRange                       ValidationErrorCode = "port-out-of-range"
	CodePortInvalid                          ValidationErrorCode = "port-invalid"
	CodeFileInvalidWindowsDriveLetter        ValidationErrorCode = "file-invalid-Windows-drive-letter"
	CodeFileInvalidWindowsDriveLetterHost    ValidationErrorCode = "file-invalid-Windows-drive-letter-host"
)

// ValidationError represents one occurrence of a WHATWG validation error.
// Its state is immutable outside this package.
type ValidationError struct {
	cause  error
	code   ValidationErrorCode
	detail string
}

var validationErrorSentinels = make(map[*ValidationError]struct{}, 31)

// validationError creates and registers a package-level sentinel template.
func validationError(code ValidationErrorCode, detail string) *ValidationError {
	err := &ValidationError{code: code, detail: detail}
	validationErrorSentinels[err] = struct{}{}
	return err
}

func validationErrorWithCause(template *ValidationError, cause error) *ValidationError {
	if template == nil {
		return &ValidationError{cause: cause}
	}
	return &ValidationError{code: template.code, detail: template.detail, cause: cause}
}

// Code returns the WHATWG identifier for this validation error.
func (e *ValidationError) Code() ValidationErrorCode {
	if e == nil {
		return ""
	}
	return e.code
}

// Detail returns the human-readable description of this validation error.
func (e *ValidationError) Detail() string {
	if e == nil {
		return ""
	}
	return e.detail
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.detail + ": " + e.cause.Error()
	}
	return e.detail
}

// Is reports whether target is a validation error with the same non-empty code.
func (e *ValidationError) Is(target error) bool {
	t, ok := target.(*ValidationError)
	return ok && e != nil && t != nil && e.code != "" && t.code != "" && e.code == t.code
}

// Unwrap returns the genuine underlying cause, if any.
func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

var (
	ErrDomainToASCII                        = validationError(CodeDomainToASCII, "unicode ToASCII records an error or returns the empty string")
	ErrDomainPercentEncoded                 = validationError(CodeDomainPercentEncoded, "the input's host to be processed as a domain contains a percent-encoded byte")
	ErrHostInvalidCodePoint                 = validationError(CodeHostInvalidCodePoint, "an opaque host (in a URL that is not special) contains a forbidden host code point")
	ErrIPv4EmptyPart                        = validationError(CodeIPv4EmptyPart, "an IPv4 address ends with a U+002E (.)")
	ErrIPv4TooFewParts                      = validationError(CodeIPv4TooFewParts, "an IPv4 address has fewer than four parts")
	ErrIPv4TooManyParts                     = validationError(CodeIPv4TooManyParts, "an IPv4 address has more than four parts")
	ErrIPv4NonNumericPart                   = validationError(CodeIPv4NonNumericPart, "an IPv4 address contains a non-numeric part")
	ErrIPv4NonDecimalPart                   = validationError(CodeIPv4NonDecimalPart, "the IPv4 address contains numbers expressed using hexadecimal or octal digits")
	ErrIPv4OutOfRangePart                   = validationError(CodeIPv4OutOfRangePart, "an IPv4 address contains a part that is greater than 255")
	ErrIPv4NonASCIIInput                    = validationError(CodeIPv4NonASCIIInput, "an IPv4 address is derived from a non-ASCII string through IDNA processing")
	ErrIPv6Unclosed                         = validationError(CodeIPv6Unclosed, "an IPv6 address is missing the closing U+005D (])")
	ErrIPv6InvalidCompression               = validationError(CodeIPv6InvalidCompression, "an IPv6 address begins with improper compression")
	ErrIPv6TooManyPieces                    = validationError(CodeIPv6TooManyPieces, "an IPv6 address has more than eight pieces")
	ErrIPv6MultipleCompression              = validationError(CodeIPv6MultipleCompression, "an IPv6 address contains multiple instances of '::'")
	ErrIPv6InvalidCodePoint                 = validationError(CodeIPv6InvalidCodePoint, "an IPv6 address contains a code point that is neither an ASCII hex digit nor a U+003A (:). Or it unexpectedly ends")
	ErrIPv6TooFewPieces                     = validationError(CodeIPv6TooFewPieces, "an uncompressed IPv6 address contains fewer than 8 pieces")
	ErrIPv6PieceLeadingZero                 = validationError(CodeIPv6PieceLeadingZero, "an IPv6 address piece contains a leading U+0030 (0)")
	ErrIPv4InIPv6TooManyPieces              = validationError(CodeIPv4InIPv6TooManyPieces, "an IPv4 address is found in an IPv6 address, but the IPv6 address has more than 6 pieces")
	ErrIPv4InIPv6InvalidCodePoint           = validationError(CodeIPv4InIPv6InvalidCodePoint, "an IPv4 address is found in an IPv6 address and one of the following is true: 1. An IPv4 part is empty or contains a non-ASCII digit. 2. An IPv4 part contains a leading 0. 3. There are too many IPv4 parts")
	ErrIPv4InIPv6OutOfRangePart             = validationError(CodeIPv4InIPv6OutOfRangePart, "an IPv4 address is found in an IPv6 address and one of the IPv4 parts is greater than 255")
	ErrIPv4InIPv6TooFewParts                = validationError(CodeIPv4InIPv6TooFewParts, "an IPv4 address is found in an IPv6 address and there are too few IPv4 parts")
	ErrInvalidURLUnit                       = validationError(CodeInvalidURLUnit, "a code point is found that is not a URL unit")
	ErrSpecialSchemeMissingFollowingSolidus = validationError(CodeSpecialSchemeMissingFollowingSolidus, "the input’s scheme is not followed by '//'")
	ErrMissingSchemeNonRelativeURL          = validationError(CodeMissingSchemeNonRelativeURL, "the input is missing a scheme, because it does not begin with an ASCII alpha, and either no base URL was provided or the base URL cannot be used as a base URL because it has an opaque path")
	ErrInvalidReverseSolidus                = validationError(CodeInvalidReverseSolidus, "the URL has a special scheme and it uses U+005C (\\) instead of U+002F (/)")
	ErrInvalidCredentials                   = validationError(CodeInvalidCredentials, "the input includes credentials")
	ErrHostMissing                          = validationError(CodeHostMissing, "the input has a special scheme, but does not contain a host")
	ErrPortOutOfRange                       = validationError(CodePortOutOfRange, "the input's port is outside the range [0-65535]")
	ErrPortInvalid                          = validationError(CodePortInvalid, "the input's port is not a number")
	ErrFileInvalidWindowsDriveLetter        = validationError(CodeFileInvalidWindowsDriveLetter, "the input is a relative-URL string that starts with a Windows drive letter and the base URL’s scheme is 'file'")
	ErrFileInvalidWindowsDriveLetterHost    = validationError(CodeFileInvalidWindowsDriveLetterHost, "a file: URL’s host is a Windows drive letter")
)

func isValidationErrorSentinel(err *ValidationError) bool {
	_, ok := validationErrorSentinels[err]
	return ok
}

// ErrParseFailure identifies every parser failure.
var ErrParseFailure = errors.New("URL parse failure")

// ParseError represents a failed Parse, ParseRef, or BasicParse operation.
type ParseError struct {
	detail           string
	validationErrors []*ValidationError
}

func newParseError(detail string, validationErrors []*ValidationError) *ParseError {
	return &ParseError{
		detail:           detail,
		validationErrors: validationErrors,
	}
}

func (e *ParseError) Error() string {
	if e == nil || e.detail == "" {
		return ErrParseFailure.Error()
	}
	return ErrParseFailure.Error() + ": " + e.detail
}

// ValidationErrors returns a defensive copy of the validation errors associated
// with this parser failure.
func (e *ParseError) ValidationErrors() []*ValidationError {
	if e == nil {
		return nil
	}
	return append([]*ValidationError(nil), e.validationErrors...)
}

// Unwrap exposes the parse-failure sentinel and all associated validation errors.
func (e *ParseError) Unwrap() []error {
	if e == nil {
		return []error{ErrParseFailure}
	}
	errs := make([]error, 1, 1+len(e.validationErrors))
	errs[0] = ErrParseFailure
	for _, validationErr := range e.validationErrors {
		if validationErr != nil {
			errs = append(errs, validationErr)
		}
	}
	return errs
}
