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
	"reflect"
	"testing"
)

func TestUrlSearchParams_Get(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		key     string
		want    string
		wantHas bool
	}{
		{"1", "http://example.com?foo=bar", "foo", "bar", true},
		{"2", "http://example.com?foo=bar&foo2=bar2", "foo", "bar", true},
		{"3", "http://example.com?foo=bar&foo=bar2", "foo", "bar", true},
		{"4", "http://example.com?foo=bar2&foo=bar", "foo", "bar2", true},
		{"5", "http://example.com?foo=bar2&foo=bar", "foo2", "", false},
		{"6", "http://example.com?foo=bar2&foo2", "foo2", "", true},
		{"7", "http://example.com/", "foo2", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			if got := s.Get(tt.key); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
			if got := s.Has(tt.key); got != tt.wantHas {
				t.Errorf("Get() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestUrlSearchParams_GetAll(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		sort           bool
		want           []string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", false, []string{"bar"}, "foo=bar"},
		{"2", "http://example.com?foo=bar&foo2=bar2", "foo", false, []string{"bar"}, "foo=bar&foo2=bar2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo", false, []string{"bar"}, "foo2=bar2&foo=bar"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo", false, []string{"bar", "bar2"}, "foo=bar&foo=bar2"},
		{"5", "http://example.com?foo=bar2&foo=bar", "foo", false, []string{"bar2", "bar"}, "foo=bar2&foo=bar"},
		{"6", "http://example.com?foo=bar", "foo", true, []string{"bar"}, "foo=bar"},
		{"7", "http://example.com?foo=bar&foo2=bar2", "foo", true, []string{"bar"}, "foo=bar&foo2=bar2"},
		{"8", "http://example.com?foo2=bar2&foo=bar", "foo", true, []string{"bar"}, "foo=bar&foo2=bar2"},
		{"9", "http://example.com?foo=bar&foo=bar2", "foo", true, []string{"bar", "bar2"}, "foo=bar&foo=bar2"},
		{"10", "http://example.com?foo=bar2&foo=bar", "foo", true, []string{"bar2", "bar"}, "foo=bar2&foo=bar"},
		{"11", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo", true, []string{"bar2", "bar"}, "foo=bar2&foo=bar&xyz=aaa&xyz=aaa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			if tt.sort {
				s.Sort()
			}
			if got := s.GetAll(tt.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_Append(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		value          string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", "bar2", "foo=bar&foo=bar2"},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "abc", "xyz", "foo=bar&foo+2=bar+2&abc=xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			s.Append(tt.key, tt.value)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_Delete(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", ""},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "foo", "foo+2=bar+2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo", "foo2=bar2"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo", ""},
		{"5", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo", "xyz=aaa&xyz=aaa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			s.Delete(tt.key)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_Set(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		key            string
		value          string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo", "bar2", "foo=bar2"},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "foo", "xyz", "foo=xyz&foo+2=bar+2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo", "xyz", "foo2=bar2&foo=xyz"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo", "xyz", "foo=xyz"},
		{"5", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo", "xyz", "xyz=aaa&foo=xyz&xyz=aaa"},
		{"6", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "foo2", "xyz", "xyz=aaa&foo=bar2&xyz=aaa&foo=bar&foo2=xyz"},
		{"7", "http://example.com?foo=bar&foo=fuzz&foo=barfuzz", "foo", "xyz", "foo=xyz"},
		{"8", "http://example.com?foo=bar&foo=fuzz&foo=barfuzz", "foo", "", "foo="},
		{"9", "http://example.com", "foo", "", "foo="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			s.Set(tt.key, tt.value)
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("String() = %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}

func TestUrlSearchParams_String(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantSerialized string
	}{
		{"1", "http://example.com?foo=bar", "foo=bar"},
		{"2", "http://example.com?foo=bar&foo 2=bar+2", "foo=bar&foo+2=bar+2"},
		{"3", "http://example.com?foo2=bar2&foo=bar", "foo2=bar2&foo=bar"},
		{"4", "http://example.com?foo=bar&foo=bar2", "foo=bar&foo=bar2"},
		{"5", "http://example.com?xyz=aaa&foo=bar2&xyz=aaa&foo=bar", "xyz=aaa&foo=bar2&xyz=aaa&foo=bar"},
		{"6", "http://example.com?foo=bar&foo=fuzz&foo=barfuzz", "foo=bar&foo=fuzz&foo=barfuzz"},
		{"7", "http://example.com?foo", "foo="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := Parse(tt.url)
			s := url.SearchParams()
			if got := s.String(); got != tt.wantSerialized {
				t.Errorf("got %v, want %v", got, tt.wantSerialized)
			}
		})
	}
}
