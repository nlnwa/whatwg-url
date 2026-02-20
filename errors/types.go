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

var (
	ErrDomainToASCII   = validationError(DomainToASCII)
	ErrDomainToUnicode = validationError(DomainToUnicode)

	ErrDomainInvalidCodePoint     = validationError(DomainInvalidCodePoint)
	ErrHostInvalidCodePoint       = validationError(HostInvalidCodePoint)
	ErrIPv4EmptyPart              = validationError(IPv4EmptyPart)
	ErrIPv4TooManyParts           = validationError(IPv4TooManyParts)
	ErrIPv4NonNumericPart         = validationError(IPv4NonNumericPart)
	ErrIPv4NonDecimalPart         = validationError(IPv4NonDecimalPart)
	ErrIPv4OutOfRangePart         = validationError(IPv4OutOfRangePart)
	ErrIPv6Unclosed               = validationError(IPv6Unclosed)
	ErrIPv6InvalidCompression     = validationError(IPv6InvalidCompression)
	ErrIPv6TooManyPieces          = validationError(IPv6TooManyPieces)
	ErrIPv6MultipleCompression    = validationError(IPv6MultipleCompression)
	ErrIPv6InvalidCodePoint       = validationError(IPv6InvalidCodePoint)
	ErrIPv6TooFewPieces           = validationError(IPv6TooFewPieces)
	ErrIPv4InIPv6TooManyPieces    = validationError(IPv4InIPv6TooManyPieces)
	ErrIPv4InIPv6InvalidCodePoint = validationError(IPv4InIPv6InvalidCodePoint)
	ErrIPv4InIPv6OutOfRangePart   = validationError(IPv4InIPv6OutOfRangePart)
	ErrIPv4InIPv6TooFewParts      = validationError(IPv4InIPv6TooFewParts)

	ErrInvalidURLUnit                       = validationError(InvalidURLUnit)
	ErrSpecialSchemeMissingFollowingSolidus = validationError(SpecialSchemeMissingFollowingSolidus)
	ErrMissingSchemeNonRelativeURL          = validationError(MissingSchemeNonRelativeURL)
	ErrInvalidReverseSolidus                = validationError(InvalidReverseSolidus)
	ErrInvalidCredentials                   = validationError(InvalidCredentials)
	ErrHostMissing                          = validationError(HostMissing)
	ErrPortMissing                          = validationError(PortMissing)
	ErrPortOutOfRange                       = validationError(PortOutOfRange)
	ErrPortInvalid                          = validationError(PortInvalid)
	ErrFileInvalidWindowsDriveLetter        = validationError(FileInvalidWindowsDriveLetter)
	ErrFileInvalidWindowsDriveLetterHost    = validationError(FileInvalidWindowsDriveLetterHost)
	ErrInvalidEncoding                      = validationError(InvalidEncoding)
)
