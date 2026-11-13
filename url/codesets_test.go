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
	"bytes"
	"testing"
)

func TestPercentEncodeSetBytes(t *testing.T) {
	allASCII := make([]byte, 128)
	for i := range allASCII {
		allASCII[i] = byte(i)
	}
	for _, test := range []struct {
		name string
		set  *PercentEncodeSet
		want []byte
	}{
		{"empty", NewPercentEncodeSet(0), nil},
		{"threshold", NewPercentEncodeSet(3), []byte{0, 1, 2}},
		{"overlapping entries", NewPercentEncodeSet(3, 1, 3, 5), []byte{0, 1, 2, 3, 5}},
		{"negative threshold", NewPercentEncodeSet(-1, 'A'), []byte{'A'}},
		{"threshold beyond ASCII", NewPercentEncodeSet(256), allASCII},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.set.Bytes(); !bytes.Equal(got, test.want) {
				t.Fatalf("Bytes() = %v, want %v", got, test.want)
			}
		})
	}
}

func Test_percentEncodeSet_RuneShouldBeEncoded(t *testing.T) {
	tests := []struct {
		name string
		pes  *PercentEncodeSet
		r    rune
		want bool
	}{
		{"C0-1", C0PercentEncodeSet, 0x00, true},
		{"C0-2", C0PercentEncodeSet, 0x20, false},
		{"C0-3", C0PercentEncodeSet, 0x7f, true},
		{"Fragment-1", FragmentPercentEncodeSet, 0x00, true},
		{"Fragment-2", FragmentPercentEncodeSet, 0x20, true},
		{"Fragment-3", FragmentPercentEncodeSet, 0x22, true},
		{"Fragment-4", FragmentPercentEncodeSet, 0x23, false},
		{"Fragment-5", FragmentPercentEncodeSet, 0x7f, true},
		{"Path-1", PathPercentEncodeSet, 0x00, true},
		{"Path-2", PathPercentEncodeSet, 0x20, true},
		{"Path-3", PathPercentEncodeSet, 0x22, true},
		{"Path-4", PathPercentEncodeSet, 0x23, true},
		{"Path-5", PathPercentEncodeSet, 0x7f, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pes.RuneShouldBeEncoded(tt.r); got != tt.want {
				t.Errorf("RuneShouldBeEncoded() = %v, want %v", got, tt.want)
			}
		})
	}
}
