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

func validationError(kind errorKind, msg string) *ValidationError {
	return &ValidationError{kind: kind, msg: msg}
}

// ValidationError indicates that the url is not valid
type ValidationError struct {
	cause error
	kind  errorKind
	msg   string
}

func (e *ValidationError) Error() string {
	msg := e.msg
	if e.cause != nil {
		return msg + ": " + e.cause.Error()
	}
	return msg
}

func (e *ValidationError) Wrap(err error) *ValidationError {
	return &ValidationError{
		cause: err,
		kind:  e.kind,
		msg:   e.msg,
	}
}

func (e *ValidationError) Is(target error) bool {
	t, ok := target.(*ValidationError)
	if !ok {
		return false
	}
	return e.kind == t.kind
}

func (e *ValidationError) Unwrap() error {
	return e.cause
}

type errorKind uint8

const (
	errDomainToASCII errorKind = iota
	errDomainPercentEncoded
	errDomainToUnicode
	errDomainInvalidCodePoint
	errHostInvalidCodePoint
	errIPv4EmptyPart
	errIPv4TooManyParts
	errIPv4NonNumericPart
	errIPv4NonDecimalPart
	errIPv4OutOfRangePart
	errIPv4NonASCIIInput
	errIPv6Unclosed
	errIPv6InvalidCompression
	errIPv6TooManyPieces
	errIPv6MultipleCompression
	errIPv6InvalidCodePoint
	errIPv6TooFewPieces
	errIPv4InIPv6TooManyPieces
	errIPv4InIPv6InvalidCodePoint
	errIPv4InIPv6OutOfRangePart
	errIPv4InIPv6TooFewParts
	errInvalidURLUnit
	errSpecialSchemeMissingFollowingSolidus
	errMissingSchemeNonRelativeURL
	errInvalidReverseSolidus
	errInvalidCredentials
	errHostMissing
	errPortMissing
	errPortOutOfRange
	errPortInvalid
	errFileInvalidWindowsDriveLetter
	errFileInvalidWindowsDriveLetterHost
)

var (
	ErrDomainToASCII                        = validationError(errDomainToASCII, "unicode ToASCII records an error or returns the empty string")
	ErrDomainPercentEncoded                 = validationError(errDomainPercentEncoded, "the input's host to be processed as a domain contains a percent-encoded byte")
	ErrDomainToUnicode                      = validationError(errDomainToUnicode, "unicode ToUnicode records an error")
	ErrDomainInvalidCodePoint               = validationError(errDomainInvalidCodePoint, "the host contains a forbidden domain code point")
	ErrHostInvalidCodePoint                 = validationError(errHostInvalidCodePoint, "an opaque host (in a URL that is not special) contains a forbidden host code point")
	ErrIPv4EmptyPart                        = validationError(errIPv4EmptyPart, "an IPv4 address ends with a U+002E (.)")
	ErrIPv4TooManyParts                     = validationError(errIPv4TooManyParts, "an IPv4 address has more than four parts")
	ErrIPv4NonNumericPart                   = validationError(errIPv4NonNumericPart, "an IPv4 address contains a non-numeric part")
	ErrIPv4NonDecimalPart                   = validationError(errIPv4NonDecimalPart, "the IPv4 address contains numbers expressed using hexadecimal or octal digits")
	ErrIPv4OutOfRangePart                   = validationError(errIPv4OutOfRangePart, "an IPv4 address contains a part that is greater than 255")
	ErrIPv4NonASCIIInput                    = validationError(errIPv4NonASCIIInput, "an IPv4 address is derived from a non-ASCII string through IDNA processing")
	ErrIPv6Unclosed                         = validationError(errIPv6Unclosed, "an IPv6 address is missing the closing U+005D (])")
	ErrIPv6InvalidCompression               = validationError(errIPv6InvalidCompression, "an IPv6 address begins with improper compression")
	ErrIPv6TooManyPieces                    = validationError(errIPv6TooManyPieces, "an IPv6 address has more than eight pieces")
	ErrIPv6MultipleCompression              = validationError(errIPv6MultipleCompression, "an IPv6 address contains multiple instances of '::'")
	ErrIPv6InvalidCodePoint                 = validationError(errIPv6InvalidCodePoint, "an IPv6 address contains a code point that is neither an ASCII hex digit nor a U+003A (:). Or it unexpectedly ends")
	ErrIPv6TooFewPieces                     = validationError(errIPv6TooFewPieces, "an uncompressed IPv6 address contains fewer than 8 pieces")
	ErrIPv4InIPv6TooManyPieces              = validationError(errIPv4InIPv6TooManyPieces, "an IPv4 address is found in an IPv6 address, but the IPv6 address has more than 6 pieces")
	ErrIPv4InIPv6InvalidCodePoint           = validationError(errIPv4InIPv6InvalidCodePoint, "an IPv4 address is found in an IPv6 address and one of the following is true: 1. An IPv4 part is empty or contains a non-ASCII digit. 2. An IPv4 part contains a leading 0. 3. There are too many IPv4 parts")
	ErrIPv4InIPv6OutOfRangePart             = validationError(errIPv4InIPv6OutOfRangePart, "an IPv4 address is found in an IPv6 address and one of the IPv4 parts is greater than 255")
	ErrIPv4InIPv6TooFewParts                = validationError(errIPv4InIPv6TooFewParts, "an IPv4 address is found in an IPv6 address and there are too few IPv4 parts")
	ErrInvalidURLUnit                       = validationError(errInvalidURLUnit, "a code point is found that is not a URL unit")
	ErrSpecialSchemeMissingFollowingSolidus = validationError(errSpecialSchemeMissingFollowingSolidus, "the input’s scheme is not followed by '//'")
	ErrMissingSchemeNonRelativeURL          = validationError(errMissingSchemeNonRelativeURL, "the input is missing a scheme, because it does not begin with an ASCII alpha, and either no base URL was provided or the base URL cannot be used as a base URL because it has an opaque path")
	ErrInvalidReverseSolidus                = validationError(errInvalidReverseSolidus, "the URL has a special scheme and it uses U+005C (\\) instead of U+002F (/)")
	ErrInvalidCredentials                   = validationError(errInvalidCredentials, "the input includes credentials")
	ErrHostMissing                          = validationError(errHostMissing, "the input has a special scheme, but does not contain a host")
	ErrPortOutOfRange                       = validationError(errPortOutOfRange, "the input's port is outside the range [0-65535]")
	ErrPortInvalid                          = validationError(errPortInvalid, "the input's port is not a number")
	ErrFileInvalidWindowsDriveLetter        = validationError(errFileInvalidWindowsDriveLetter, "the input is a relative-URL string that starts with a Windows drive letter and the base URL’s scheme is 'file'")
	ErrFileInvalidWindowsDriveLetterHost    = validationError(errFileInvalidWindowsDriveLetterHost, "a file: URL’s host is a Windows drive letter")
)
