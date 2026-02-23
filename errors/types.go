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

type kind uint8

const (
	domainToASCII kind = iota
	domainToUnicode
	domainInvalidCodePoint
	hostInvalidCodePoint
	ipv4EmptyPart
	ipv4TooManyParts
	ipv4NonNumericPart
	ipv4NonDecimalPart
	ipv4OutOfRangePart
	ipv6Unclosed
	ipv6InvalidCompression
	ipv6TooManyPieces
	ipv6MultipleCompression
	ipv6InvalidCodePoint
	ipv6TooFewPieces
	ipv4InIPv6TooManyPieces
	ipv4InIPv6InvalidCodePoint
	ipv4InIPv6OutOfRangePart
	ipv4InIPv6TooFewParts
	invalidURLUnit
	specialSchemeMissingFollowingSolidus
	missingSchemeNonRelativeURL
	invalidReverseSolidus
	invalidCredentials
	hostMissing
	portMissing
	portOutOfRange
	portInvalid
	fileInvalidWindowsDriveLetter
	fileInvalidWindowsDriveLetterHost
)

var (
	ErrDomainToASCII   = validationError(domainToASCII, "unicode ToASCII records an error or returns the empty string")
	ErrDomainToUnicode = validationError(domainToUnicode, "unicode ToUnicode records an error")

	ErrDomainInvalidCodePoint     = validationError(domainInvalidCodePoint, "the host contains a forbidden domain code point")
	ErrHostInvalidCodePoint       = validationError(hostInvalidCodePoint, "an opaque host (in a URL that is not special) contains a forbidden host code point")
	ErrIPv4EmptyPart              = validationError(ipv4EmptyPart, "an IPv4 address ends with a U+002E (.)")
	ErrIPv4TooManyParts           = validationError(ipv4TooManyParts, "an IPv4 address has more than four parts")
	ErrIPv4NonNumericPart         = validationError(ipv4NonNumericPart, "an IPv4 address contains a non-numeric part")
	ErrIPv4NonDecimalPart         = validationError(ipv4NonDecimalPart, "the IPv4 address contains numbers expressed using hexadecimal or octal digits")
	ErrIPv4OutOfRangePart         = validationError(ipv4OutOfRangePart, "an IPv4 address contains a part that is greater than 255")
	ErrIPv6Unclosed               = validationError(ipv6Unclosed, "an IPv6 address is missing the closing U+005D (])")
	ErrIPv6InvalidCompression     = validationError(ipv6InvalidCompression, "an IPv6 address begins with improper compression")
	ErrIPv6TooManyPieces          = validationError(ipv6TooManyPieces, "an IPv6 address has more than eight pieces")
	ErrIPv6MultipleCompression    = validationError(ipv6MultipleCompression, "an IPv6 address contains multiple instances of '::'")
	ErrIPv6InvalidCodePoint       = validationError(ipv6InvalidCodePoint, "an IPv6 address contains a code point that is neither an ASCII hex digit nor a U+003A (:). Or it unexpectedly ends")
	ErrIPv6TooFewPieces           = validationError(ipv6TooFewPieces, "an uncompressed IPv6 address contains fewer than 8 pieces")
	ErrIPv4InIPv6TooManyPieces    = validationError(ipv4InIPv6TooManyPieces, "an IPv4 address is found in an IPv6 address, but the IPv6 address has more than 6 pieces")
	ErrIPv4InIPv6InvalidCodePoint = validationError(ipv4InIPv6InvalidCodePoint, "an IPv4 address is found in an IPv6 address and one of the following is true: 1. An IPv4 part is empty or contains a non-ASCII digit. 2. An IPv4 part contains a leading 0. 3. There are too many IPv4 parts")
	ErrIPv4InIPv6OutOfRangePart   = validationError(ipv4InIPv6OutOfRangePart, "an IPv4 address is found in an IPv6 address and one of the IPv4 parts is greater than 255")
	ErrIPv4InIPv6TooFewParts      = validationError(ipv4InIPv6TooFewParts, "an IPv4 address is found in an IPv6 address and there are too few IPv4 parts")

	ErrInvalidURLUnit                       = validationError(invalidURLUnit, "a code point is found that is not a URL unit")
	ErrSpecialSchemeMissingFollowingSolidus = validationError(specialSchemeMissingFollowingSolidus, "the input’s scheme is not followed by '//'")
	ErrMissingSchemeNonRelativeURL          = validationError(missingSchemeNonRelativeURL, "the input is missing a scheme, because it does not begin with an ASCII alpha, and either no base URL was provided or the base URL cannot be used as a base URL because it has an opaque path")
	ErrInvalidReverseSolidus                = validationError(invalidReverseSolidus, "the URL has a special scheme and it uses U+005C (\\) instead of U+002F (/)")
	ErrInvalidCredentials                   = validationError(invalidCredentials, "the input includes credentials")
	ErrHostMissing                          = validationError(hostMissing, "the input has a special scheme, but does not contain a host")
	ErrPortMissing                          = validationError(portMissing, "the input has a hostname followed by a ':' but is missing the port number")
	ErrPortOutOfRange                       = validationError(portOutOfRange, "the input's port is outside the range [0-65535]")
	ErrPortInvalid                          = validationError(portInvalid, "the input's port is not a number")
	ErrFileInvalidWindowsDriveLetter        = validationError(fileInvalidWindowsDriveLetter, "the input is a relative-URL string that starts with a Windows drive letter and the base URL’s scheme is 'file'")
	ErrFileInvalidWindowsDriveLetterHost    = validationError(fileInvalidWindowsDriveLetterHost, "a file: URL’s host is a Windows drive letter")
)
