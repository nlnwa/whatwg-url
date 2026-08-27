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

import "github.com/nationallibraryofnorway/whatwg-url/internal"

var (
	ErrDomainToASCII                        error = internal.ErrDomainToASCII
	ErrDomainToUnicode                      error = internal.ErrDomainToUnicode
	ErrDomainInvalidCodePoint               error = internal.ErrDomainInvalidCodePoint
	ErrHostInvalidCodePoint                 error = internal.ErrHostInvalidCodePoint
	ErrIPv4EmptyPart                        error = internal.ErrIPv4EmptyPart
	ErrIPv4TooManyParts                     error = internal.ErrIPv4TooManyParts
	ErrIPv4NonNumericPart                   error = internal.ErrIPv4NonNumericPart
	ErrIPv4NonDecimalPart                   error = internal.ErrIPv4NonDecimalPart
	ErrIPv4OutOfRangePart                   error = internal.ErrIPv4OutOfRangePart
	ErrIPv6Unclosed                         error = internal.ErrIPv6Unclosed
	ErrIPv6InvalidCompression               error = internal.ErrIPv6InvalidCompression
	ErrIPv6TooManyPieces                    error = internal.ErrIPv6TooManyPieces
	ErrIPv6MultipleCompression              error = internal.ErrIPv6MultipleCompression
	ErrIPv6InvalidCodePoint                 error = internal.ErrIPv6InvalidCodePoint
	ErrIPv6TooFewPieces                     error = internal.ErrIPv6TooFewPieces
	ErrIPv4InIPv6TooManyPieces              error = internal.ErrIPv4InIPv6TooManyPieces
	ErrIPv4InIPv6InvalidCodePoint           error = internal.ErrIPv4InIPv6InvalidCodePoint
	ErrIPv4InIPv6OutOfRangePart             error = internal.ErrIPv4InIPv6OutOfRangePart
	ErrIPv4InIPv6TooFewParts                error = internal.ErrIPv4InIPv6TooFewParts
	ErrInvalidURLUnit                       error = internal.ErrInvalidURLUnit
	ErrSpecialSchemeMissingFollowingSolidus error = internal.ErrSpecialSchemeMissingFollowingSolidus
	ErrMissingSchemeNonRelativeURL          error = internal.ErrMissingSchemeNonRelativeURL
	ErrInvalidReverseSolidus                error = internal.ErrInvalidReverseSolidus
	ErrInvalidCredentials                   error = internal.ErrInvalidCredentials
	ErrHostMissing                          error = internal.ErrHostMissing
	ErrPortMissing                          error = internal.ErrPortMissing
	ErrPortOutOfRange                       error = internal.ErrPortOutOfRange
	ErrPortInvalid                          error = internal.ErrPortInvalid
	ErrFileInvalidWindowsDriveLetter        error = internal.ErrFileInvalidWindowsDriveLetter
	ErrFileInvalidWindowsDriveLetterHost    error = internal.ErrFileInvalidWindowsDriveLetterHost
)
