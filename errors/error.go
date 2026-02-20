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

package errors

type Code uint16

const (
	// IDNA
	DomainToASCII Code = iota
	DomainInvalidCodePoint
	DomainToUnicode

	// Host parsing
	HostInvalidCodePoint

	IPv4EmptyPart
	IPv4TooManyParts
	IPv4NonNumericPart
	IPv4NonDecimalPart
	IPv4OutOfRangePart

	IPv6Unclosed
	IPv6InvalidCompression
	IPv6TooManyPieces
	IPv6MultipleCompression
	IPv6InvalidCodePoint
	IPv6TooFewPieces

	IPv4InIPv6TooManyPieces
	IPv4InIPv6InvalidCodePoint
	IPv4InIPv6OutOfRangePart
	IPv4InIPv6TooFewParts

	// URL parsing
	InvalidURLUnit
	SpecialSchemeMissingFollowingSolidus
	MissingSchemeNonRelativeURL
	InvalidReverseSolidus
	InvalidCredentials
	HostMissing
	PortMissing
	PortOutOfRange
	PortInvalid
	FileInvalidWindowsDriveLetter
	FileInvalidWindowsDriveLetterHost

	InvalidEncoding // TODO not part of spec
)

func Message(c Code) string {
	switch c {

	case DomainToASCII:
		return "unicode ToASCII records an error or returns the empty string"

	case DomainInvalidCodePoint:
		return "the host contains a forbidden domain code point"

	case DomainToUnicode:
		return "unicode ToUnicode records an error"

	case HostInvalidCodePoint:
		return "an opaque host (in a URL that is not special) contains a forbidden host code point"

	case IPv4EmptyPart:
		return "an IPv4 address ends with a U+002E (.)"
	case IPv4TooManyParts:
		return "an IPv4 address has more than four parts"
	case IPv4NonNumericPart:
		return "an IPv4 address contains a non-numeric part"
	case IPv4NonDecimalPart:
		return "the IPv4 address contains numbers expressed using hexadecimal or octal digits"
	case IPv4OutOfRangePart:
		return "an IPv4 address contains a part that is greater than 255"

	case IPv6Unclosed:
		return "an IPv6 address is missing the closing U+005D (])"
	case IPv6InvalidCompression:
		return "an IPv6 address begins with improper compression"
	case IPv6TooManyPieces:
		return "an IPv6 address has more than eight pieces"
	case IPv6MultipleCompression:
		return "an IPv6 address contains multiple instances of '::'"
	case IPv6InvalidCodePoint:
		return "an IPv6 address contains a code point that is neither an ASCII hex digit nor a U+003A (:). Or it unexpectedly ends"
	case IPv6TooFewPieces:
		return "an uncompressed IPv6 address contains fewer than 8 pieces"

	case IPv4InIPv6TooManyPieces:
		return "an IPv4 address is found in an IPv6 address, but the IPv6 address has more than 6 pieces"
	case IPv4InIPv6InvalidCodePoint:
		return "an IPv4 address is found in an IPv6 address and one of the following is true: 1. An IPv4 part is empty or contains a non-ASCII digit. 2. An IPv4 part contains a leading 0. 3. There are too many IPv4 parts"
	case IPv4InIPv6OutOfRangePart:
		return "an IPv4 address is found in an IPv6 address and one of the IPv4 parts is greater than 255"
	case IPv4InIPv6TooFewParts:
		return "an IPv4 address is found in an IPv6 address and there are too few IPv4 parts"

	case InvalidURLUnit:
		return "a code point is found that is not a URL unit"
	case SpecialSchemeMissingFollowingSolidus:
		return "the input’s scheme is not followed by '//'"
	case MissingSchemeNonRelativeURL:
		return "the input is missing a scheme, because it does not begin with an ASCII alpha, and either no base URL was provided or the base URL cannot be used as a base URL because it has an opaque path"
	case InvalidReverseSolidus:
		return "the URL has a special scheme and it uses U+005C (\\) instead of U+002F (/)"
	case InvalidCredentials:
		return "the input includes credentials"
	case HostMissing:
		return "the input has a special scheme, but does not contain a host"
	case PortMissing:
		return "the input has a hostname followed by a ':' but is missing the port number"
	case PortOutOfRange:
		return "the input's port is outside the range [0-65535]"
	case PortInvalid:
		return "the input's port is not a number"
	case FileInvalidWindowsDriveLetter:
		return "the input is a relative-URL string that starts with a Windows drive letter and the base URL’s scheme is 'file'"
	case FileInvalidWindowsDriveLetterHost:
		return "a file: URL’s host is a Windows drive letter"
	case InvalidEncoding:
		return "the input could not be interpreted using the specified encoding"

	default:
		return "validation error"
	}
}

func validationError(code Code) *ValidationError {
	return &ValidationError{code: code}
}

// ValidationError indicates that the url is not valid
type ValidationError struct {
	cause error
	code  Code
}

func (e *ValidationError) Error() string {
	msg := Message(e.code)
	if e.cause != nil {
		return msg + ": " + e.cause.Error()
	}
	return msg
}

func (e *ValidationError) Wrap(err error) *ValidationError {
	return &ValidationError{
		cause: err,
		code:  e.code,
	}
}

func (e *ValidationError) Code() Code {
	return e.code
}

func (e *ValidationError) Unwrap() error {
	return e.cause
}
