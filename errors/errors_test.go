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

package errors

import (
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		url       string
		failure   bool
	}{
		{"1", HostInvalidCodePoint, "http://example.com\t", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Error(tt.errorType, tt.url, false)
			fmt.Printf("Error: %s\n", err)
			fmt.Printf("Type: %s\n", Type(err))
			fmt.Printf("Url: %s\n", Url(err))
			//if err := Error(tt.args.code, tt.args.url); (err != nil) != tt.wantErr {
			//	t.Errorf("Error() error = %v, wantErr %v", err, tt.wantErr)
			//}
		})
	}
}
