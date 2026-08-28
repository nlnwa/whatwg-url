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

import "strconv"

// IPv4Addr is the 32-bit integer representation of an IPv4 address.
type IPv4Addr uint32

// String implements the IPv4 serializer.
// https://url.spec.whatwg.org/#concept-ipv4-serializer
func (address IPv4Addr) String() string {
	// Let output be the empty string and n be the value of address.
	// Implementation note: the standard prepends the four octets. Appending
	// them from most to least significant produces the same serialization
	// without moving bytes.
	var buf [15]byte
	b := buf[:0]
	b = strconv.AppendUint(b, uint64(address>>24), 10)
	b = append(b, '.')
	b = strconv.AppendUint(b, uint64((address>>16)&0xFF), 10)
	b = append(b, '.')
	b = strconv.AppendUint(b, uint64((address>>8)&0xFF), 10)
	b = append(b, '.')
	b = strconv.AppendUint(b, uint64(address&0xFF), 10)

	// Return output.
	return string(b)
}

type ipv6Addr [8]uint16

func (address *ipv6Addr) String() string {
	var buf [39]byte
	return string(address.serializeTo(buf[:0]))
}

func (address *ipv6Addr) StringWithBrackets() string {
	var buf [41]byte
	b := buf[:0]
	b = append(b, '[')
	b = address.serializeTo(b)
	b = append(b, ']')
	return string(b)
}

// serializeTo implements the IPv6 serializer.
// https://url.spec.whatwg.org/#concept-ipv6-serializer
func (address *ipv6Addr) serializeTo(output []byte) []byte {
	// Let output be the empty string.
	// Implementation note: callers supply the append target so bracketed host
	// serialization can reserve its prefix in the same stack buffer.

	// Let compress be the result of finding the IPv6 address compressed piece
	// index given address.
	compress := address.compressedPieceIndex()

	// Let ignore0 be false.
	ignore0 := false

	// For each pieceIndex of address's pieces' indices:
	for pieceIndex := range 8 {
		// If ignore0 is true and address[pieceIndex] is 0, then continue.
		if ignore0 && address[pieceIndex] == 0 {
			continue
		}

		// Otherwise, if ignore0 is true, set ignore0 to false.
		if ignore0 {
			ignore0 = false
		}

		// If compress is pieceIndex, then append "::" when pieceIndex is 0,
		// otherwise append U+003A (:), set ignore0 to true, and continue.
		if pieceIndex == compress {
			if pieceIndex == 0 {
				output = append(output, ':', ':')
			} else {
				output = append(output, ':')
			}
			ignore0 = true
			continue
		}

		// Append address[pieceIndex], represented as the shortest possible
		// lowercase hexadecimal number, to output.
		output = strconv.AppendUint(output, uint64(address[pieceIndex]), 16)

		// If pieceIndex is not 7, then append U+003A (:) to output.
		if pieceIndex != 7 {
			output = append(output, ':')
		}
	}

	// Return output.
	return output
}

// compressedPieceIndex finds the IPv6 address compressed piece index.
// https://url.spec.whatwg.org/#concept-ipv6-serializer-compress
func (address *ipv6Addr) compressedPieceIndex() int {
	// Let longestIndex be null and longestSize be 1.
	longestIndex, longestSize := -1, 1

	// Let foundIndex be null and foundSize be 0.
	foundIndex, foundSize := -1, 0

	// For each pieceIndex of address's pieces' indices:
	for pieceIndex := range 8 {
		// If address's piece at pieceIndex is not 0, update the longest run and
		// reset the found run.
		if address[pieceIndex] != 0 {
			if foundSize > longestSize {
				longestIndex, longestSize = foundIndex, foundSize
			}
			foundIndex, foundSize = -1, 0
			continue
		}

		// Otherwise, initialize foundIndex if null and increment foundSize.
		if foundIndex < 0 {
			foundIndex = pieceIndex
		}
		foundSize++
	}

	// If foundSize is greater than longestSize, then return foundIndex.
	if foundSize > longestSize {
		return foundIndex
	}

	// Return longestIndex.
	return longestIndex
}
