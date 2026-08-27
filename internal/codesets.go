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

package internal

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

type PercentEncodeSet struct {
	allBelow rune
	ascii    [128]bool
}

// NewPercentEncodeSet returns a new PercentEncodeSet with the given rune values in the set.
//
// allBelow specifies that all runes below the given value are in the set.
// Values outside the single-byte range are ignored.
func NewPercentEncodeSet(allBelow rune, chars ...rune) *PercentEncodeSet {
	p := &PercentEncodeSet{allBelow: allBelow}
	for _, ch := range chars {
		if ch >= 0 && ch < utf8.RuneSelf {
			p.ascii[ch] = true
		}
	}
	return p
}

// Set returns a new PercentEncodeSet with the given rune values added.
// Values outside the single-byte range are ignored.
func (p *PercentEncodeSet) Set(chars ...rune) *PercentEncodeSet {
	r := &PercentEncodeSet{
		allBelow: p.allBelow,
		ascii:    p.ascii,
	}
	for _, ch := range chars {
		if ch >= 0 && ch < utf8.RuneSelf {
			r.ascii[ch] = true
		}
	}
	return r
}

// Clear returns a new PercentEncodeSet with the given rune values cleared.
// Values outside the single-byte range are ignored.
func (p *PercentEncodeSet) Clear(chars ...rune) *PercentEncodeSet {
	r := &PercentEncodeSet{
		allBelow: p.allBelow,
		ascii:    p.ascii,
	}
	for _, ch := range chars {
		if ch >= 0 && ch < utf8.RuneSelf {
			r.ascii[ch] = false
		}
	}
	return r
}

// Bytes returns the explicit ASCII bytes configured in this set.
func (p *PercentEncodeSet) Bytes() []byte {
	var result []byte
	for i := range max(p.allBelow-1, 0) {
		result = append(result, byte(i))
	}
	for i, enabled := range p.ascii {
		if enabled {
			result = append(result, byte(i))
		}
	}
	return result
}

func (p *PercentEncodeSet) RuneShouldBeEncoded(r rune) bool {
	if r < p.allBelow {
		return true
	}
	if r > 0x7E {
		return true
	}
	return p.ascii[r]
}

func (p *PercentEncodeSet) ByteShouldBeEncoded(b byte) bool {
	if rune(b) < p.allBelow {
		return true
	}
	if b > 0x7E {
		return true
	}
	return p.ascii[b]
}

func (p *PercentEncodeSet) RuneNotInSet(r rune) bool {
	if r < p.allBelow {
		return false
	}
	return !p.ascii[r]
}

var C0PercentEncodeSet = NewPercentEncodeSet(0x20)
var C0OrSpacePercentEncodeSet = NewPercentEncodeSet(0x21)
var FragmentPercentEncodeSet = C0OrSpacePercentEncodeSet.Set(0x22, 0x3c, 0x3e, 0x60)
var QueryPercentEncodeSet = C0OrSpacePercentEncodeSet.Set(0x22, 0x23, 0x3c, 0x3e)
var SpecialQueryPercentEncodeSet = QueryPercentEncodeSet.Set(0x27)
var PathPercentEncodeSet = QueryPercentEncodeSet.Set(0x3f, 0x5e, 0x60, 0x7b, 0x7d)
var UserInfoPercentEncodeSet = PathPercentEncodeSet.Set(0x2f, 0x3a, 0x3b, 0x3d, 0x40, 0x5b, 0x5c, 0x5d, 0x7c)
var HostPercentEncodeSet *PercentEncodeSet

var LaxPathPercentEncodeSet = PathPercentEncodeSet.Clear(0x2e, 0x3c, 0x3e, 0x5e)
var LaxQueryPercentEncodeSet = QueryPercentEncodeSet.Clear(0x22, 0x25, 0x2f, 0x3b, 0x3f, 0x7b)
var RepeatedQueryPercentEncodeSet = C0OrSpacePercentEncodeSet.Set('#', '%', '&', '=')

const percentPercentEncoded = "%25"
const spacePercentEncoded = "%20"

var urlCodePointASCII [128]bool
var forbiddenHostASCII [128]bool
var forbiddenDomainASCII [128]bool

func init() {
	HostPercentEncodeSet = NewPercentEncodeSet(0x2D)
	for i := 0x2d; i < 0x7f; i++ {
		if i == '-' || i == '.' ||
			(i >= '0' && i <= '9') ||
			(i >= 'a' && i <= 'z') ||
			(i >= 'A' && i <= 'Z') {
			continue
		}
		HostPercentEncodeSet = HostPercentEncodeSet.Set(rune(i))
	}

	for b := byte('0'); b <= '9'; b++ {
		urlCodePointASCII[b] = true
	}
	for b := byte('a'); b <= 'z'; b++ {
		urlCodePointASCII[b] = true
	}
	for b := byte('A'); b <= 'Z'; b++ {
		urlCodePointASCII[b] = true
	}

	// "someURLCodePoints" set:
	for _, b := range []byte{
		'$', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/', ':', ';', '=',
		'?', '@', '_', '~',
	} {
		urlCodePointASCII[b] = true
	}

	for i := 0x00; i <= 0x1f; i++ {
		forbiddenDomainASCII[byte(i)] = true
	}

	// plus specific punctuation
	for _, b := range []byte{0x00, 0x09, 0x0a, 0x0d, 0x20,
		'#', '/', ':', '<', '>', '?', '@', '[', '\\', ']', '^', '|'} {
		forbiddenHostASCII[b] = true
		forbiddenDomainASCII[b] = true
	}
	for _, b := range []byte{'%', 0x7f} {
		forbiddenDomainASCII[b] = true
	}
}

func isURLCodePoint(r rune) bool {
	if r < utf8.RuneSelf {
		return urlCodePointASCII[r]
	}
	if r >= 0xa0 && r <= 0x10fffd {
		if unicode.Is(unicode.Noncharacter_Code_Point, r) {
			return false
		}
		if unicode.Is(unicode.Cs, r) {
			return false
		}
		return true
	}
	return false
}

func isASCIIAlpha[T ByteRune](b T) bool {
	return ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}

func isASCIIDigit[T ByteRune](b T) bool {
	return b >= '0' && b <= '9'
}

func isASCIIAlphanumericPlus[T ByteRune](b T) bool {
	return isASCIIAlpha(b) || isASCIIDigit(b) || b == '+' || b == '-' || b == '.'
}

type ByteRune interface {
	~byte | ~rune
}

func isHex[T ByteRune](b T) bool {
	return ('0' <= b && b <= '9') ||
		('a' <= b && b <= 'f') ||
		('A' <= b && b <= 'F')
}

func onlyASCIIDigits[T ByteSequence](s T) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func onlyASCIIHexDigits[T ByteSequence](s T) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if !(('0' <= b && b <= '9') ||
			('a' <= b && b <= 'f') ||
			('A' <= b && b <= 'F')) {
			return false
		}
	}
	return true
}

func (c *Canonicalizer) repeatDecodeThenEncode(input string, tr *PercentEncodeSet) string {
	decoded := repeatPercentDecodeString(input, c.RepeatedPercentDecodeEncoding)
	return PercentEncode(decoded, tr)
}

func hasPctXX(s string) bool {
	// Scan for a '%' followed by two hex digits.
	for i := 0; i+2 < len(s); i++ {
		if s[i] == '%' && isHex(s[i+1]) && isHex(s[i+2]) {
			return true
		}
	}
	return false
}

func repeatPercentDecodeString(input string, encoding *charmap.Charmap) string {
	prev := input
	for {
		if !hasPctXX(prev) {
			return prev
		}
		decoded := percentDecode(prev, encoding)
		if prev == decoded {
			return decoded
		}
		prev = decoded
	}
}

type ByteSequence interface {
	~string | ~[]byte
}

func percentEncodeByte(b byte, tr *PercentEncodeSet) []byte {
	// Fast path: no encoding needed.
	if tr == nil || !tr.ByteShouldBeEncoded(b) {
		return []byte{b}
	}

	// Worst case: each byte becomes 3 bytes: %XX
	out := make([]byte, 0, 3)
	out = append(out, '%', upperHex[b>>4], upperHex[b&0x0f])

	return out
}

func percentEncode(r rune, tr *PercentEncodeSet) []byte {
	var tmp [utf8.UTFMax]byte
	n := utf8.EncodeRune(tmp[:], r)

	// Fast path: no encoding needed.
	if tr == nil || !tr.RuneShouldBeEncoded(r) {
		return tmp[:n]
	}

	// Worst case: each byte becomes 3 bytes: %XX
	out := make([]byte, 0, n*3)
	for i := range n {
		b := tmp[i]
		if !tr.ByteShouldBeEncoded(b) {
			out = append(out, b)
			continue
		}
		out = append(out, '%', upperHex[b>>4], upperHex[b&0x0f])
	}

	return out
}

// PercentDecodeInPlace percent-decodes b in-place and returns the resulting subslice.
// It implements the WHATWG-style rule:
// - "%"+2 hex digits => replaced with that byte
// - otherwise '%' is left as-is.
func PercentDecodeInPlace(b []byte) []byte {
	w := 0
	for r := 0; r < len(b); r++ {
		if b[r] == '%' && r+2 < len(b) && isHex(b[r+1]) && isHex(b[r+2]) {
			b[w] = (unhex(b[r+1]) << 4) | unhex(b[r+2])
			w++
			r += 2
			continue
		}
		b[w] = b[r]
		w++
	}
	return b[:w]
}

func percentDecode[T ByteSequence](raw T, encoding *charmap.Charmap) string {
	var out strings.Builder
	out.Grow(len(raw))

	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '%' ||
			i+2 >= len(raw) ||
			!isHex(raw[i+1]) ||
			!isHex(raw[i+2]) {

			out.WriteByte(c)
			continue
		}

		b := (unhex(raw[i+1]) << 4) | unhex(raw[i+2])
		if encoding != nil {
			out.WriteRune(encoding.DecodeByte(b))
		} else {
			out.WriteByte(b)
		}
		i += 2
	}
	return out.String()
}

func unhex[T ByteRune](c T) T {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

const upperHex = "0123456789ABCDEF"

// appendPercentEncoded appends r to dst, percent-encoding bytes as required by tr.
func appendPercentEncoded(dst []byte, r rune, set *PercentEncodeSet) []byte {
	if set == nil {
		return utf8.AppendRune(dst, r)
	}

	// ASCII fast path: decide on the single byte.
	if r < utf8.RuneSelf {
		b := byte(r)
		if !set.ByteShouldBeEncoded(b) {
			return append(dst, b)
		}
		return append(dst, '%', upperHex[b>>4], upperHex[b&0x0f])
	}

	// Non-ASCII: UTF-8 encode then decide per byte.
	var tmp [utf8.UTFMax]byte
	n := utf8.EncodeRune(tmp[:], r)
	for i := 0; i < n; i++ {
		b := tmp[i]
		if !set.ByteShouldBeEncoded(b) {
			dst = append(dst, b)
		} else {
			dst = append(dst, '%', upperHex[b>>4], upperHex[b&0x0f])
		}
	}
	return dst
}

// remainingIsInvalidPercentEncoded returns true if the first three characters in the input string are not '%' followed by two hex digits.
// If true, the second return value is the invalid percent encoded string.
func remainingIsInvalidPercentEncoded(input string) (bool, string) {
	if len(input) < 3 || input[0] != '%' {
		return true, ""
	}
	if !isHex(input[1]) || !isHex(input[2]) {
		l := min(len(input), 3)
		return true, input[0:l]
	}
	return false, ""
}

func PercentEncode[T ByteSequence](input T, tr *PercentEncodeSet) string {
	var buf strings.Builder
	buf.Grow(len(input) * 3)

	if tr == nil {
		for i := 0; i < len(input); i++ {
			b := input[i]
			buf.WriteByte('%')
			buf.WriteByte(upperHex[b>>4])
			buf.WriteByte(upperHex[b&0x0f])
		}
		return buf.String()
	}

	for i := 0; i < len(input); i++ {
		b := input[i]
		if !tr.ByteShouldBeEncoded(b) {
			buf.WriteByte(b)
			continue
		}
		buf.WriteByte('%')
		buf.WriteByte(upperHex[b>>4])
		buf.WriteByte(upperHex[b&0x0f])
	}

	return buf.String()
}
