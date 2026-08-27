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
	"bytes"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// parseHost parses the host part of the input string.
//
// The host parser takes a scalar value string input with an optional boolean isOpaque (default false), and then runs these steps. They return failure or a host.
// parseHost is retained as a package-internal convenience for focused host
// parser tests. Basic URL parsing uses parseContext.parseHost directly so all
// nested validation events share the same per-parse ledger.
func (p *parser) parseHost(u *Url, input []byte, isNotSpecial bool) (string, error) {
	ctx := newParseContext(p)
	host, err := ctx.parseHost(u, input, isNotSpecial)
	ctx.attachValidationErrors(u)
	if err != nil {
		return host, ctx.normalizeFailure(err)
	}
	return host, nil
}

func (c *parseContext) parseHost(u *Url, input []byte, isNotSpecial bool) (string, error) {
	// If input starts with U+005B ([), then:
	if len(input) > 0 && input[0] == '[' {
		// If input does not end with U+005D (]),
		if input[len(input)-1] != ']' {
			// IPv6-unclosed validation error, return failure.
			if err := c.reportValidation(CodeIPv6Unclosed, ErrIPv6Unclosed.Detail(), nil); err != nil {
				return "", err
			}
			return "", c.returnFailure(ErrIPv6Unclosed.Detail())
		}
		// Return the result of IPv6 parsing input with its leading U+005B ([)
		// and trailing U+005D (]) removed.
		addr, err := parseIPv6WithValidation(input[1:len(input)-1], c.reportValidation)
		if err != nil {
			var parseErr *ParseError
			if errors.As(err, &parseErr) {
				return "", parseErr
			}
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				return "", c.returnFailure(err.Error())
			}
			if err := c.reportValidation(validationErr.Code(), validationErr.Detail(), validationErr.Unwrap()); err != nil {
				return "", err
			}
			return "", c.returnFailure(validationErr.Detail())
		}
		u.isIPv6 = true
		return addr.StringWithBrackets(), nil
	}

	// If isOpaque is true, then return the result of opaque-host parsing input.
	if isNotSpecial {
		return c.parseOpaqueHost(u, input)
	}

	// Assert: input is not the empty string.
	if len(input) == 0 {
		return "", nil
	}

	// Let domain be the result of running UTF-8 decode without BOM on the
	// percent-decoding of input.
	//
	// Note: Alternatively UTF-8 decode without BOM or fail can be used,
	// coupled with an early return for failure, as domain to ASCII fails on U+FFFD (�).
	buf := append([]byte(nil), input...)

	isPercentEncoded := false

	if bytes.IndexByte(buf, '%') != -1 {
		buf, isPercentEncoded = percentDecodeInPlace(buf)
		if isPercentEncoded {
			if err := c.reportValidation(CodeDomainPercentEncoded, ErrDomainPercentEncoded.Detail(), nil); err != nil {
				return "", err
			}
		}
	}

	// Remove UTF-8 BOM if present (URL Standard: “UTF-8 decode without BOM”).
	if len(buf) >= 3 &&
		buf[0] == 0xEF &&
		buf[1] == 0xBB &&
		buf[2] == 0xBF {
		buf = buf[3:]
	}

	// Fast path: ASCII-only and no punycode labels.
	// (asciiNoPunycodeLabels assumes ASCII; it returns false if any non-ASCII.)
	if asciiNoPunycodeLabels(buf) {
		for i, c := range buf {
			if 'A' <= c && c <= 'Z' {
				buf[i] |= 0x20
			}
		}
		needsStrictValidation := !isStrictASCIIValid(buf)
		endsInNumber := endsInANumber(buf)
		asciiDomain := string(buf)
		if needsStrictValidation {
			// The local check is conservative. Use the strict IDNA profile only
			// when it detects a possible validation error so that x/net remains
			// the source of the detailed error.
			if _, err := strictDomainToASCII(asciiDomain); err != nil {
				if err := c.reportValidation(CodeDomainToASCII, ErrDomainToASCII.Detail(), err); err != nil {
					return "", err
				}
			}
		}
		if asciiDomain == "" {
			return "", c.returnFailure("Unicode ToASCII returned the empty string")
		}
		for _, b := range buf {
			if forbiddenDomainASCII[b] {
				return "", c.returnFailure("the host contains a forbidden domain code point")
			}
		}
		if endsInNumber {
			return c.parseIPv4(u, asciiDomain)
		}
		return asciiDomain, nil
	}

	// The IDNA implementation rejects invalid UTF-8 as U+FFFD, so malformed
	// input reaches the domain-to-ASCII failure required by the URL Standard
	// without a separate utf8.Valid check.
	// Let asciiDomain be the result of running domain to ASCII with domain and false.
	domain := string(buf)

	asciiDomain, warning, failure := processDomainToASCII(
		domain,
		false,
		true,
	)
	if warning != nil {
		var validationErr *ValidationError
		if errors.As(warning, &validationErr) {
			if err := c.reportValidation(validationErr.Code(), validationErr.Detail(), validationErr.Unwrap()); err != nil {
				return "", err
			}
		}
	}
	// If asciiDomain is failure,
	if failure != nil {
		if c.LaxHostParsing {
			return string(input), nil
		}
		// then return failure.
		return "", c.returnFailure(failure.Error())
	}
	// If asciiDomain ends in a number,
	if endsInANumber(asciiDomain) {
		// If domain is not an ASCII string, IPv4-non-ASCII-input validation error.
		if !isASCII(domain) {
			if err := c.reportValidation(CodeIPv4NonASCIIInput, ErrIPv4NonASCIIInput.Detail(), nil); err != nil {
				return "", err
			}
		}
		// Return the result of IPv4 parsing asciiDomain.
		return c.parseIPv4(u, asciiDomain)
	}

	// Return asciiDomain.
	return asciiDomain, nil
}

// The ends in a number checker takes an ASCII byte sequence input and then
// runs these steps. They return a boolean.
func endsInANumber[T ByteSequence](input T) bool {
	// Remove at most one trailing U+002E (.) logically.
	end := len(input)
	if end > 0 && input[end-1] == '.' {
		end--
	}
	// Let last be the last item from strictly splitting input on U+002E (.).
	start := end
	for start > 0 && input[start-1] != '.' {
		start--
	}
	last := input[start:end]
	// If last is non-empty and contains only ASCII digits,
	if len(last) > 0 && onlyASCIIDigits(last) {
		// then return true.
		return true
	}
	// Parsing last as an IPv4 number can otherwise succeed only when it is
	// "0X" or "0x", followed by zero or more ASCII hex digits. Values that
	// overflow the IPv4 number parser still make the domain end in a number.
	return len(last) >= 2 && last[0] == '0' && (last[1] == 'x' || last[1] == 'X') &&
		onlyASCIIHexDigits(last[2:])
}

func (c *parseContext) parseIPv4(u *Url, input string) (string, error) {
	// Let parts be the result of strictly splitting input on U+002E (.).
	parts := strings.Split(input, ".")
	// If the last item in parts is the empty string, then:
	if parts[len(parts)-1] == "" {
		// IPv4-empty-part validation error.
		if err := c.reportValidation(CodeIPv4EmptyPart, ErrIPv4EmptyPart.Detail(), nil); err != nil {
			return input, err
		}
		// If parts’s size is greater than 1,
		if len(parts) > 1 {
			// then remove the last item from parts.
			parts = parts[:len(parts)-1]
		}
	}
	// If parts's size is less than 4, IPv4-too-few-parts validation error.
	if len(parts) < 4 {
		if err := c.reportValidation(CodeIPv4TooFewParts, ErrIPv4TooFewParts.Detail(), nil); err != nil {
			return input, err
		}
	}
	// If parts’s size is greater than 4,
	if len(parts) > 4 {
		// IPv4-too-many-parts validation error, return failure.
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
		// If result is failure,
		if err != nil {
			if errors.Is(err, ErrIPv4OutOfRangePart) {
				if validationError {
					if err := c.reportValidation(CodeIPv4NonDecimalPart, ErrIPv4NonDecimalPart.Detail(), nil); err != nil {
						return input, err
					}
				}
				// The mathematical value cannot fit in the parser's machine
				// representation. It is necessarily greater than 255, so record
				// the standard range event before taking the later bare failure.
				if err := c.reportValidation(CodeIPv4OutOfRangePart, ErrIPv4OutOfRangePart.Detail(), nil); err != nil {
					return input, err
				}
				return input, c.returnFailure("an IPv4 number exceeds the supported integer range")
			}
			// IPv4-non-numeric-part validation error, return failure.
			if err := c.reportValidation(CodeIPv4NonNumericPart, ErrIPv4NonNumericPart.Detail(), nil); err != nil {
				return input, err
			}
			return input, c.returnFailure(ErrIPv4NonNumericPart.Detail())
		}
		// If result[1] is true,
		if validationError {
			// IPv4-non-decimal-part validation error.
			if err := c.reportValidation(CodeIPv4NonDecimalPart, ErrIPv4NonDecimalPart.Detail(), nil); err != nil {
				return input, err
			}
		}
		// Append result[0] to numbers.
		numbers = append(numbers, n)
	}
	last := len(numbers) - 1
	// If any item in numbers is greater than 255, IPv4-out-of-range-part
	// validation error. This is one standard event regardless of how many
	// individual parts exceed the limit.
	for _, n := range numbers {
		if n > 255 {
			if err := c.reportValidation(CodeIPv4OutOfRangePart, ErrIPv4OutOfRangePart.Detail(), nil); err != nil {
				return input, err
			}
			break
		}
	}
	for _, n := range numbers[:last] {
		// If any but the last item in numbers is greater than 255,
		if n > 255 {
			// then return failure.
			return input, c.returnFailure("an IPv4 part other than the last exceeds 255")
		}
	}
	// If the last item in numbers is greater than or equal to 256(5 − numbers’s size),
	// note: 1<<uint(8*(5-len(numbers))) is the same as 256(5 − numbers’s size)
	if numbers[last] >= 1<<uint(8*(5-len(numbers))) {
		// then return failure.
		return input, c.returnFailure("the final IPv4 part exceeds the available address range")
	}
	// Let ipv4 be the last item in numbers.
	ipv4 := uint64(numbers[last])
	// Remove the last item from numbers.
	numbers = numbers[:last]
	// Let counter be 0.
	// For each n of numbers:
	for counter, n := range numbers {
		// Increment ipv4 by n × 256(3 − counter).
		// note: 1<<uint(8*(3-counter)) is the same as 256(3 − counter)
		ipv4 += uint64(n) << uint(8*(3-counter))
		// Increment counter by 1.
	}
	// Return ipv4.
	u.isIPv4 = true
	return IPv4Addr(ipv4).String(), nil
}

func parseIPv6(input []byte) (*ipv6Addr, error) {
	return parseIPv6WithValidation(input, nil)
}

func parseIPv6WithValidation(input []byte, reportValidation func(ValidationErrorCode, string, error) error) (*ipv6Addr, error) {

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
		// Increase pieceIndex by 1
		pieceIdx++
		// and then set compress to pieceIndex.
		compress = pieceIdx
	}

	// While c is not the EOF code point:
	for i < len(input) {
		c := input[i]

		// If pieceIndex is 8,
		if pieceIdx == 8 {
			// IPv6-too-many-pieces validation error, return failure.
			return nil, ErrIPv6TooManyPieces
		}
		// If c is U+003A (:), then:
		if c == ':' {
			// If compress is non-null,
			if compress >= 0 {
				// IPv6-multiple-compression validation error, return failure.
				return nil, ErrIPv6MultipleCompression
			}
			// Increase pointer
			i++
			// and pieceIndex by 1,
			pieceIdx++
			// set compress to pieceIndex,
			compress = pieceIdx
			// and then continue.
			continue
		}
		// Let value
		value := uint16(0)
		// and length be 0.
		length := 0
		pieceStart := i

		// While length is less than 4 and c is an ASCII hex digit
		for length < 4 && isHex(c) {
			// set value to value × 0x10 + c interpreted as hexadecimal number,
			value = value*0x10 + uint16(unhex(c))
			// and increase pointer and length by 1.
			i++
			length++

			if i < len(input) {
				c = input[i]
			} else {
				c = 0 // non hex to break loop
			}
		}

		eof := i >= len(input)

		// If c is U+002E (.), then:
		if c == '.' {
			// If length is 0, IPv4-in-IPv6-invalid-code-point validation error,
			if length == 0 {
				// return failure.
				return nil, ErrIPv4InIPv6InvalidCodePoint
			}
			// Decrease pointer by length.
			i -= length
			// If pieceIndex is greater than 6, IPv4-in-IPv6-too-many-pieces validation error,
			if pieceIdx > 6 {
				// return failure.
				return nil, ErrIPv4InIPv6TooManyPieces
			}
			// Let numbersSeen be 0.
			numbersSeen := 0
			// While c is not the EOF code point:
			for i < len(input) {
				c = input[i]
				// Let ipv4Piece be null.
				ipv4Piece := -1
				// If numbersSeen is greater than 0, then:
				if numbersSeen > 0 {
					// If c is a U+002E (.) and numbersSeen is less than 4,
					if c == '.' && numbersSeen < 4 {
						// then increase pointer by 1.
						i++
						if i >= len(input) {
							return nil, ErrIPv4InIPv6InvalidCodePoint
						}
						c = input[i]
					} else {
						// Otherwise, IPv4-in-IPv6-invalid-code-point validation error, return failure.
						return nil, ErrIPv4InIPv6InvalidCodePoint
					}
				}
				// If c is not an ASCII digit,
				if !isASCIIDigit(c) {
					// IPv4-in-IPv6-invalid-code-point validation error, return failure.
					return nil, ErrIPv4InIPv6InvalidCodePoint
				}
				// While c is an ASCII digit:
				for isASCIIDigit(c) {
					// Let number be c interpreted as decimal number.
					number := int(unhex(c))

					// If ipv4Piece is null,
					if ipv4Piece < 0 {
						// then set ipv4Piece to number.
						ipv4Piece = number
					} else // Otherwise, if ipv4Piece is 0,
					if ipv4Piece == 0 {
						// IPv4-in-IPv6-invalid-code-point validation error, return failure.
						return nil, ErrIPv4InIPv6InvalidCodePoint
					} else { // Otherwise,
						// set ipv4Piece to ipv4Piece × 10 + number.
						ipv4Piece = ipv4Piece*10 + number
					}
					// If ipv4Piece is greater than 255,
					if ipv4Piece > 255 {
						// IPv4-in-IPv6-out-of-range-part validation error, return failure.
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
				// Set address[pieceIndex] to address[pieceIndex] × 0x100 + ipv4Piece.
				address[pieceIdx] = address[pieceIdx]*0x100 + uint16(ipv4Piece)
				// Increase numbersSeen by 1.
				numbersSeen++
				// If numbersSeen is 2 or 4,
				if numbersSeen == 2 || numbersSeen == 4 {
					// then increase pieceIndex by 1.
					pieceIdx++
				}
			}
			// If numbersSeen is not 4,
			if numbersSeen != 4 {
				// IPv4-in-IPv6-too-few-parts validation error, return failure.
				return nil, ErrIPv4InIPv6TooFewParts
			}
			break
		} else
		// Otherwise, if c is U+003A (:):
		if c == ':' {
			// Increase pointer by 1.
			i++
			// If c is the EOF code point,
			if i >= len(input) {
				// IPv6-invalid-code-point validation error, return failure.
				return nil, ErrIPv6InvalidCodePoint
			}
		} else
		// Otherwise, if c is not the EOF code point,
		if !eof {
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
		// Set address[pieceIndex] to value.
		address[pieceIdx] = uint16(value)
		// Increase pieceIndex by 1.
		pieceIdx++
	}
	// If compress is non-null, then:
	if compress >= 0 {
		// Let swaps be pieceIndex − compress.
		swaps := pieceIdx - compress
		// Set pieceIndex to 7.
		pieceIdx = 7
		// While pieceIndex is not 0 and swaps is greater than 0,
		for pieceIdx != 0 && swaps > 0 {
			// swap address[pieceIndex] with address[compress + swaps − 1],
			t := address[pieceIdx]
			address[pieceIdx] = address[compress+swaps-1]
			address[compress+swaps-1] = t
			// and then decrease both pieceIndex
			pieceIdx--
			// and swaps by 1.
			swaps--
		}
	} else
	// Otherwise, if compress is null and pieceIndex is not 8,
	if compress == -1 && pieceIdx != 8 {
		// IPv6-too-few-pieces validation error, return failure.
		return nil, ErrIPv6TooFewPieces
	}
	// Return address.
	return address, nil
}

func (ctx *parseContext) parseOpaqueHost(u *Url, input []byte) (string, error) {
	// Worst case: every byte becomes %HH (3x).
	out := make([]byte, 0, len(input)*3)

	for i := 0; i < len(input); {
		c := input[i]

		// ASCII fast path

		if c < utf8.RuneSelf {
			// If input contains a forbidden host code point,
			if forbiddenHostASCII[c] {
				// host-invalid-code-point validation error, return failure.
				if err := ctx.reportValidation(CodeHostInvalidCodePoint, validationDetail(ErrHostInvalidCodePoint, byteCodePointDescription(c)), nil); err != nil {
					return "", err
				}
				if ctx.LaxHostParsing {
					return string(input), nil
				}
				return "", ctx.returnFailure(ErrHostInvalidCodePoint.Detail())
			}

			// If input contains a code point that is not a URL code point and not U+0025 (%),
			if !urlCodePointASCII[c] && c != '%' {
				// invalid-URL-unit validation error.
				if err := ctx.reportValidation(CodeInvalidURLUnit, validationDetail(ErrInvalidURLUnit, byteCodePointDescription(c)), nil); err != nil {
					return "", err
				}
			}

			// If input contains a U+0025 (%) and the two code points following it are not ASCII hex digits,
			if c == '%' {
				if i+2 >= len(input) || !isHex(input[i+1]) || !isHex(input[i+2]) {
					// Mirror your existing remainingIsInvalidPercentEncoded behavior:
					// Provide whatever description you currently build (here: just "%").

					// invalid-URL-unit validation error.
					if err := ctx.reportValidation(CodeInvalidURLUnit, validationDetail(ErrInvalidURLUnit, codePointDescription('%')), nil); err != nil {
						return "", err
					}
					if ctx.PercentEncodeSinglePercentSign {
						out = append(out, percentPercentEncoded...)
						i++
						continue
					}
				}
			}

			// Return the result of running UTF-8 percent-encode on input using
			// the C0 control percent-encode set.
			if C0PercentEncodeSet.ByteShouldBeEncoded(c) {
				out = append(out, '%', upperHex[c>>4], upperHex[c&15])
			} else {
				out = append(out, c)
			}
			i++
			continue
		}

		// Non-ASCII: decode one rune
		r, size := utf8.DecodeRune(input[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte in opaque host.
			// Spec-wise this is an invalid-URL-unit; decide whether you want strict fail or
			// tolerate under AcceptInvalidCodepoints (if that option is meant to apply here).
			if err := ctx.reportValidation(CodeInvalidURLUnit, validationDetail(ErrInvalidURLUnit, invalidUTF8ByteDescription(input[i])), nil); err != nil {
				return "", err
			}
			// Encode the raw byte (not U+FFFD) if you accept invalid code points; otherwise encode U+FFFD.
			if ctx.AcceptInvalidCodepoints {
				c = input[i]
				if C0PercentEncodeSet.ByteShouldBeEncoded(c) {
					out = append(out, '%', upperHex[c>>4], upperHex[c&15])
				} else {
					out = append(out, c)
				}
				i++
			} else {
				out = appendPercentEncoded(out, utf8.RuneError, C0PercentEncodeSet)
				i++
			}
			continue
		}

		// If input contains a code point that is not a URL code point and not U+0025 (%),
		if !isURLCodePoint(r) && r != '%' {
			// host-invalid-code-point validation error, return failure.
			if err := ctx.reportValidation(CodeInvalidURLUnit, validationDetail(ErrInvalidURLUnit, codePointDescription(r)), nil); err != nil {
				return "", err
			}
		}

		// If '%' rune appears here (it won't; '%' is ASCII) you'd handle it above.
		out = appendPercentEncoded(out, r, C0PercentEncodeSet)
		i += size
	}

	return string(out), nil
}

// The IPv4 number parser takes an ASCII string input and then runs these
// steps. They return failure or a tuple of a number and a boolean.
func parseIPv4Number(input string) (number int64, validationError bool, err error) {
	// If input is the empty string, then return failure.
	if len(input) == 0 {
		return 0, false, ErrIPv4EmptyPart
	}
	// Let validationError be false.

	// Let R be 10.
	R := 10

	// If input contains at least two code points and the first two code points
	// are either "0X" or "0x", then:
	if len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X') {
		// Set validationError to true.
		validationError = true
		// Remove the first two code points from input.
		input = input[2:]
		// Set R to 16.
		R = 16
	} else
	// Otherwise, if input contains at least two code points and the first code
	// point is U+0030 (0), then:
	if len(input) >= 2 && input[0] == '0' {
		// Set validationError to true.
		validationError = true
		// Remove the first code point from input.
		input = input[1:]
		// Set R to 8.
		R = 8
	}
	// If input is the empty string,
	if len(input) == 0 {
		// then return (0, true).
		return 0, true, nil
	}
	// If input contains a code point that is not a radix-R digit, then return failure.
	switch R {
	case 8, 10:
		if !onlyASCIIDigits(input) {
			err = ErrIPv4NonNumericPart
			return
		}
	case 16:
		if !onlyASCIIHexDigits(input) {
			err = ErrIPv4NonNumericPart
			return
		}
	}
	// Let output be the mathematical integer value that is represented by
	// input in radix-R notation, using ASCII hex digits for digits with values 0 through 15.
	number, err = strconv.ParseInt(input, R, 64)
	if errors.Is(err, strconv.ErrRange) {
		err = ErrIPv4OutOfRangePart
	}
	// Return (output, validationError).
	return
}

type ipv6Addr [8]uint16

func (address *ipv6Addr) String() string {
	var buf [40]byte
	return string(address.serializeTo(buf[:0]))
}

func (address *ipv6Addr) StringWithBrackets() string {
	var buf [42]byte
	b := buf[:0]

	b = append(b, '[')
	b = address.serializeTo(b)
	b = append(b, ']')

	return string(b)
}

func (address *ipv6Addr) serializeTo(b []byte) []byte {
	// Find the longest run of zeros for compression
	compress, compressLength := -1, 0
	currentIdx, currentLength := -1, 0
	for pieceIdx := range 8 {
		if address[pieceIdx] == 0 {
			if currentIdx < 0 {
				currentIdx = pieceIdx
			}
			currentLength++
		} else {
			if currentLength > 1 && currentLength > compressLength {
				compress = currentIdx
				compressLength = currentLength
			}
			currentIdx = -1
			currentLength = 0
		}
	}
	if currentLength > 1 && currentLength > compressLength {
		compress = currentIdx
	}

	ignore0 := false
	for i := range 8 {
		if ignore0 && address[i] == 0 {
			continue
		}
		ignore0 = false

		if i == compress {
			b = append(b, ':', ':')
			ignore0 = true
			// If the compression extends to the end, we are done
			if i+compressLength == 8 {
				break
			}
			continue
		}

		b = strconv.AppendUint(b, uint64(address[i]), 16)

		if i < 7 && i+1 != compress {
			b = append(b, ':')
		}
	}

	return b
}

type IPv4Addr uint32

func (address IPv4Addr) String() string {
	var buf [15]byte
	b := buf[:0]

	b = strconv.AppendUint(b, uint64(address>>24), 10)
	b = append(b, '.')
	b = strconv.AppendUint(b, uint64((address>>16)&0xFF), 10)
	b = append(b, '.')
	b = strconv.AppendUint(b, uint64((address>>8)&0xFF), 10)
	b = append(b, '.')
	b = strconv.AppendUint(b, uint64(address&0xFF), 10)

	return string(b)
}

func domainToASCIIProfile(beStrict bool) *idna.Profile {
	return idna.New(
		idna.MapForLookup(),
		idna.VerifyDNSLength(beStrict),
		idna.StrictDomainName(beStrict),
		idna.CheckHyphens(beStrict),
		idna.CheckJoiners(true),
		idna.BidiRule(),
		idna.Transitional(false),
	)
}

var (
	idnaStrict    = domainToASCIIProfile(true)
	idnaNonStrict = domainToASCIIProfile(false)
)

// strictDomainToASCII preserves a trailing root label while working around
// x/net/idna treating that label as an empty label during DNS-length
// verification. UTS #46 excludes a trailing root-label dot from the DNS
// length and label checks.
func strictDomainToASCII(domain string) (string, error) {
	trimmed := domain
	hasRootLabel := false
	for _, separator := range []string{".", "\u3002", "\uff0e", "\uff61"} {
		if strings.HasSuffix(trimmed, separator) && len(trimmed) > len(separator) {
			trimmed = strings.TrimSuffix(trimmed, separator)
			hasRootLabel = true
			break
		}
	}

	result, err := idnaStrict.ToASCII(trimmed)
	if hasRootLabel {
		result += "."
	}
	return result, err
}

// processDomainToASCII converts a string to ASCII using IDNA. A strict IDNA
// error can be returned separately as a non-fatal validation warning when
// relaxed processing succeeds.
func processDomainToASCII(domain string, beStrict, validate bool) (result string, warning, failure error) {
	if domain == "" {
		return "", nil, nil
	}

	if beStrict {
		result, err := strictDomainToASCII(domain)
		if err != nil {
			return result, nil, validationErrorWithCause(ErrDomainToASCII, err)
		}
		return result, nil, nil
	}

	ascii := isASCII(domain)
	haveResult := false
	if validate {
		strictResult, err := strictDomainToASCII(domain)
		if err == nil {
			if !ascii {
				result = strictResult
				haveResult = true
			}
		} else {
			warning = validationErrorWithCause(ErrDomainToASCII, err)
		}
	}

	// For URL parsing, ASCII input is returned lowercased regardless of
	// Unicode ToASCII's outcome for web compatibility.
	if ascii {
		result = strings.ToLower(domain)
	} else if !haveResult {
		var err error
		result, err = idnaNonStrict.ToASCII(domain)
		if err != nil {
			// Strict processing above is the single domain-to-ASCII validation
			// occurrence. Failure of the relaxed conversion is the subsequent
			// bare failure step.
			return result, warning, err
		}
	}

	if result == "" {
		return "", warning, errors.New("Unicode ToASCII returned the empty string")
	}

	for _, r := range result {
		if r < utf8.RuneSelf && forbiddenDomainASCII[r] {
			return result, warning, errors.New("the host contains a forbidden domain code point")
		}
	}

	return result, warning, nil
}

// domainToASCII is the output-only domain conversion wrapper used by direct
// callers that do not need strict validation warnings.
// https://url.spec.whatwg.org/#concept-domain-to-ascii
func domainToASCII(domain string, beStrict bool) (string, error) {
	result, _, failure := processDomainToASCII(domain, beStrict, false)
	return result, failure
}

func asciiNoPunycodeLabels(s []byte) bool {
	p := 0
	for i := range s {
		b := s[i]
		if b >= utf8.RuneSelf { // non-ASCII
			return false
		}
		switch {
		case b == '.':
			p = 0
		case p == 0 && (b == 'x' || b == 'X'):
			p = 1
		case p == 1 && (b == 'n' || b == 'N'):
			p = 2
		case p == 2 && b == '-':
			p = 3
		case p == 3 && b == '-':
			return false
		default:
			p = -1
		}
	}
	return true
}

// isStrictASCIIValid reports whether a lowercased ASCII domain with no
// Punycode labels satisfies the strict UTS #46 checks. Under these
// preconditions, Bidi, Joiner, normalization, and Punycode validation cannot
// fail, leaving only STD3, hyphen, empty-label, and DNS-length checks.
func isStrictASCIIValid(domain []byte) bool {
	if len(domain) == 0 {
		return false
	}
	if domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	labelStart := 0
	for i := 0; i <= len(domain); i++ {
		if i < len(domain) && domain[i] != '.' {
			b := domain[i]
			if ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || b == '-' {
				continue
			}
			return false
		}

		labelLen := i - labelStart
		if labelLen == 0 || labelLen > 63 ||
			domain[labelStart] == '-' || domain[i-1] == '-' ||
			(labelLen >= 4 && domain[labelStart+2] == '-' && domain[labelStart+3] == '-') {
			return false
		}
		labelStart = i + 1
	}

	return true
}

// containsOnlyASCIIOrMiscAndNoPunycode returns true if the string contains only ASCII characters or characters from Section 4.1.1 in UTS #46
// and does not contain any labels starting with acePrefix (xn--)
func containsOnlyASCIIOrMiscAndNoPunycode(s string) bool {
	p := 0
	for i := 0; i < len(s); i++ {
		b := s[i]

		if b >= utf8.RuneSelf {
			// Must be one of the allowed UTF-8 sequences
			r, size := utf8.DecodeRuneInString(s[i:])
			if r != '\u2260' && r != '\u226e' && r != '\u226f' {
				return false
			}
			i += size - 1
			p = -1
			continue
		}

		switch {
		case b == '.':
			p = 0
		case p == 0 && (b == 'x' || b == 'X'):
			p = 1
		case p == 1 && (b == 'n' || b == 'N'):
			p = 2
		case p == 2 && b == '-':
			p = 3
		case p == 3 && b == '-':
			return false
		default:
			p = -1
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
