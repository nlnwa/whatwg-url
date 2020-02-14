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
	"strings"
	"unicode/utf8"
)

type inputString struct {
	runes   []rune
	pointer int
	eof     bool
	length  int
}

func newInputString(s string) *inputString {
	i := &inputString{runes: []rune(s), pointer: -1}
	i.length = len(i.runes)
	return i
}

func (i *inputString) nextCodePoint() rune {
	i.pointer++
	if i.pointer >= i.length {
		i.eof = true
		return utf8.RuneError
	}
	return i.runes[i.pointer]
}

func (i *inputString) rewindLast() {
	i.eof = false
	i.pointer--
}

func (i *inputString) reset() {
	i.pointer = -1
	i.eof = false
}

func (i *inputString) rewind(length int) {
	i.pointer -= length
	i.eof = false
}

func (i *inputString) remainingFromPointer() string {
	if i.eof {
		return ""
	}
	return string(i.runes[i.pointer:])
}

func (i *inputString) remainingStartsWith(s string) bool {
	if i.eof {
		return false
	}

	return strings.HasPrefix(string(i.runes[i.pointer+1:]), s)
}

func (i *inputString) remainingIsInvalidPercentEncoded() bool {
	if !i.eof && i.runes[i.pointer] == '%' &&
		(len(i.runes) < (i.pointer+3) ||
			(!ASCIIHexDigit.Test(uint(i.runes[i.pointer+1])) || !ASCIIHexDigit.Test(uint(i.runes[i.pointer+2])))) {
		return true
	}
	return false
}

func (i *inputString) String() string {
	return string(i.runes)
}
