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

package url

import (
	"github.com/willf/bitset"
	"unicode"
)

type percentEncodeSet struct {
	*bitset.BitSet
}

func (p *percentEncodeSet) runeInSet(r rune) bool {
	if r > 0x7E || p.Test(uint(r)) {
		return true
	}
	return false
}

func isURLCodePoint(r rune) bool {
	if ASCIIAlphanumeric.Test(uint(r)) {
		return true
	}
	if someURLCodePoints.Test(uint(r)) {
		return true
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

var C0 = bitset.New(0x1f).Set(0x00).Set(0x01).Set(0x02).Set(0x03).Set(0x04).Set(0x05).Set(0x06).
	Set(0x07).Set(0x08).Set(0x09).Set(0x0a).Set(0x0b).Set(0x0c).Set(0x0d).Set(0x0e).Set(0x0f).
	Set(0x10).Set(0x11).Set(0x12).Set(0x13).Set(0x14).Set(0x15).Set(0x16).Set(0x17).Set(0x18).
	Set(0x19).Set(0x1a).Set(0x1b).Set(0x1c).Set(0x1d).Set(0x1e).Set(0x1f)
var C0OrSpace = bitset.New(0x20).Set(0x20)
var ASCIITabOrNewline = bitset.New(0x0d).Set(0x09).Set(0x0a).Set(0x0d)
var ASCIIUpperAlpha = bitset.New(0x5a)
var ASCIILowerAlpha = bitset.New(0x7a)
var ASCIIAlpha = bitset.New(0x7a)
var ASCIIDigit = bitset.New(0x39)
var ASCIIUpperHexDigit = bitset.New(0x46)
var ASCIILowerHexDigit = bitset.New(0x66)
var ASCIIHexDigit = bitset.New(0x66)
var ASCIIAlphanumeric = bitset.New(0x7a)
var C0PercentEncodeSet = &percentEncodeSet{C0}
var FragmentPercentEncodeSet = &percentEncodeSet{C0.Clone().Set(0x20).Set(0x22).Set(0x3c).Set(0x3e).Set(0x60)}
var PathPercentEncodeSet = &percentEncodeSet{FragmentPercentEncodeSet.BitSet.Clone().Set(0x23).Set(0x3f).Set(0x7b).Set(0x7d)}
var UserInfoPercentEncodeSet = &percentEncodeSet{PathPercentEncodeSet.BitSet.Clone().Set(0x2f).Set(0x3a).
	Set(0x3b).Set(0x3d).Set(0x40).Set(0x5b).Set(0x5c).Set(0x5d).Set(0x5e).Set(0x7c)}
var ForbiddenHostCodePoint = bitset.New(0x5d).Set(0x00).Set(0x09).Set(0x0a).Set(0x0d).Set(0x20).
	Set(0x23).Set(0x25).Set(0x2f).Set(0x3a).Set(0x3f).Set(0x40).Set(0x5b).Set(0x5c).Set(0x5d)

var someURLCodePoints = bitset.New(0x7e).Set(0x24).Set(0x26).Set(0x27).Set(0x28).Set(0x29).
	Set(0x2a).Set(0x2b).Set(0x2c).Set(0x2d).Set(0x2e).Set(0x2f).Set(0x3a).Set(0x3b).Set(0x3d).
	Set(0x3f).Set(0x40).Set(0x5f).Set(0x7e)

func init() {
	C0OrSpace.InPlaceUnion(C0)
	for i := 'a'; i <= 'z'; i++ {
		ASCIILowerAlpha.Set(uint(i))
	}
	for i := 'A'; i <= 'Z'; i++ {
		ASCIIUpperAlpha.Set(uint(i))
	}
	ASCIIAlpha.InPlaceUnion(ASCIILowerAlpha)
	ASCIIAlpha.InPlaceUnion(ASCIIUpperAlpha)
	for i := '0'; i <= '9'; i++ {
		ASCIIDigit.Set(uint(i))
	}
	ASCIIAlphanumeric.InPlaceUnion(ASCIIAlpha)
	ASCIIAlphanumeric.InPlaceUnion(ASCIIDigit)

	ASCIIUpperHexDigit.InPlaceUnion(ASCIIDigit)
	for i := 'A'; i <= 'F'; i++ {
		ASCIIUpperHexDigit.Set(uint(i))
	}

	ASCIILowerHexDigit.InPlaceUnion(ASCIIDigit)
	for i := 'a'; i <= 'f'; i++ {
		ASCIILowerHexDigit.Set(uint(i))
	}

	ASCIIHexDigit.InPlaceUnion(ASCIIUpperHexDigit)
	ASCIIHexDigit.InPlaceUnion(ASCIILowerHexDigit)
}
