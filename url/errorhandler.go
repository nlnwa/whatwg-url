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
	"errors"
	"fmt"
)

func validationDetailForCodePoint(template *ValidationError, r rune) string {
	if template == nil {
		return fmt.Sprintf("U+%04X (%q)", r, r)
	}
	return fmt.Sprintf("%s: U+%04X (%q)", template.Detail(), r, r)
}

func validationDetailForInvalidUTF8Byte(template *ValidationError, b byte) string {
	if template == nil {
		return fmt.Sprintf("invalid UTF-8 byte 0x%02X", b)
	}
	return fmt.Sprintf("%s: invalid UTF-8 byte 0x%02X", template.Detail(), b)
}

func validationDetail(template *ValidationError, context string) string {
	if template == nil {
		return context
	}
	if context == "" {
		return template.Detail()
	}
	return template.Detail() + ": " + context
}

// parseContext owns all validation events detected by one invocation of the
// basic URL parser. It deliberately keeps diagnostics separate from parser
// control flow and from any partially constructed URL.
type parseContext struct {
	*parser
	validationErrors []*ValidationError
}

func newParseContext(p *parser) *parseContext {
	return &parseContext{parser: p}
}

// reportValidation records exactly one WHATWG validation-error occurrence.
// FailOnValidationError is a policy layered over that standard event; it does
// not determine which standard parser steps return failure.
func (c *parseContext) reportValidation(code ValidationErrorCode, detail string, cause error) error {
	occurrence := &ValidationError{code: code, detail: detail, cause: cause}
	if failure := c.recordValidationOccurrence(occurrence); failure != nil {
		return failure
	}
	return nil
}

// reportValidationOccurrence records an already-materialized validation
// occurrence without replacing it. Package-level sentinels are templates, not
// parse occurrences, so clone them before adding them to the event ledger.
func (c *parseContext) reportValidationOccurrence(occurrence *ValidationError) *ParseError {
	if occurrence == nil {
		return nil
	}
	if isValidationErrorSentinel(occurrence) {
		occurrence = &ValidationError{
			code:   occurrence.code,
			detail: occurrence.detail,
			cause:  occurrence.cause,
		}
	}
	return c.recordValidationOccurrence(occurrence)
}

func (c *parseContext) recordValidationOccurrence(occurrence *ValidationError) *ParseError {
	c.validationErrors = append(c.validationErrors, occurrence)
	if c.FailOnValidationError {
		return c.returnFailure(occurrence.Error())
	}
	return nil
}

// returnFailure represents a specification "return failure" step without
// inventing a validation category. Previously reported events are included.
func (c *parseContext) returnFailure(detail string) *ParseError {
	return newParseError(detail, c.validationErrors)
}

func (c *parseContext) attachValidationErrors(u *Url) {
	if u == nil || !c.ReportValidationErrors || len(c.validationErrors) == 0 {
		return
	}
	u.validationErrors = append(u.validationErrors, c.validationErrors...)
}

// normalizeFailure is a defensive boundary for helper errors. Parser states
// use reportValidation and returnFailure directly; this also ensures no raw
// helper error can escape BasicParse.
func (c *parseContext) normalizeFailure(err error) *ParseError {
	if err == nil {
		return nil
	}
	if parseErr, ok := errors.AsType[*ParseError](err); ok {
		return parseErr
	}
	if validationErr, ok := errors.AsType[*ValidationError](err); ok {
		if failure := c.reportValidationOccurrence(validationErr); failure != nil {
			return failure
		}
	}
	return c.returnFailure(err.Error())
}
