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
	"maps"

	"github.com/nationallibraryofnorway/whatwg-url/internal"
)

// WithReportValidationErrors is kept for backward compatibility.
// Validation errors are recorded on returned URL values by default.
//
// This API is EXPERIMENTAL.
func WithReportValidationErrors() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.ReportValidationErrors = true
	})
}

// WithFailOnValidationError makes the parser return an error for all validation errors.
//
// This API is EXPERIMENTAL.
func WithFailOnValidationError() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.FailOnValidationError = true
	})
}

// WithLaxHostParsing ignores some decoding errors and returns the host as is.
//
// This API is EXPERIMENTAL.
func WithLaxHostParsing() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.LaxHostParsing = true
	})
}

// WithCollapseConsecutiveSlashes collapses consecutive slashes in path into one
// (e.g. http://example.com//foo///bar => http://example.com/foo/bar).
//
// This API is EXPERIMENTAL.
func WithCollapseConsecutiveSlashes() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.CollapseConsecutiveSlashes = true
	})
}

// WithAcceptInvalidCodepoints percent encodes values which are not valid UTF-8.
//
// This API is EXPERIMENTAL.
func WithAcceptInvalidCodepoints() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.AcceptInvalidCodepoints = true
	})
}

// WithPreParseHostFunc is a function which allows manipulation of host string before it is parsed.
//
// This API is EXPERIMENTAL.
func WithPreParseHostFunc(f func(url *Url, host string) string) internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.PreParseHostFunc = f
	})
}

// WithPostParseHostFunc is a function which allows manipulation of host string after it is parsed.
// It is called only if the host isn't an IP address.
//
// This API is EXPERIMENTAL.
func WithPostParseHostFunc(f func(url *Url, host string) string) internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.PostParseHostFunc = f
	})
}

// WithPercentEncodeSinglePercentSign percent encodes a '%' which is not followed by two hexadecimal digits
// instead of complaining about invalid percent encoding.
//
// This API is EXPERIMENTAL.
func WithPercentEncodeSinglePercentSign() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.PercentEncodeSinglePercentSign = true
	})
}

// WithAllowSettingPathForNonBaseUrl allows to set path for a url which cannot be a base url.
// WhathWg standard says this should be illegal
//
// This API is EXPERIMENTAL.
func WithAllowSettingPathForNonBaseUrl() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.AllowSettingPathForNonBaseUrl = true
	})
}

// WithSkipWindowsDriveLetterNormalization skips conversion of 'C|' to 'C:'.
// WhathWg standard says only a normalized Windows drive letter is conforming.
//
// This API is EXPERIMENTAL.
func WithSkipWindowsDriveLetterNormalization() internal.ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.SkipWindowsDriveLetterNormalization = true
	})
}

// WithSpecialSchemes allows overriding the notion of special schemes.
// special is a map of 'scheme' => 'default port'
//
// WhatWg standard removed gopher from special schemes. This is how you add it back:
//
//	special := map[string]string{
//	                       "ftp":    21,
//	                       "file":   0,
//	                       "http":   80,
//	                       "https":  443,
//	                       "ws":     80,
//	                       "wss":    443,
//	                       "gopher": 70,
//	               }
//
// This API is EXPERIMENTAL.
func WithSpecialSchemes(special map[string]uint16) ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		if o.SpecialSchemes == nil {
			o.SpecialSchemes = make(map[string]uint16)
		}
		maps.Copy(o.SpecialSchemes, special)
		o.HasCustomSpecialSchemes = true
	})
}

// WithPathPercentEncodeSet allows to set an alternative set of characters to percent encode in path component.
//
// This API is EXPERIMENTAL.
func WithPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.PathPercentEncodeSet = encodeSet
	})
}

// WithQueryPercentEncodeSet allows to set an alternative set of characters to percent encode in query component
// when scheme is not special.
//
// This API is EXPERIMENTAL.
func WithQueryPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.QueryPercentEncodeSet = encodeSet
	})
}

// WithSpecialQueryPercentEncodeSet allows to set an alternative set of characters to percent encode in query component
// when scheme is special.
//
// This API is EXPERIMENTAL.
func WithSpecialQueryPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.SpecialQueryPercentEncodeSet = encodeSet
	})
}

// WithFragmentPathPercentEncodeSet allows to set an alternative set of characters to percent encode in fragment
// component when scheme is not special.
//
// This API is EXPERIMENTAL.
func WithFragmentPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.FragmentPercentEncodeSet = encodeSet
	})
}

// WithSpecialFragmentPathPercentEncodeSet allows to set an alternative set of characters to percent encode in fragment
// component when scheme is special.
//
// This API is EXPERIMENTAL.
func WithSpecialFragmentPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.SpecialFragmentPercentEncodeSet = encodeSet
	})
}

// WithSkipTrailingSlashNormalization skips normalizing of empty paths.
//
// This API is EXPERIMENTAL.
func WithSkipTrailingSlashNormalization() ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.SkipTrailingSlashNormalization = true
	})
}

// WithSkipEqualsForEmptySearchParamsValue skips writing '=' when setting an empty value for a search parameter.
//
// e.g. url.SearchParams().Set("name", "") gives 'http://...?name' instead of 'http://...?name='
//
// This API is EXPERIMENTAL.
func WithSkipEqualsForEmptySearchParamsValue() ParserOption {
	return internal.NewParserOption(func(o *internal.Parser) {
		o.SkipEqualsForEmptySearchParamsValue = true
	})
}
