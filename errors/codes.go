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

import (
	"fmt"
)

// ErrorCode is data type of error codes for different kind of errors
type ErrorCode int32

// Validation errors
const (
	IllegalCodePoint ErrorCode = iota + 100
	InvalidPercentEncoding
	IllegalLeadingOrTrailingChar
	IllegalTabOrNewline
	AtInAuthority
	IllegalSlashes
	IllegalLocalFileAndHostCombo
	BadWindowsDriveLetter
	IllegalIPv4Address
	IllegalIPv6Address
	CouldNotDecodeHost
)

// Validation failures
const (
	FailIllegalCodePoint ErrorCode = iota + 500
	FailIllegalScheme
	FailRelativeUrlWithNoBase
	FailMissingHost
	FailIllegalHost
	FailIllegalPort
)

func (e ErrorCode) String() string {
	return fmt.Sprintf("%d: %s", e, messages[e])
}

func (e ErrorCode) Int32() int32 {
	return int32(e)
}

var messages = map[ErrorCode]string{
	// Validation errors
	IllegalCodePoint:             "illegal code point",
	InvalidPercentEncoding:       "invalid percent encoding",
	IllegalLeadingOrTrailingChar: "illegal leading or trailing character",
	IllegalTabOrNewline:          "illegal tab or newline",
	AtInAuthority:                "'@' in authority",
	IllegalSlashes:               "illegal combination of slashes",
	IllegalLocalFileAndHostCombo: "illegal combination of host and local file reference",
	BadWindowsDriveLetter:        "badly formatted windows drive letter",
	IllegalIPv4Address:           "illegal IPv4 address",
	IllegalIPv6Address:           "illegal IPv6 address",
	CouldNotDecodeHost:           "could not decode host",

	// Validation failures
	FailIllegalCodePoint:      "illegal code point",
	FailIllegalScheme:         "illegal scheme",
	FailRelativeUrlWithNoBase: "relative url with missing or invalid base url",
	FailMissingHost:           "missing host",
	FailIllegalHost:           "illegal host",
	FailIllegalPort:           "illegal port",
}
