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

package internal

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
func (p *Parser) parseHost(u *Url, input []byte, isNotSpecial bool) (string, error) {
	if p.PreParseHostFunc != nil {
		s := p.PreParseHostFunc(u, string(input))
		input = []byte(s)
	}

	// If input starts with U+005B ([), then:
	if len(input) > 0 && input[0] == '[' {
		// If input does not end with U+005D (]),
		if input[len(input)-1] != ']' {
			// IPv6-unclosed validation error, return failure.
			return "", p.handleErrorAndFail(u, ErrIPv6Unclosed)
		}
		// Return the result of IPv6 parsing input with its leading U+005B ([)
		// and trailing U+005D (]) removed.
		addr, err := parseIPv6(input[1 : len(input)-1])
		if err != nil {
			if err := p.handleError(u, err); err != nil {
				return "", err
			}
		}
		u.isIPv6 = true
		return addr.StringWithBrackets(), nil
	}

	// If isOpaque is true, then return the result of opaque-host parsing input.
	if isNotSpecial {
		return p.parseOpaqueHost(u, input)
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

	// Let asciiDomain be the result of running domain to ASCII with domain and false.
	asciiDomain, err := p.hostToASCII(input)
	// If asciiDomain is failure,
	if err != nil {
		if p.LaxHostParsing {
			return string(input), nil
		}
		// then return failure.
		if err := p.handleError(u, err); err != nil {
			return "", err
		}
	}
	// If asciiDomain ends in a number,
	if endsInANumber(asciiDomain) {
		// then return the result of IPv4 parsing asciiDomain.
		return p.parseIPv4(u, asciiDomain)
	}

	if p.PostParseHostFunc != nil {
		return p.PostParseHostFunc(u, asciiDomain), nil
	}

	// Return asciiDomain.
	return asciiDomain, nil
}

func (p *Parser) hostToASCII(input []byte) (string, error) {
	buf := append([]byte(nil), input...)

	if bytes.IndexByte(buf, '%') != -1 {
		buf = PercentDecodeInPlace(buf)
	}

	// Fast path: ASCII-only and no punycode labels.
	// (asciiNoPctNoPunycode assumes ASCII; it returns false if any non-ASCII.)
	if asciiNoPunycodeLabels(buf) {
		for i, c := range buf {
			if forbiddenDomainASCII[c] {
				return "", ErrDomainInvalidCodePoint
			}
			if 'A' <= c && c <= 'Z' {
				buf[i] |= 0x20
			}
		}
		return string(buf), nil
	}

	// Remove UTF-8 BOM if present (URL Standard: “UTF-8 decode without BOM”).
	if len(buf) >= 3 &&
		buf[0] == 0xEF &&
		buf[1] == 0xBB &&
		buf[2] == 0xBF {
		buf = buf[3:]
	}

	if !utf8.Valid(buf) {
		return "", ErrDomainToASCII
	}
	return domainToASCII(string(buf), false)
}

// The ends in a number checker takes an ASCII string input and then runs these
// steps. They return a boolean.
func endsInANumber(input string) bool {
	// Let parts be the result of strictly splitting input on U+002E (.).
	parts := strings.Split(input, ".")
	// If the last item in parts is the empty string, then:
	if parts[len(parts)-1] == "" {
		// If parts’s size is 1,
		if len(parts) == 1 {
			// then return false.
			return false
		}
		// Remove the last item from parts.
		parts = parts[0 : len(parts)-1]
	}
	// Let last be the last item in parts.
	last := parts[len(parts)-1]
	// If last is non-empty and contains only ASCII digits,
	if last != "" && onlyASCIIDigits(last) {
		// then return true.
		return true
	}
	// If parsing last as an IPv4 number does not return failure,
	_, _, err := parseIPv4Number(last)
	if err == nil {
		// then return true.
		return true
		// Note: This is equivalent to checking that last is "0X" or "0x",
		// followed by zero or more ASCII hex digits.
	}
	return errors.Is(err, ErrIPv4OutOfRangePart)
}

func (p *Parser) parseIPv4(u *Url, input string) (string, error) {
	// Let parts be the result of strictly splitting input on U+002E (.).
	parts := strings.Split(input, ".")
	// If the last item in parts is the empty string, then:
	if parts[len(parts)-1] == "" {
		// IPv4-empty-part validation error.
		if err := p.handleError(u, ErrIPv4EmptyPart); err != nil {
			return input, err
		}
		// If parts’s size is greater than 1,
		if len(parts) > 1 {
			// then remove the last item from parts.
			parts = parts[:len(parts)-1]
		}
	}
	// If parts’s size is greater than 4,
	if len(parts) > 4 {
		// IPv4-too-many-parts validation error, return failure.
		if err := p.handleError(u, ErrIPv4TooManyParts); err != nil {
			return input, err
		}
	}
	// Let numbers be an empty list.
	var numbers []int64
	// For each part of parts:
	for _, part := range parts {
		// Let result be the result of parsing part.
		n, validationError, err := parseIPv4Number(part)
		// If result is failure,
		if err != nil {
			// IPv4-non-numeric-part validation error, return failure.
			if err := p.handleError(u, ErrIPv4NonNumericPart); err != nil {
				return input, err
			}
		}
		// If result[1] is true,
		if validationError {
			// IPv4-non-decimal-part validation error.
			if err := p.handleError(u, ErrIPv4NonDecimalPart); err != nil {
				return input, err
			}
		}
		// Append result[0] to numbers.
		numbers = append(numbers, n)
	}
	last := len(numbers) - 1
	for _, n := range numbers {
		// If any item in numbers is greater than 255,
		if n > 255 {
			// IPv4-out-of-range-part validation error.
			err := p.handleError(u, ErrIPv4OutOfRangePart)
			if err != nil && p.FailOnValidationError {
				return input, err
			}
		}
	}
	for _, n := range numbers[:last] {
		// If any but the last item in numbers is greater than 255,
		if n > 255 {
			// then return failure.
			if err := p.handleErrorAndFail(u, ErrIPv4OutOfRangePart); err != nil {
				return input, err
			}
		}
	}
	// If the last item in numbers is greater than or equal to 256(5 − numbers’s size),
	// note: 1<<uint(8*(5-len(numbers))) is the same as 256(5 − numbers’s size)
	if numbers[last] >= 1<<uint(8*(5-len(numbers))) {
		// then return failure.
		if err := p.handleErrorAndFail(u, ErrIPv4OutOfRangePart); err != nil {
			return input, err
		}
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
	if len(input) < 2 {
		return nil, ErrIPv6TooFewPieces
	}

	// Let address be a new IPv6 address whose pieces are all 0.
	address := &ipv6Addr{}
	// Let pieceIndex be 0.
	pieceIdx := 0
	// Let compress be null.
	compress := -1
	// Let pointer be a pointer for input.
	i := 0

	c := input[i]

	// If c is U+003A (:), then:
	if c == ':' {
		// If remaining does not start with U+003A (:),
		if input[i+1] != ':' {
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
		c = input[i]

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

func (p *Parser) parseOpaqueHost(u *Url, input []byte) (string, error) {
	// Worst case: every byte becomes %HH (3x).
	out := make([]byte, 0, len(input)*3)

	for i := 0; i < len(input); {
		c := input[i]

		// ASCII fast path

		if c < utf8.RuneSelf {
			// If input contains a forbidden host code point,
			if forbiddenHostASCII[c] {
				if p.LaxHostParsing {
					return string(input), nil
				}
				// host-invalid-code-point validation error, return failure.
				if err := p.handleErrorWithDescription(u, ErrHostInvalidCodePoint, byteCodePointDescription(c)); err != nil {
					return "", err
				}
			}

			// If input contains a code point that is not a URL code point and not U+0025 (%),
			if !urlCodePointASCII[c] && c != '%' {
				// invalid-URL-unit validation error.
				if err := p.handleErrorWithDescription(u, ErrInvalidURLUnit, byteCodePointDescription(c)); err != nil {
					return "", err
				}
			}

			// If input contains a U+0025 (%) and the two code points following it are not ASCII hex digits,
			if c == '%' {
				if i+2 >= len(input) || !isHex(input[i+1]) || !isHex(input[i+2]) {
					// Mirror your existing remainingIsInvalidPercentEncoded behavior:
					// Provide whatever description you currently build (here: just "%").

					// invalid-URL-unit validation error.
					if err := p.handleErrorWithDescription(u, ErrInvalidURLUnit, codePointDescription('%')); err != nil {
						return "", err
					}
					if p.PercentEncodeSinglePercentSign {
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
			if err := p.handleErrorWithDescription(u, ErrInvalidURLUnit, invalidUTF8ByteDescription(input[i])); err != nil {
				return "", err
			}
			// Encode the raw byte (not U+FFFD) if you accept invalid code points; otherwise encode U+FFFD.
			if p.AcceptInvalidCodepoints {
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
			if err := p.handleErrorWithDescription(u, ErrInvalidURLUnit, codePointDescription(r)); err != nil {
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
		idna.BidiRule(),
		idna.VerifyDNSLength(beStrict),
		idna.StrictDomainName(beStrict),
		idna.CheckHyphens(beStrict),
		idna.CheckJoiners(true),
		idna.Transitional(false),
	)
}

var (
	idnaStrict    = domainToASCIIProfile(true)
	idnaNonStrict = domainToASCIIProfile(false)
)

// domainToASCII converts a string to ASCII using IDNA
// https://url.spec.whatwg.org/#concept-domain-to-ascii
func domainToASCII(domain string, beStrict bool) (string, error) {
	if domain == "" {
		return "", nil
	}

	idnaProfile := idnaNonStrict
	if beStrict {
		idnaProfile = idnaStrict
	}

	// Convert to punycode
	a, err := idnaProfile.ToASCII(domain)
	if err != nil {
		return a, ErrDomainToASCII.Wrap(err)
	}

	if !beStrict {
		if a == "" {
			return "", ErrDomainToASCII
		}
		for _, r := range a {
			if r < utf8.RuneSelf && forbiddenDomainASCII[r] {
				return a, ErrDomainInvalidCodePoint
			}
		}
	}

	return a, nil
}

func asciiNoPunycodeLabels(s []byte) bool {
	p := 0
	for i := 0; i < len(s); i++ {
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
