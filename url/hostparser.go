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
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// domainParserToASCIIProfile configures Unicode ToASCII as required by the
// domain parser ToASCII algorithm.
func domainParserToASCIIProfile(beStrict bool) *idna.Profile {
	return idna.New(
		idna.MapForLookup(),
		idna.VerifyDNSLength(beStrict),
		idna.StrictDomainName(beStrict),
		idna.CheckHyphens(beStrict),
		idna.CheckJoiners(true),
		idna.BidiRule(),
		idna.Transitional(false),
	)
}

var (
	idnaStrict    = domainParserToASCIIProfile(true)
	idnaNonStrict = domainParserToASCIIProfile(false)
)

// domainParserToASCII implements the domain parser ToASCII algorithm.
// https://url.spec.whatwg.org/#domain-parser-to-ascii
func domainParserToASCII(domain string, beStrict bool) (string, error) {
	if !beStrict {
		return idnaNonStrict.ToASCII(domain)
	}

	// Implementation note: preserve a trailing root label while working around
	// x/net/idna treating it as an empty label during DNS-length verification.
	// UTS #46 excludes that trailing dot from DNS length and label checks.
	trimmed := domain
	hasRootLabel := false
	for _, separator := range []string{".", "\u3002", "\uff0e", "\uff61"} {
		if strings.HasSuffix(trimmed, separator) && len(trimmed) > len(separator) {
			trimmed = strings.TrimSuffix(trimmed, separator)
			hasRootLabel = true
			break
		}
	}

	result, err := idnaStrict.ToASCII(trimmed)
	if hasRootLabel {
		result += "."
	}
	return result, err
}

// domainParser implements the domain parser algorithm. warning represents its
// non-fatal domain-to-ASCII validation event; failure represents a step that
// returns failure.
// https://url.spec.whatwg.org/#concept-domain-parser
func domainParser(domain string, beStrict bool) (result string, warning, failure error) {
	if domain == "" {
		return "", nil, nil
	}

	// Let strictResult be the result of running domain parser ToASCII with
	// domain and true.
	strictResult, strictErr := domainParserToASCII(domain, true)

	// If strictResult is failure, domain-to-ASCII validation error. This step
	// does not return.
	if strictErr != nil {
		warning = validationErrorWithCause(ErrDomainToASCII, strictErr)
	}

	// If beStrict is true, return failure when strictResult is failure;
	// otherwise return strictResult.
	if beStrict {
		if strictErr != nil {
			return strictResult, warning, warning
		}
		return strictResult, warning, nil
	}

	// Let result be null.
	ascii := isASCII(domain)
	haveResult := false

	// If domain is an ASCII string, then set result to domain, lowercased.
	if ascii {
		result = strings.ToLower(domain)
	} else if strictErr == nil {
		// Implementation note: strict and non-strict ToASCII have identical
		// output when strict processing succeeds, so reuse strictResult.
		result = strictResult
		haveResult = true
	}

	// Otherwise, set result to domain parser ToASCII with domain and false. If
	// result is failure, then return failure.
	if !ascii && !haveResult {
		var err error
		result, err = domainParserToASCII(domain, false)
		if err != nil {
			return result, warning, err
		}
	}

	// If result is the empty string, then return failure.
	if result == "" {
		return "", warning, errors.New("Unicode ToASCII returned the empty string")
	}

	// If result contains a forbidden domain code point, then return failure.
	for _, r := range result {
		if r < utf8.RuneSelf && forbiddenDomainASCII[r] {
			return result, warning, errors.New("the host contains a forbidden domain code point")
		}
	}

	return result, warning, nil
}

// parseHost is the package-internal entry point used by focused host-parser
// tests. Basic URL parsing calls parseContext.parseHost so nested validation
// events share the parse's event ledger.
func (p *parser) parseHost(u *Url, input []byte, isOpaque bool) (string, error) {
	ctx := newParseContext(p)
	host, err := ctx.parseHost(u, input, isOpaque)
	ctx.attachValidationErrors(u)
	if err != nil {
		return host, ctx.normalizeFailure(err)
	}
	return host, nil
}

// parseHost implements the host parser.
// https://url.spec.whatwg.org/#concept-host-parser
func (c *parseContext) parseHost(u *Url, input []byte, isOpaque bool) (string, error) {
	// If input starts with U+005B ([), then:
	if len(input) > 0 && input[0] == '[' {
		// If input does not end with U+005D (]),
		if input[len(input)-1] != ']' {
			// IPv6-unclosed validation error, return failure.
			if err := c.reportValidation(CodeIPv6Unclosed, ErrIPv6Unclosed.Detail(), nil); err != nil {
				return "", err
			}
			return "", c.returnFailure(ErrIPv6Unclosed.Detail())
		}
		// Return the result of IPv6 parsing input with its leading U+005B ([)
		// and trailing U+005D (]) removed.
		addr, err := parseIPv6(input[1:len(input)-1], c.reportValidation)
		if err != nil {
			if parseErr, ok := errors.AsType[*ParseError](err); ok {
				return "", parseErr
			}
			validationErr, ok := errors.AsType[*ValidationError](err)
			if !ok {
				return "", c.returnFailure(err.Error())
			}
			if err := c.reportValidationOccurrence(validationErr); err != nil {
				return "", err
			}
			return "", c.returnFailure(validationErr.Detail())
		}
		u.isIPv6 = true
		return addr.StringWithBrackets(), nil
	}

	// If isOpaque is true, then return the result of opaque-host parsing input.
	if isOpaque {
		return c.parseOpaqueHost(input)
	}

	// Assert: input is not the empty string.
	if len(input) == 0 {
		return "", nil
	}

	// If input contains a percent-encoded byte, domain-percent-encoded
	// validation error.
	buf := input
	bufOwned := false
	if containsPercentEncodedByte(buf) {
		if err := c.reportValidation(CodeDomainPercentEncoded, ErrDomainPercentEncoded.Detail(), nil); err != nil {
			return "", err
		}

		// Implementation note: percent-decoding mutates its buffer. Copy only
		// on this path so borrowed parser input is never modified.
		buf = append([]byte(nil), buf...)
		bufOwned = true
		buf, _ = percentDecodeInPlace(buf)
	}

	// Let domain be the result of running UTF-8 decode without BOM on the
	// percent-decoding of input.
	if len(buf) >= 3 &&
		buf[0] == 0xEF &&
		buf[1] == 0xBB &&
		buf[2] == 0xBF {
		buf = buf[3:]
	}

	// Implementation note: ASCII domains without an ACE prefix can use the
	// allocation-conscious path below while preserving domain-parser behavior.
	if isASCIIWithoutACEPrefix(buf) {
		return c.parseASCIIHostWithoutACEPrefix(u, buf, bufOwned)
	}

	// Implementation note: x/net/idna treats malformed UTF-8 as U+FFFD, so it
	// reaches the domain-parser failure required by the standard without a
	// separate utf8.Valid scan.
	domain := string(buf)

	// Let asciiDomain be the result of running the domain parser with domain and
	// false.
	asciiDomain, warning, failure := domainParser(domain, false)
	if warning != nil {
		if validationErr, ok := errors.AsType[*ValidationError](warning); ok {
			if err := c.reportValidationOccurrence(validationErr); err != nil {
				return "", err
			}
		}
	}
	// If asciiDomain is failure, then return failure.
	if failure != nil {
		// Implementation note: lax host recovery returns the original input only
		// after all validation events detected by the domain parser. Fail-fast
		// therefore takes precedence.
		if c.LaxHostParsing {
			return string(input), nil
		}
		return "", c.returnFailure(failure.Error())
	}
	// If asciiDomain ends in a number,
	if endsInANumber(asciiDomain) {
		// If domain is not an ASCII string, IPv4-non-ASCII-input validation error.
		if !isASCII(domain) {
			if err := c.reportValidation(CodeIPv4NonASCIIInput, ErrIPv4NonASCIIInput.Detail(), nil); err != nil {
				return "", err
			}
		}
		// Return the result of IPv4 parsing asciiDomain.
		return c.parseIPv4(u, asciiDomain)
	}

	// Return asciiDomain.
	return asciiDomain, nil
}

// parseASCIIHostWithoutACEPrefix implements the domain parser and remaining
// host-parser steps for an ASCII domain without an ACE-prefixed label.
//
// Implementation note: keeping this optimization separate leaves the main
// host parser aligned with the standard algorithm while avoiding IDNA work on
// the dominant clean-ASCII path.
func (c *parseContext) parseASCIIHostWithoutACEPrefix(u *Url, domain []byte, domainOwned bool) (string, error) {
	// Domain parser step 5: set result to domain, lowercased.
	needsLowercase := false
	for _, b := range domain {
		if 'A' <= b && b <= 'Z' {
			needsLowercase = true
			break
		}
	}
	if needsLowercase {
		// Implementation note: lowercase in place only when the host parser
		// already owns the percent-decoded buffer; otherwise copy on write.
		if !domainOwned {
			domain = append([]byte(nil), domain...)
		}
		for i, b := range domain {
			if 'A' <= b && b <= 'Z' {
				domain[i] |= 0x20
			}
		}
	}

	asciiDomain := string(domain)
	// Domain parser steps 1–2: obtain the strict result and report a
	// domain-to-ASCII validation error if strict processing fails.
	// Implementation note: strict validity is insensitive to ASCII case, so
	// checking the lowercased result here avoids a second pass. The local
	// predicate proves the common strict-success case without calling IDNA.
	strictValid := isStrictASCIIValid(domain)
	if !strictValid {
		// Implementation note: the local check is conservative. Keep strict IDNA
		// as the source of detailed error causes on possible-invalid input.
		if _, err := domainParserToASCII(asciiDomain, true); err != nil {
			if err := c.reportValidation(CodeDomainToASCII, ErrDomainToASCII.Detail(), err); err != nil {
				return "", err
			}
		}
	}

	// Domain parser step 7: if result is the empty string, return failure.
	if asciiDomain == "" {
		return "", c.returnFailure("Unicode ToASCII returned the empty string")
	}
	// Domain parser step 8: if result contains a forbidden domain code point,
	// return failure.
	for _, b := range domain {
		if forbiddenDomainASCII[b] {
			return "", c.returnFailure("the host contains a forbidden domain code point")
		}
	}

	// Host parser step 8: if asciiDomain ends in a number, return the result of
	// IPv4 parsing asciiDomain. The non-ASCII validation substep cannot apply.
	if endsInANumber(domain) {
		return c.parseIPv4(u, asciiDomain)
	}
	// Host parser step 9: return asciiDomain.
	return asciiDomain, nil
}

// parseOpaqueHost implements the opaque-host parser.
// https://url.spec.whatwg.org/#concept-opaque-host-parser
func (c *parseContext) parseOpaqueHost(input []byte) (string, error) {
	// Implementation note: this traversal fuses the opaque-host parser's four
	// whole-input phases. For each code point it checks forbidden host code
	// points, URL code points, and malformed percent-encoding in standard step
	// order, then performs C0 percent-encoding before advancing. This preserves
	// the established input-order validation events, fail-fast behavior, and
	// clean-path performance; see the opaque-host phase-order tests.

	// Implementation note: preallocate for the worst case, where every input
	// byte is serialized as %HH.
	out := make([]byte, 0, len(input)*3)

	for i := 0; i < len(input); {
		b := input[i]

		// Implementation note: handle ASCII without rune decoding.
		if b < utf8.RuneSelf {
			// If input contains a forbidden host code point,
			if forbiddenHostASCII[b] {
				// host-invalid-code-point validation error, return failure.
				if err := c.reportValidation(CodeHostInvalidCodePoint, validationDetailForCodePoint(ErrHostInvalidCodePoint, rune(b)), nil); err != nil {
					return "", err
				}
				// Implementation note: lax host recovery is applied only after the
				// standard validation event; fail-fast therefore takes precedence.
				if c.LaxHostParsing {
					return string(input), nil
				}
				return "", c.returnFailure(ErrHostInvalidCodePoint.Detail())
			}

			// If input contains a code point that is not a URL code point and not
			// U+0025 (%), invalid-URL-unit validation error.
			if !urlCodePointASCII[b] && b != '%' {
				if err := c.reportValidation(CodeInvalidURLUnit, validationDetailForCodePoint(ErrInvalidURLUnit, rune(b)), nil); err != nil {
					return "", err
				}
			}

			// If input contains a U+0025 (%) and the two code points following it
			// are not ASCII hex digits, invalid-URL-unit validation error.
			if b == '%' && (i+2 >= len(input) || !isHex(input[i+1]) || !isHex(input[i+2])) {
				if err := c.reportValidation(CodeInvalidURLUnit, validationDetailForCodePoint(ErrInvalidURLUnit, '%'), nil); err != nil {
					return "", err
				}
				// Implementation note: single-percent recovery occurs only after
				// reporting, so fail-fast takes precedence.
				if c.PercentEncodeSinglePercentSign {
					out = append(out, percentPercentEncoded...)
					i++
					continue
				}
			}

			// Return the result of running UTF-8 percent-encode on input using the
			// C0 control percent-encode set.
			if C0PercentEncodeSet.ByteShouldBeEncoded(b) {
				out = append(out, '%', upperHex[b>>4], upperHex[b&15])
			} else {
				out = append(out, b)
			}
			i++
			continue
		}

		r, size := utf8.DecodeRune(input[i:])
		if r == utf8.RuneError && size == 1 {
			// Implementation note: raw invalid UTF-8 is outside the scalar-value
			// input domain. Report it as an invalid URL unit, then either encode
			// U+FFFD or preserve the byte under the recovery option.
			if err := c.reportValidation(CodeInvalidURLUnit, validationDetailForInvalidUTF8Byte(ErrInvalidURLUnit, b), nil); err != nil {
				return "", err
			}
			if c.AcceptInvalidCodepoints {
				out = append(out, '%', upperHex[b>>4], upperHex[b&15])
			} else {
				out = appendPercentEncoded(out, utf8.RuneError, C0PercentEncodeSet)
			}
			i++
			continue
		}

		// If input contains a code point that is not a URL code point and not
		// U+0025 (%), invalid-URL-unit validation error.
		if !isURLCodePoint(r) && r != '%' {
			if err := c.reportValidation(CodeInvalidURLUnit, validationDetailForCodePoint(ErrInvalidURLUnit, r), nil); err != nil {
				return "", err
			}
		}

		// Return the result of running UTF-8 percent-encode on input using the
		// C0 control percent-encode set.
		out = appendPercentEncoded(out, r, C0PercentEncodeSet)
		i += size
	}

	return string(out), nil
}

func containsPercentEncodedByte(input []byte) bool {
	for i := 0; i+2 < len(input); i++ {
		if input[i] == '%' && isHex(input[i+1]) && isHex(input[i+2]) {
			return true
		}
	}
	return false
}

// isASCIIWithoutACEPrefix reports whether input is ASCII and none of its
// labels starts with the case-insensitive ACE prefix "xn--".
func isASCIIWithoutACEPrefix(input []byte) bool {
	p := 0
	for i := range input {
		b := input[i]
		if b >= utf8.RuneSelf {
			return false
		}
		switch {
		case b == '.':
			p = 0
		case p == 0 && (b == 'x' || b == 'X'):
			p = 1
		case p == 1 && (b == 'n' || b == 'N'):
			p = 2
		case p == 2 && b == '-':
			p = 3
		case p == 3 && b == '-':
			return false
		default:
			p = -1
		}
	}
	return true
}

// isStrictASCIIValid reports whether a lowercased ASCII domain with no
// Punycode labels satisfies the strict UTS #46 checks. Under these
// preconditions, Bidi, Joiner, normalization, and Punycode validation cannot
// fail, leaving only STD3, hyphen, empty-label, and DNS-length checks.
func isStrictASCIIValid(domain []byte) bool {
	// Empty input cannot produce the non-empty domain required by the domain
	// parser.
	if len(domain) == 0 {
		return false
	}

	// A final U+002E represents the root label. UTS #46 excludes that label
	// from empty-label and DNS-length validation.
	if domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}
	if len(domain) == 0 {
		// After removing the root label, at least one non-empty label must remain.
		return false
	}
	if len(domain) > 253 {
		// VerifyDnsLength limits the textual domain, excluding a trailing root
		// label, to 253 ASCII bytes.
		return false
	}

	labelStart := 0
	for i := 0; i <= len(domain); i++ {
		if i < len(domain) && domain[i] != '.' {
			b := domain[i]
			// UseSTD3ASCIIRules permits only ASCII letters, digits, and U+002D
			// HYPHEN-MINUS in these non-ACE labels.
			if ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || b == '-' {
				continue
			}
			return false
		}

		labelLen := i - labelStart
		if labelLen == 0 {
			// Strict splitting produced an empty label, as in ".example" or
			// "example..com".
			return false
		}
		if labelLen > 63 {
			// VerifyDnsLength limits each label to 63 ASCII bytes.
			return false
		}
		if domain[labelStart] == '-' || domain[i-1] == '-' {
			// CheckHyphens forbids a leading or trailing U+002D HYPHEN-MINUS.
			return false
		}
		if labelLen >= 4 && domain[labelStart+2] == '-' && domain[labelStart+3] == '-' {
			// CheckHyphens also forbids hyphens in both the third and fourth
			// positions. ACE-prefixed labels have already been excluded by the
			// caller and are handled by the full IDNA path.
			return false
		}
		labelStart = i + 1
	}

	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
