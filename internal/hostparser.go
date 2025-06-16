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
	gerrors "errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"

	"github.com/nationallibraryofnorway/whatwg-url/errors"
)

// parseHost parses the host part of the input string.
//
// The host parser takes a scalar value string input with an optional boolean isOpaque (default false), and then runs these steps. They return failure or a host.
func (p *Parser) parseHost(u *Url, input string, isNotSpecial bool) (string, error) {
	if p.PreParseHostFunc != nil {
		input = p.PreParseHostFunc(u, input)
	}

	// If input starts with U+005B ([), then:
	if len(input) > 0 && input[0] == '[' {
		// If input does not end with U+005D (]),
		if input[len(input)-1] != ']' {
			// IPv6-unclosed validation error, return failure.
			if err := p.handleError(u, errors.ErrIPv6Unclosed); err != nil {
				return "", err
			}
			return "", errors.ErrIPv6Unclosed
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
		return "[" + addr.String() + "]", nil
	}

	// If isOpaque is true, then return the result of opaque-host parsing input.
	if isNotSpecial {
		return p.parseOpaqueHost(u, input)
	}

	// Assert: input is not the empty string.
	if input == "" {
		return "", nil
	}

	// Let domain be the result of running UTF-8 decode without BOM on the percent-decoding of input.

	// Let asciiDomain be the result of running domain to ASCII with domain and false.
	asciiDomain, err := p.hostToASCII(input)
	// If asciiDomain is failure,
	if err != nil {
		if p.LaxHostParsing {
			return input, nil
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

func (p *Parser) hostToASCII(input string) (string, error) {
	domain := percentDecode(input, p.EncodingOverride)

	if !isASCII(domain) && !utf8.ValidString(domain) {
		return "", errors.ErrInvalidEncoding
	}

	// Fast path: ASCII-only and no punycode labels.
	// (asciiNoPctNoPunycode assumes ASCII; it returns false if any non-ASCII.)
	if asciiNoPctNoPunycode(domain) {
		return validateAndASCIILower(domain)
	}

	return domainToASCII(domain, false)
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

	return gerrors.Is(err, errors.ErrIPv4OutOfRangePart)
}

func (p *Parser) parseIPv4(u *Url, input string) (string, error) {
	// Let parts be the result of strictly splitting input on U+002E (.).
	parts := strings.Split(input, ".")
	// If the last item in parts is the empty string, then:
	if parts[len(parts)-1] == "" {
		// IPv4-empty-part validation error.
		if err := p.handleError(u, errors.ErrIPv4EmptyPart); err != nil {
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
		if err := p.handleError(u, errors.ErrIPv4TooManyParts); err != nil {
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
			if err := p.handleError(u, errors.ErrIPv4NonNumericPart); err != nil {
				return input, err
			}
		}
		// If result[1] is true,
		if validationError {
			// IPv4-non-decimal-part validation error.
			if err := p.handleError(u, errors.ErrIPv4NonDecimalPart); err != nil {
				return input, err
			}
		}
		// Append result[0] to numbers.
		numbers = append(numbers, n)
	}
	for _, n := range numbers {
		// If any item in numbers is greater than 255,
		if n > 255 {
			// IPv4-out-of-range-part validation error.
			err := p.handleError(u, errors.ErrIPv4OutOfRangePart)
			if err != nil && p.FailOnValidationError {
				return input, err
			}
		}
	}
	for _, n := range numbers[:len(numbers)-1] {
		// If any but the last item in numbers is greater than 255,
		if n > 255 {
			// then return failure.
			if err := p.handleError(u, errors.ErrIPv4OutOfRangePart); err != nil {
				return input, err
			}
		}
	}
	// If the last item in numbers is greater than or equal to 256(5 − numbers’s size),
	if numbers[len(numbers)-1] >= int64(math.Pow(256, float64(5-len(numbers)))) {
		// then return failure.
		if err := p.handleError(u, errors.ErrIPv4OutOfRangePart); err != nil {
			return input, err
		}
	}
	// Let ipv4 be the last item in numbers.
	var ipv4 = IPv4Addr(numbers[len(numbers)-1])
	// Remove the last item from numbers.
	numbers = numbers[:len(numbers)-1]
	// Let counter be 0.
	// For each n of numbers:
	for counter, n := range numbers {
		// Increment ipv4 by n × 256(3 − counter).
		ipv4 += IPv4Addr(n * int64(math.Pow(256, float64(3-counter))))
		// Increment counter by 1.
	}
	// Return ipv4.
	u.isIPv4 = true
	return ipv4.String(), nil
}

func parseIPv6(input string) (*ipv6Addr, error) {
	if len(input) < 2 {
		return nil, errors.ErrIPv6TooFewPieces
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
			return nil, errors.ErrIPv6InvalidCompression
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
			return nil, errors.ErrIPv6TooManyPieces
		}
		// If c is U+003A (:), then:
		if c == ':' {
			// If compress is non-null,
			if compress >= 0 {
				// IPv6-multiple-compression validation error, return failure.
				return nil, errors.ErrIPv6MultipleCompression
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
				return nil, errors.ErrIPv4InIPv6InvalidCodePoint
			}
			// Decrease pointer by length.
			i -= length
			// If pieceIndex is greater than 6, IPv4-in-IPv6-too-many-pieces validation error,
			if pieceIdx > 6 {
				// return failure.
				return nil, errors.ErrIPv4InIPv6TooManyPieces
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
							return nil, errors.ErrIPv4InIPv6InvalidCodePoint
						}
						c = input[i]
					} else {
						// Otherwise, IPv4-in-IPv6-invalid-code-point validation error, return failure.
						return nil, errors.ErrIPv4InIPv6InvalidCodePoint
					}
				}
				// If c is not an ASCII digit,
				if !isASCIIDigit(c) {
					// IPv4-in-IPv6-invalid-code-point validation error, return failure.
					return nil, errors.ErrIPv4InIPv6InvalidCodePoint
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
						return nil, errors.ErrIPv4InIPv6InvalidCodePoint
					} else { // Otherwise,
						// set ipv4Piece to ipv4Piece × 10 + number.
						ipv4Piece = ipv4Piece*10 + number
					}
					// If ipv4Piece is greater than 255,
					if ipv4Piece > 255 {
						// IPv4-in-IPv6-out-of-range-part validation error, return failure.
						return nil, errors.ErrIPv4InIPv6OutOfRangePart
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
				return nil, errors.ErrIPv4InIPv6TooFewParts
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
				return nil, errors.ErrIPv6InvalidCodePoint
			}
		} else
		// Otherwise, if c is not the EOF code point,
		if !eof {
			// IPv6-invalid-code-point validation error, return failure.
			return nil, errors.ErrIPv6InvalidCodePoint
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
		return nil, errors.ErrIPv6TooFewPieces
	}
	// Return address.
	return address, nil
}

func (p *Parser) parseOpaqueHost(u *Url, input string) (string, error) {
	out := make([]byte, 0, len(input)*3)

	for i, r := range input {
		if r < utf8.RuneSelf && forbiddenHostASCII[r] {
			if p.LaxHostParsing {
				return input, nil
			}
			if err := p.handleErrorWithDescription(u, errors.ErrHostInvalidCodePoint, string(r)); err != nil {
				return "", err
			}
		}
		if !isURLCodePoint(r) && r != '%' {
			if err := p.handleErrorWithDescription(u, errors.ErrInvalidURLUnit, string(r)); err != nil {
				return "", err
			}
		}
		if r == '%' {
			invalidPercentEncoding, d := remainingIsInvalidPercentEncoded(input[i:])
			if invalidPercentEncoding {
				if err := p.handleErrorWithDescription(u, errors.ErrInvalidURLUnit, d); err != nil {
					return "", err
				}
				if p.PercentEncodeSinglePercentSign {
					out = append(out, percentPercentEncoded...)
					continue
				}
			}
		}

		out = appendPercentEncoded(out, r, C0PercentEncodeSet)
	}

	return string(out), nil
}

// The IPv4 number parser takes an ASCII string input and then runs these
// steps. They return failure or a tuple of a number and a boolean.
func parseIPv4Number(input string) (number int64, validationError bool, err error) {
	// If input is the empty string, then return failure.
	if input == "" {
		return 0, false, errors.ErrIPv4EmptyPart
	}
	// Let validationError be false.

	// Let R be 10.
	R := 10

	// If input contains at least two code points and the first two code points
	// are either "0X" or "0x", then:
	if len(input) >= 2 && (input[:2] == "0x" || input[:2] == "0X") {
		// Set validationError to true.
		validationError = true
		// Remove the first two code points from input.
		input = input[2:]
		// Set R to 16.
		R = 16
	} else
	// Otherwise, if input contains at least two code points and the first code
	// point is U+0030 (0), then:
	if len(input) >= 2 && input[:1] == "0" {
		// Set validationError to true.
		validationError = true
		// Remove the first code point from input.
		input = input[1:]
		// Set R to 8.
		R = 8
	}
	// If input is the empty string,
	if input == "" {
		// then return (0, true).
		return 0, true, nil
	}
	// If input contains a code point that is not a radix-R digit, then return failure.
	switch R {
	case 8, 10:
		if !onlyASCIIDigits(input) {
			err = errors.ErrIPv4NonNumericPart
			return
		}
	case 16:
		if !onlyASCIIHexDigits(input) {
			err = errors.ErrIPv4NonNumericPart
			return
		}
	}
	// Let output be the mathematical integer value that is represented by
	// input in radix-R notation, using ASCII hex digits for digits with values 0 through 15.
	number, err = strconv.ParseInt(input, R, 64)
	if gerrors.Is(err, strconv.ErrRange) {
		err = errors.ErrIPv4OutOfRangePart
	}
	// Return (output, validationError).
	return
}

type ipv6Addr [8]uint16

func (address *ipv6Addr) String() string {
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

	var buf [40]byte
	b := buf[:0]

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

	return string(b)
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
		return a, errors.ErrDomainToASCII.Wrap(err)
	}

	if !beStrict {
		if a == "" {
			return "", errors.ErrDomainToASCII
		}
		for _, r := range a {
			if r < utf8.RuneSelf && forbiddenDomainASCII[r] {
				return a, errors.ErrDomainInvalidCodePoint
			}
		}
	}

	return a, nil
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// Reject forbidden domain code points and lowercase ASCII without an extra pass.
// Only allocates if it actually needs to lowercase something.
func validateAndASCIILower(s string) (string, error) {
	if s == "" {
		return "", errors.ErrDomainToASCII
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		if forbiddenDomainASCII[c] {
			return s, errors.ErrDomainInvalidCodePoint
		}

		if 'A' <= c && c <= 'Z' {
			b := []byte(s)
			b[i] = c + ('a' - 'A')

			for j := i + 1; j < len(b); j++ {
				d := b[j]
				if forbiddenDomainASCII[d] {
					return s, errors.ErrDomainInvalidCodePoint
				}
				if 'A' <= d && d <= 'Z' {
					b[j] = d + ('a' - 'A')
				}
			}
			return string(b), nil
		}
	}

	return s, nil
}

func asciiNoPctNoPunycode(s string) bool {
	p := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0x80 { // non-ASCII
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
