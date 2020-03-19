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
	"fmt"
	"github.com/nlnwa/whatwg-url/errors"
	"golang.org/x/net/idna"
	"math"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (p *Parser) parseHost(u *Url, parser *Parser, input string, isNotSpecial bool) (string, error) {
	if p.opts.preParseHostFunc != nil {
		input = p.opts.preParseHostFunc(u, parser, input)
	}
	if input == "" {
		return "", nil
	}
	if input[0] == '[' {
		if !strings.HasSuffix(input, "]") {
			return "", errors.Error(errors.IllegalIPv6Address, "")
		}
		input = strings.Trim(input, "[]")
		return p.parseIPv6(newInputString(input, p.opts.acceptInvalidCodepoints))
	}
	if isNotSpecial {
		return p.parseOpaqueHost(input)
	}

	domain, err := url.QueryUnescape(input)
	if err != nil {
		if p.opts.laxHostParsing {
			return input, nil
		}
		return "", errors.Error(errors.CouldNotDecodeHost, "")
	}

	if !utf8.ValidString(domain) {
		if p.opts.laxHostParsing {
			return parser.PercentEncodeString(input, HostPercentEncodeSet), nil
		}
		return "", errors.Error(errors.CouldNotDecodeHost, "")
	}

	asciiDomain, err := p.ToASCII(domain)
	if err != nil {
		if p.opts.laxHostParsing {
			return domain, nil
		}
		return "", errors.Error(errors.CouldNotDecodeHost, "")
	}

	ipv4Host, ok, err := p.parseIPv4(u, asciiDomain)
	if ok || err != nil {
		return ipv4Host, err
	}
	if p.opts.postParseHostFunc != nil {
		asciiDomain = p.opts.postParseHostFunc(u, p, asciiDomain)
	}
	return asciiDomain, nil
}

func (p *Parser) parseIPv4Number(input string, validationError *bool) (int64, error) {
	R := 10
	if len(input) >= 2 && (strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X")) {
		*validationError = true
		input = input[2:]
		R = 16
	} else if len(input) >= 2 && strings.HasPrefix(input, "0") {
		*validationError = true
		input = input[1:]
		R = 8
	}
	if input == "" {
		return 0, nil
	}
	return strconv.ParseInt(input, R, 64)
}

func (p *Parser) parseIPv4(u *Url, input string) (string, bool, error) {
	validationError := false
	parts := strings.Split(input, ".")
	if parts[len(parts)-1] == "" {
		validationError = true
		if len(parts) > 1 {
			parts = parts[:len(parts)-1]
		}
	}
	if len(parts) > 4 {
		return input, false, nil
	}
	var numbers []int64
	for _, part := range parts {
		if part == "" {
			return input, false, nil
		}
		n, err := p.parseIPv4Number(part, &validationError)
		if err != nil {
			return input, false, nil
		}
		numbers = append(numbers, n)
	}
	if validationError {
		if err := p.handleError(u, errors.IllegalIPv4Address); err != nil {
			return input, false, err
		}
	}
	for _, n := range numbers {
		if n > 255 {
			if err := p.handleError(u, errors.IllegalIPv4Address); err != nil {
				return input, false, err
			}
		}
	}
	for _, n := range numbers[:len(numbers)-1] {
		if n > 255 {
			return "", false, errors.Error(errors.IllegalIPv4Address, "")
		}
	}
	if numbers[len(numbers)-1] >= int64(math.Pow(256, float64(5-len(numbers)))) {
		return "", false, errors.Error(errors.IllegalIPv4Address, "")
	}
	var ipv4 = IPv4Addr(numbers[len(numbers)-1])
	numbers = numbers[:len(numbers)-1]
	for counter, n := range numbers {
		ipv4 += IPv4Addr(n * int64(math.Pow(256, float64(3-counter))))
	}

	return ipv4.String(), true, nil
}

func (p *Parser) parseIPv6(input *inputString) (string, error) {
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
	return "[" + address.String() + "]", nil
}

func (p *Parser) parseOpaqueHost(input string) (string, error) {
	output := ""
	for _, c := range input {
		if ForbiddenHostCodePoint.Test(uint(c)) && c != '%' {
			return "", fmt.Errorf("illegal IPv6 address '%v'", input)
		}
		output += p.percentEncode(c, C0PercentEncodeSet)
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
)

func (p *Parser) ToASCII(src string) (string, error) {
	a, err := idnaProfile.ToASCII(src)
	if err != nil {
		if !p.opts.laxHostParsing && !strings.Contains(err.Error(), src) {
			return "", err
		}
		a = p.PercentEncodeString(src, HostPercentEncodeSet)
	}
	return a, nil
}
