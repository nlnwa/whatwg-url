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
	goerrors "errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"

	"github.com/nlnwa/whatwg-url/errors"
)

// parseHost parses the host part of the input string.
func (p *parser) parseHost(u *Url, parser *parser, input string, isNotSpecial bool) (string, error) {
	if p.opts.preParseHostFunc != nil {
		input = p.opts.preParseHostFunc(u, input)
	}
	if input == "" {
		return "", nil
	}
	if input[0] == '[' {
		if !strings.HasSuffix(input, "]") {
			return "", errors.Error(errors.IllegalIPv6Address, "")
		}
		input = strings.Trim(input, "[]")
		return p.parseIPv6(u, newInputString(input))
	}
	if isNotSpecial {
		return p.parseOpaqueHost(input)
	}

	domain, err := url.PathUnescape(input)
	if err != nil {
		if p.opts.laxHostParsing {
			return input, nil
		}
		return "", errors.Wrap(err, errors.CouldNotDecodeHost, "")
	}

	if !utf8.ValidString(domain) {
		if p.opts.laxHostParsing {
			return percentEncodeString(input, HostPercentEncodeSet), nil
		}
		return "", errors.ErrorWithDescr(errors.CouldNotDecodeHost, "not a valid UTF-8 string", "")
	}

	asciiDomain, err := p.ToASCII(domain)
	if err != nil {
		if p.opts.laxHostParsing {
			return domain, nil
		}
		return "", errors.Wrap(err, errors.CouldNotDecodeHost, "")
	}
	for _, c := range asciiDomain {
		if ForbiddenDomainCodePoint.Test(uint(c)) {
			if p.opts.laxHostParsing {
				return parser.PercentEncodeString(asciiDomain, HostPercentEncodeSet), nil
			} else {
				return "", errors.ErrorWithDescr(errors.IllegalCodePoint, string(c), "")
			}
		}
	}

	if p.endsInANumber(asciiDomain) {
		ipv4Host, err := p.parseIPv4(u, asciiDomain)
		return ipv4Host, err
	}

	if p.opts.postParseHostFunc != nil {
		asciiDomain = p.opts.postParseHostFunc(u, asciiDomain)
	}
	return asciiDomain, nil
}

func (p *parser) endsInANumber(input string) bool {
	parts := strings.Split(input, ".")
	if parts[len(parts)-1] == "" {
		if len(parts) == 1 {
			return false
		}
		parts = parts[0 : len(parts)-1]
	}
	last := parts[len(parts)-1]
	if last != "" && containsOnly(last, ASCIIDigit) {
		return true
	}
	if _, _, err := p.parseIPv4Number(last); err == nil || goerrors.Is(err, strconv.ErrRange) {
		return true
	}
	return false
}

func (p *parser) parseIPv4Number(input string) (number int64, validationError bool, err error) {
	if input == "" {
		err = errors.ErrorWithDescr(errors.CouldNotDecodeHost, "empty IPv4 number", "")
		return
	}
	R := 10
	if len(input) >= 2 && (strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X")) {
		validationError = true
		input = input[2:]
		R = 16
	} else if len(input) >= 2 && strings.HasPrefix(input, "0") {
		validationError = true
		input = input[1:]
		R = 8
	}
	if input == "" {
		validationError = true
		return
	}
	number, err = strconv.ParseInt(input, R, 64)
	return
}

func (p *parser) parseIPv4(u *Url, input string) (string, error) {
	parts := strings.Split(input, ".")
	if parts[len(parts)-1] == "" {
		if err := p.handleError(u, errors.IllegalIPv4Address); err != nil {
			return input, err
		}
		if len(parts) > 1 {
			parts = parts[:len(parts)-1]
		}
	}
	if len(parts) > 4 {
		_, err := p.handleFailure(u, errors.IllegalIPv4Address, fmt.Errorf("IPv4 too many parts"))
		return "", err
	}
	var numbers []int64
	for _, part := range parts {
		n, validationError, err := p.parseIPv4Number(part)
		if err != nil {
			return input, err
		}
		if validationError {
			if err := p.handleError(u, errors.IllegalIPv4Address); err != nil {
				return input, err
			}
		}
		numbers = append(numbers, n)
	}
	for _, n := range numbers {
		if n > 255 {
			if err := p.handleError(u, errors.IllegalIPv4Address); err != nil {
				return input, err
			}
		}
	}
	for _, n := range numbers[:len(numbers)-1] {
		if n > 255 {
			return "", errors.Error(errors.IllegalIPv4Address, "")
		}
	}
	if numbers[len(numbers)-1] >= int64(math.Pow(256, float64(5-len(numbers)))) {
		return "", errors.Error(errors.IllegalIPv4Address, "")
	}
	var ipv4 = IPv4Addr(numbers[len(numbers)-1])
	numbers = numbers[:len(numbers)-1]
	for counter, n := range numbers {
		ipv4 += IPv4Addr(n * int64(math.Pow(256, float64(3-counter))))
	}

	u.isIPv4 = true
	return ipv4.String(), nil
}

func (p *parser) parseIPv6(u *Url, input *inputString) (string, error) {
	address := &IPv6Addr{}
	pieceIdx := 0
	compress := -1

	c := input.nextCodePoint()
	if c == ':' {
		if !input.remainingStartsWith(":") {
			return "", errors.Error(errors.IllegalIPv6Address, "")
		}
		input.nextCodePoint()
		c = input.nextCodePoint()
		pieceIdx++
		compress = pieceIdx
	}
	for !input.eof {
		if pieceIdx == 8 {
			return "", errors.Error(errors.IllegalIPv6Address, "")
		}
		if c == ':' {
			if compress >= 0 {
				return "", errors.Error(errors.IllegalIPv6Address, "")
			}
			c = input.nextCodePoint()
			pieceIdx++
			compress = pieceIdx
			continue
		}
		value := 0
		length := 0

		for length < 4 && ASCIIHexDigit.Test(uint(c)) {
			v, _ := strconv.ParseInt(string(c), 16, 32)
			value = value*0x10 + int(v)
			c = input.nextCodePoint()
			length++
		}

		if c == '.' {
			if length == 0 {
				return "", errors.Error(errors.IllegalIPv6Address, "")
			}
			input.rewind(length + 1)
			c = input.nextCodePoint()
			if pieceIdx > 6 {
				return "", errors.Error(errors.IllegalIPv6Address, "")
			}
			numbersSeen := 0
			for !input.eof {
				ipv4Piece := -1
				if numbersSeen > 0 {
					if c == '.' && numbersSeen < 4 {
						c = input.nextCodePoint()
					} else {
						return "", errors.Error(errors.IllegalIPv6Address, "")
					}
				}
				if !ASCIIDigit.Test(uint(c)) {
					return "", errors.Error(errors.IllegalIPv6Address, "")
				}
				for ASCIIDigit.Test(uint(c)) {
					number, _ := strconv.Atoi(string(c))
					if ipv4Piece < 0 {
						ipv4Piece = number
					} else if ipv4Piece == 0 {
						return "", errors.Error(errors.IllegalIPv6Address, "")
					} else {
						ipv4Piece = ipv4Piece*10 + number
					}

					if ipv4Piece > 255 {
						return "", errors.Error(errors.IllegalIPv6Address, "")
					}
					c = input.nextCodePoint()
				}
				address[pieceIdx] = address[pieceIdx]*0x100 + uint16(ipv4Piece)
				numbersSeen++
				if numbersSeen == 2 || numbersSeen == 4 {
					pieceIdx++
				}
			}
			if numbersSeen != 4 {
				return "", errors.Error(errors.IllegalIPv6Address, "")
			}
			break
		} else if c == ':' {
			c = input.nextCodePoint()
			if input.eof {
				return "", errors.Error(errors.IllegalIPv6Address, "")
			}
		} else if !input.eof {
			return "", errors.Error(errors.IllegalIPv6Address, "")
		}
		address[pieceIdx] = uint16(value)
		pieceIdx++
	}
	if compress >= 0 {
		swaps := pieceIdx - compress
		pieceIdx = 7
		for pieceIdx != 0 && swaps > 0 {
			t := address[pieceIdx]
			address[pieceIdx] = address[compress+swaps-1]
			address[compress+swaps-1] = t
			pieceIdx--
			swaps--
		}
	} else if compress < 0 && pieceIdx != 8 {
		return "", errors.Error(errors.IllegalIPv6Address, "")
	}
	u.isIPv6 = true
	return "[" + address.String() + "]", nil
}

func (p *parser) parseOpaqueHost(input string) (string, error) {
	output := ""
	for _, c := range input {
		if ForbiddenHostCodePoint.Test(uint(c)) && c != '%' {
			if p.opts.laxHostParsing {
				return input, nil
			} else {
				return "", errors.ErrorWithDescr(errors.IllegalCodePoint, string(c), "")
			}
		}
		output += p.percentEncodeRune(c, C0PercentEncodeSet)
	}
	return output, nil
}

type IPv6Addr [8]uint16

func (address *IPv6Addr) String() string {
	output := ""
	compress := -1

	currentIdx := -1
	currentLength := 0
	compressLength := 0
	for pieceIdx := 0; pieceIdx < 8; pieceIdx++ {
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
		compressLength = currentLength
	}

	ignore0 := false
	for pieceIdx := 0; pieceIdx < 8; pieceIdx++ {
		if ignore0 && address[pieceIdx] == 0 {
			continue
		} else if ignore0 {
			ignore0 = false
		}
		if compress == pieceIdx {
			separator := ":"
			if pieceIdx == 0 {
				separator = "::"
			}
			output += separator
			ignore0 = true
			continue
		}
		output += strconv.FormatUint(uint64(address[pieceIdx]), 16)
		// 32512
		if pieceIdx != 7 {
			output += ":"
		}
	}
	return output
}

type IPv4Addr uint32

func (address *IPv4Addr) String() string {
	output := ""
	n := int(*address)
	for i := 1; i <= 4; i++ {
		output = strconv.Itoa(n%256) + output
		if i != 4 {
			output = "." + output
		}
		n = int(math.Floor(float64(n / 256)))
	}
	return output
}

var idnaProfile = idna.New(
	idna.MapForLookup(),
	idna.BidiRule(),
	idna.VerifyDNSLength(true),
	idna.StrictDomainName(false),
	idna.ValidateLabels(true),
	idna.CheckHyphens(false),
	idna.CheckJoiners(true),
	idna.Transitional(false),
)

func (p *parser) ToASCII(src string) (string, error) {
	// If encoding is set, convert to Unicode
	if p.opts.encodingOverride != nil {
		if u, err := p.stringToUnicode(src); err == nil {
			src = u
		}
	}

	// Convert to punycode
	a, err := idnaProfile.ToASCII(src)
	if err != nil {
		if !p.opts.laxHostParsing && !strings.Contains(err.Error(), src) {
			return "", err
		}
		a = p.PercentEncodeString(src, HostPercentEncodeSet)
	}
	return a, nil
}

func (p *parser) stringToUnicode(src string) (string, error) {
	var bb []byte
	for _, r := range []rune(src) {
		if b, ok := p.opts.encodingOverride.EncodeRune(r); ok && b > 31 {
			bb = append(bb, b)
		} else {
			return "", fmt.Errorf("could not conver %v to Unicode using %v", src, p.opts.encodingOverride.String())
		}
	}
	return string(bb), nil
}

func percentEncodeString(s string, tr *PercentEncodeSet) string {
	sb := strings.Builder{}
	for _, b := range []byte(s) {
		sb.WriteString(percentEncodeByte(b, tr))
	}
	return sb.String()
}

func percentEncodeByte(b byte, tr *PercentEncodeSet) string {
	if tr != nil && !tr.ByteShouldBeEncoded(b) {
		return string(b)
	}

	percentEncoded := make([]byte, 3)
	percentEncoded[0] = '%'
	percentEncoded[1] = "0123456789ABCDEF"[b>>4]
	percentEncoded[2] = "0123456789ABCDEF"[b&15]
	return string(percentEncoded)
}
