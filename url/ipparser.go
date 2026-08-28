/*
 * Copyright 2019 National Library of Norway.
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
	"strconv"
	"strings"
)

// The ends in a number checker takes an ASCII string input and runs these
// steps. It returns a boolean.
// https://url.spec.whatwg.org/#ends-in-a-number-checker
func endsInANumber[T ByteSequence](input T) bool {
	// Assert: input is not the empty string. Callers in the host parser meet
	// this precondition; accepting empty input here keeps this helper total for
	// focused package tests.

	// Let parts be the result of strictly splitting input on U+002E (.). If the
	// last item is empty, remove it. This implementation locates the resulting
	// last item without allocating the full list.
	end := len(input)
	if end > 0 && input[end-1] == '.' {
		end--
	}
	start := end
	for start > 0 && input[start-1] != '.' {
		start--
	}
	last := input[start:end]

	// If last is not empty and contains only ASCII digits, then return true.
	if len(last) > 0 && onlyASCIIDigits(last) {
		return true
	}

	// If parsing last as an IPv4 number does not return failure, then return
	// true. This is equivalent to checking for "0X" or "0x" followed by zero
	// or more ASCII hex digits.
	// Implementation note: use that equivalent shape check directly so machine
	// integer overflow cannot change the result for long mathematical integers.
	return len(last) >= 2 && last[0] == '0' && (last[1] == 'x' || last[1] == 'X') &&
		onlyASCIIHexDigits(last[2:])
}

// parseIPv4 implements the IPv4 parser.
// https://url.spec.whatwg.org/#concept-ipv4-parser
func (c *parseContext) parseIPv4(u *Url, input string) (string, error) {
	// Let parts be the result of strictly splitting input on U+002E (.).
	parts := strings.Split(input, ".")

	// If the last item in parts is the empty string, then:
	if parts[len(parts)-1] == "" {
		// IPv4-empty-part validation error.
		if err := c.reportValidation(CodeIPv4EmptyPart, ErrIPv4EmptyPart.Detail(), nil); err != nil {
			return input, err
		}
		// If parts's size is greater than 1, then remove the last item from parts.
		if len(parts) > 1 {
			parts = parts[:len(parts)-1]
		}
	}

	// If parts's size is less than 4, IPv4-too-few-parts validation error.
	if len(parts) < 4 {
		if err := c.reportValidation(CodeIPv4TooFewParts, ErrIPv4TooFewParts.Detail(), nil); err != nil {
			return input, err
		}
	}

	// If parts's size is greater than 4, IPv4-too-many-parts validation error,
	// return failure.
	if len(parts) > 4 {
		if err := c.reportValidation(CodeIPv4TooManyParts, ErrIPv4TooManyParts.Detail(), nil); err != nil {
			return input, err
		}
		return input, c.returnFailure(ErrIPv4TooManyParts.Detail())
	}

	// Let numbers be an empty list.
	var numbers []int64

	// For each part of parts:
	for _, part := range parts {
		// Let result be the result of parsing part.
		n, validationError, err := parseIPv4Number(part)

		// If result is failure, IPv4-non-numeric-part validation error, return
		// failure.
		if err != nil {
			// Implementation note: an overflowing mathematical integer is still
			// numeric. Preserve the standard non-decimal and out-of-range events
			// before the subsequent failure, even though int64 cannot hold it.
			if errors.Is(err, ErrIPv4OutOfRangePart) {
				if validationError {
					if err := c.reportValidation(CodeIPv4NonDecimalPart, ErrIPv4NonDecimalPart.Detail(), nil); err != nil {
						return input, err
					}
				}
				if err := c.reportValidation(CodeIPv4OutOfRangePart, ErrIPv4OutOfRangePart.Detail(), nil); err != nil {
					return input, err
				}
				return input, c.returnFailure("an IPv4 number exceeds the supported integer range")
			}
			if err := c.reportValidation(CodeIPv4NonNumericPart, ErrIPv4NonNumericPart.Detail(), nil); err != nil {
				return input, err
			}
			return input, c.returnFailure(ErrIPv4NonNumericPart.Detail())
		}

		// If result[1] is true, IPv4-non-decimal-part validation error.
		if validationError {
			if err := c.reportValidation(CodeIPv4NonDecimalPart, ErrIPv4NonDecimalPart.Detail(), nil); err != nil {
				return input, err
			}
		}

		// Append result[0] to numbers.
		numbers = append(numbers, n)
	}

	// If any item in numbers is greater than 255, IPv4-out-of-range-part
	// validation error.
	last := len(numbers) - 1
	for _, n := range numbers {
		if n > 255 {
			if err := c.reportValidation(CodeIPv4OutOfRangePart, ErrIPv4OutOfRangePart.Detail(), nil); err != nil {
				return input, err
			}
			break
		}
	}

	// If any but the last item in numbers is greater than 255, then return
	// failure.
	for _, n := range numbers[:last] {
		if n > 255 {
			return input, c.returnFailure("an IPv4 part other than the last exceeds 255")
		}
	}

	// If the last item in numbers is greater than or equal to
	// 256^(5 − numbers's size), then return failure.
	if numbers[last] >= 1<<uint(8*(5-len(numbers))) {
		return input, c.returnFailure("the final IPv4 part exceeds the available address range")
	}

	// Let ipv4 be the last item in numbers.
	ipv4 := uint64(numbers[last])

	// Remove the last item from numbers.
	numbers = numbers[:last]

	// Let counter be 0. For each n of numbers, increment ipv4 by n ×
	// 256^(3 − counter), then increment counter by 1.
	for counter, n := range numbers {
		ipv4 += uint64(n) << uint(8*(3-counter))
	}

	// Return ipv4.
	u.isIPv4 = true
	return IPv4Addr(ipv4).String(), nil
}

// The IPv4 number parser takes an ASCII string input and runs these steps. It
// returns failure or a tuple of a number and a boolean.
// https://url.spec.whatwg.org/#ipv4-number-parser
func parseIPv4Number(input string) (number int64, validationError bool, err error) {
	// If input is the empty string, then return failure.
	if len(input) == 0 {
		return 0, false, ErrIPv4EmptyPart
	}

	// Let validationError be false. Let R be 10.
	R := 10

	// If input contains at least two code points and the first two code points
	// are either "0X" or "0x", then set validationError to true, remove those
	// code points, and set R to 16.
	if len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X') {
		validationError = true
		input = input[2:]
		R = 16
	} else if len(input) >= 2 && input[0] == '0' {
		// Otherwise, if input contains at least two code points and the first is
		// U+0030 (0), set validationError to true, remove it, and set R to 8.
		validationError = true
		input = input[1:]
		R = 8
	}

	// If input is the empty string, then return (0, true).
	if len(input) == 0 {
		return 0, true, nil
	}

	// If input contains a code point that is not a radix-R digit, then return
	// failure.
	switch R {
	case 8, 10:
		if !onlyASCIIDigits(input) {
			return 0, validationError, ErrIPv4NonNumericPart
		}
	case 16:
		if !onlyASCIIHexDigits(input) {
			return 0, validationError, ErrIPv4NonNumericPart
		}
	}

	// Let output be the mathematical integer represented by input in radix-R
	// notation, using ASCII hex digits for values 0 through 15.
	number, err = strconv.ParseInt(input, R, 64)
	if errors.Is(err, strconv.ErrRange) {
		// Implementation note: signal integer overflow distinctly so parseIPv4
		// can preserve the standard event sequence.
		err = ErrIPv4OutOfRangePart
	}

	// Return (output, validationError).
	return
}

// parseIPv6 implements the IPv6 parser.
// https://url.spec.whatwg.org/#concept-ipv6-parser
func parseIPv6(input []byte, reportValidation func(ValidationErrorCode, string, error) error) (*ipv6Addr, error) {
	// Implementation note: reportValidation is optional so focused callers can
	// parse without collecting non-fatal leading-zero validation events.
	// Let address be a new IPv6 address whose pieces are all 0.
	address := &ipv6Addr{}
	// Let pieceIndex be 0.
	pieceIdx := 0
	// Let compress be null.
	compress := -1
	// Let pointer be a pointer for input.
	i := 0

	// If c is U+003A (:), then:
	if len(input) > 0 && input[i] == ':' {
		// If remaining does not start with U+003A (:),
		if i+1 >= len(input) || input[i+1] != ':' {
			// IPv6-invalid-compression validation error, return failure.
			return nil, ErrIPv6InvalidCompression
		}
		// Increase pointer by 2.
		i += 2
		// Increase pieceIndex by 1 and then set compress to pieceIndex.
		pieceIdx++
		compress = pieceIdx
	}

	// While c is not the EOF code point:
	for i < len(input) {
		c := input[i]

		// If pieceIndex is 8, IPv6-too-many-pieces validation error, return
		// failure.
		if pieceIdx == 8 {
			return nil, ErrIPv6TooManyPieces
		}

		// If c is U+003A (:), then:
		if c == ':' {
			// If compress is non-null, IPv6-multiple-compression validation error,
			// return failure.
			if compress >= 0 {
				return nil, ErrIPv6MultipleCompression
			}
			// Increase pointer and pieceIndex by 1, set compress to pieceIndex,
			// and then continue.
			i++
			pieceIdx++
			compress = pieceIdx
			continue
		}

		// Let value and length be 0.
		value := uint16(0)
		length := 0
		pieceStart := i

		// While length is less than 4 and c is an ASCII hex digit, set value to
		// value × 0x10 + c interpreted as hexadecimal, and increase pointer and
		// length by 1.
		for length < 4 && isHex(c) {
			value = value*0x10 + uint16(unhex(c))
			i++
			length++
			if i < len(input) {
				c = input[i]
			} else {
				c = 0
			}
		}

		eof := i >= len(input)

		// If c is U+002E (.), then:
		if c == '.' {
			// If length is 0, IPv4-in-IPv6-invalid-code-point validation error,
			// return failure.
			if length == 0 {
				return nil, ErrIPv4InIPv6InvalidCodePoint
			}
			// Decrease pointer by length.
			i -= length
			// If pieceIndex is greater than 6, IPv4-in-IPv6-too-many-pieces
			// validation error, return failure.
			if pieceIdx > 6 {
				return nil, ErrIPv4InIPv6TooManyPieces
			}

			// Let numbersSeen be 0. While c is not the EOF code point:
			numbersSeen := 0
			for i < len(input) {
				c = input[i]
				// Let ipv4Piece be null.
				ipv4Piece := -1

				// If numbersSeen is greater than 0, then:
				if numbersSeen > 0 {
					// If c is U+002E (.) and numbersSeen is less than 4, increase
					// pointer by 1. Otherwise, IPv4-in-IPv6-invalid-code-point
					// validation error, return failure.
					if c == '.' && numbersSeen < 4 {
						i++
						if i >= len(input) {
							return nil, ErrIPv4InIPv6InvalidCodePoint
						}
						c = input[i]
					} else {
						return nil, ErrIPv4InIPv6InvalidCodePoint
					}
				}

				// If c is not an ASCII digit, IPv4-in-IPv6-invalid-code-point
				// validation error, return failure.
				if !isASCIIDigit(c) {
					return nil, ErrIPv4InIPv6InvalidCodePoint
				}

				// While c is an ASCII digit:
				for isASCIIDigit(c) {
					// Let number be c interpreted as a decimal number.
					number := int(unhex(c))
					// If ipv4Piece is null, set it to number. Otherwise, if it is
					// 0, IPv4-in-IPv6-invalid-code-point validation error, return
					// failure. Otherwise, update it with the next decimal digit.
					if ipv4Piece < 0 {
						ipv4Piece = number
					} else if ipv4Piece == 0 {
						return nil, ErrIPv4InIPv6InvalidCodePoint
					} else {
						ipv4Piece = ipv4Piece*10 + number
					}
					// If ipv4Piece is greater than 255,
					// IPv4-in-IPv6-out-of-range-part validation error, return failure.
					if ipv4Piece > 255 {
						return nil, ErrIPv4InIPv6OutOfRangePart
					}
					// Increase pointer by 1.
					i++
					if i < len(input) {
						c = input[i]
					} else {
						c = 0
					}
				}

				// Set address[pieceIndex] to address[pieceIndex] × 0x100 +
				// ipv4Piece.
				address[pieceIdx] = address[pieceIdx]*0x100 + uint16(ipv4Piece)
				// Increase numbersSeen by 1.
				numbersSeen++
				// If numbersSeen is 2 or 4, increase pieceIndex by 1.
				if numbersSeen == 2 || numbersSeen == 4 {
					pieceIdx++
				}
			}

			// If numbersSeen is not 4, IPv4-in-IPv6-too-few-parts validation
			// error, return failure.
			if numbersSeen != 4 {
				return nil, ErrIPv4InIPv6TooFewParts
			}
			break
		} else if c == ':' {
			// Otherwise, if c is U+003A (:), increase pointer by 1. If c is the
			// EOF code point, IPv6-invalid-code-point validation error, return
			// failure.
			i++
			if i >= len(input) {
				return nil, ErrIPv6InvalidCodePoint
			}
		} else if !eof {
			// Otherwise, if c is not the EOF code point,
			// IPv6-invalid-code-point validation error, return failure.
			return nil, ErrIPv6InvalidCodePoint
		}

		// If length is greater than 1 and the first code point of the piece is
		// U+0030 (0), IPv6-piece-leading-zero validation error.
		if length > 1 && input[pieceStart] == '0' && reportValidation != nil {
			if err := reportValidation(CodeIPv6PieceLeadingZero, ErrIPv6PieceLeadingZero.Detail(), nil); err != nil {
				return nil, err
			}
		}

		// Set address[pieceIndex] to value and increase pieceIndex by 1.
		address[pieceIdx] = value
		pieceIdx++
	}

	// If compress is non-null, then:
	if compress >= 0 {
		// Let swaps be pieceIndex − compress. Set pieceIndex to 7. While
		// pieceIndex is not 0 and swaps is greater than 0, swap the indicated
		// pieces and decrease both values by 1.
		swaps := pieceIdx - compress
		pieceIdx = 7
		for pieceIdx != 0 && swaps > 0 {
			t := address[pieceIdx]
			address[pieceIdx] = address[compress+swaps-1]
			address[compress+swaps-1] = t
			pieceIdx--
			swaps--
		}
	} else if pieceIdx != 8 {
		// Otherwise, if compress is null and pieceIndex is not 8,
		// IPv6-too-few-pieces validation error, return failure.
		return nil, ErrIPv6TooFewPieces
	}

	// Return address.
	return address, nil
}
