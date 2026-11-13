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
)

type ParserOption interface {
	apply(*parser)
}

func newParserOption(f func(*parser)) parserOptionFunc {
	return parserOptionFunc{fn: f}
}

type parserOptionFunc struct {
	fn func(*parser)
}

func (opt parserOptionFunc) apply(parser *parser) {
	opt.fn(parser)
}

// WithReportValidationErrors controls whether validation errors detected during
// a successful parse are recorded on the resulting Url.
//
// This option only controls successful-URL history. It does not change WHATWG
// continuation or failure behavior, and ParseError always contains every
// validation error detected before a failed parse, even when reporting is
// disabled.
//
// Default: true
//
// This API is EXPERIMENTAL.
func WithReportValidationErrors(enable bool) ParserOption {
	return newParserOption(func(o *parser) {
		o.ReportValidationErrors = enable
	})
}

// WithFailOnValidationError makes the first detected WHATWG validation error
// immediately terminate parsing with a ParseError.
//
// This is a policy layered over standard error detection; it does not change
// the validation error's code or imply that the WHATWG algorithm would normally
// fail at that point. This option takes precedence over WithLaxHostParsing,
// WithAcceptInvalidCodepoints, and WithPercentEncodeSinglePercentSign, so no
// configured recovery is applied after the validation error is detected.
//
// This API is EXPERIMENTAL.
func WithFailOnValidationError() ParserOption {
	return newParserOption(func(o *parser) {
		o.FailOnValidationError = true
	})
}

// WithLaxHostParsing enables non-standard recovery from domain, opaque-host,
// IPv4, and IPv6 parsing failures. Domain, opaque-host, and IPv6 failures retain
// the input host. IPv4 failures retain the ASCII-normalized domain as a hostname,
// without marking it as an IPv4 address. Valid IPv4 addresses normalize normally.
//
// The parser detects and records the applicable WHATWG validation error before
// applying this recovery. WithFailOnValidationError takes precedence and
// prevents recovery. Recovered invalid IP hosts are not marked as IP addresses.
//
// This API is EXPERIMENTAL.
func WithLaxHostParsing() ParserOption {
	return newParserOption(func(o *parser) {
		o.LaxHostParsing = true
	})
}

// WithSkipDotSegmentNormalization preserves literal and percent-encoded dot
// segments in hierarchical paths instead of resolving them during parsing.
// It also applies to relative references and path setters. Relative references
// still replace the last base-path segment before appending their own path.
// Opaque paths and slash collapsing are unaffected.
//
// This API is EXPERIMENTAL.
func WithSkipDotSegmentNormalization() ParserOption {
	return newParserOption(func(o *parser) {
		o.SkipDotSegmentNormalization = true
	})
}

// WithCollapseConsecutiveSlashes collapses consecutive slashes in path into one
// (e.g. http://example.com//foo///bar => http://example.com/foo/bar).
//
// This API is EXPERIMENTAL.
func WithCollapseConsecutiveSlashes() ParserOption {
	return newParserOption(func(o *parser) {
		o.CollapseConsecutiveSlashes = true
	})
}

// WithAcceptInvalidCodepoints enables non-standard recovery that percent-encodes
// input bytes which are not valid UTF-8.
//
// The parser detects and records the applicable WHATWG validation error before
// applying this recovery. WithFailOnValidationError takes precedence and
// prevents recovery.
//
// This API is EXPERIMENTAL.
func WithAcceptInvalidCodepoints() ParserOption {
	return newParserOption(func(o *parser) {
		o.AcceptInvalidCodepoints = true
	})
}

// WithPercentEncodeSinglePercentSign enables non-standard recovery that
// percent-encodes a '%' which is not followed by two hexadecimal digits.
//
// The parser first records the invalid-URL-unit validation error required by
// the WHATWG algorithm. WithFailOnValidationError takes precedence and prevents
// recovery.
//
// This API is EXPERIMENTAL.
func WithPercentEncodeSinglePercentSign() ParserOption {
	return newParserOption(func(o *parser) {
		o.PercentEncodeSinglePercentSign = true
	})
}

// WithAllowSettingPathForNonBaseUrl allows SetPathname to change an opaque path.
// The WHATWG URL Standard leaves opaque paths unchanged by this setter.
//
// This API is EXPERIMENTAL.
func WithAllowSettingPathForNonBaseUrl() ParserOption {
	return newParserOption(func(o *parser) {
		o.AllowSettingPathForNonBaseUrl = true
	})
}

// WithSkipWindowsDriveLetterNormalization skips conversion of 'C|' to 'C:'.
// The WHATWG URL Standard requires normalized Windows drive letters.
//
// This API is EXPERIMENTAL.
func WithSkipWindowsDriveLetterNormalization() ParserOption {
	return newParserOption(func(o *parser) {
		o.SkipWindowsDriveLetterNormalization = true
	})
}

// WithSpecialSchemes adds or overrides special schemes and their default ports.
// Schemes not present in the map retain their existing defaults. Scheme names
// must be lowercase, without a trailing colon.
//
// To treat gopher as a special scheme:
//
//	p := url.NewParser(url.WithSpecialSchemes(map[string]uint16{"gopher": 70}))
//
// This API is EXPERIMENTAL.
func WithSpecialSchemes(special map[string]uint16) ParserOption {
	return newParserOption(func(o *parser) {
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
	return newParserOption(func(o *parser) {
		o.PathPercentEncodeSet = encodeSet
	})
}

// WithQueryPercentEncodeSet allows to set an alternative set of characters to percent encode in query component
// when scheme is not special.
//
// This API is EXPERIMENTAL.
func WithQueryPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newParserOption(func(o *parser) {
		o.QueryPercentEncodeSet = encodeSet
	})
}

// WithSpecialQueryPercentEncodeSet allows to set an alternative set of characters to percent encode in query component
// when scheme is special.
//
// This API is EXPERIMENTAL.
func WithSpecialQueryPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newParserOption(func(o *parser) {
		o.SpecialQueryPercentEncodeSet = encodeSet
	})
}

// WithFragmentPathPercentEncodeSet allows to set an alternative set of characters to percent encode in fragment
// component when scheme is not special.
//
// This API is EXPERIMENTAL.
func WithFragmentPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newParserOption(func(o *parser) {
		o.FragmentPercentEncodeSet = encodeSet
	})
}

// WithSpecialFragmentPathPercentEncodeSet allows to set an alternative set of characters to percent encode in fragment
// component when scheme is special.
//
// This API is EXPERIMENTAL.
func WithSpecialFragmentPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newParserOption(func(o *parser) {
		o.SpecialFragmentPercentEncodeSet = encodeSet
	})
}

// WithSkipTrailingSlashNormalization skips normalizing of empty paths.
//
// This API is EXPERIMENTAL.
func WithSkipTrailingSlashNormalization() ParserOption {
	return newParserOption(func(o *parser) {
		o.SkipTrailingSlashNormalization = true
	})
}
