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

package internal

import (
	gerrors "errors"

	"github.com/nationallibraryofnorway/whatwg-url/errors"
)

// handleError handles an error according to the options set for the parser
func (p *Parser) handleError(u *Url, err error) error {
	var verr *errors.ValidationError
	if !gerrors.As(err, &verr) {
		return err
	}
	if p.ReportValidationErrors {
		u.validationErrors = append(u.validationErrors, verr)
	}
	if verr.Failure() || p.FailOnValidationError {
		return verr
	}
	return nil
}

// handleErrorWithDescription handles an error according to the options set for the parser
func (p *Parser) handleErrorWithDescription(u *Url, err *errors.ValidationError, description string) error {
	return p.handleWrappedError(u, err, gerrors.New(description))
}

// handleWrappedError handles an error according to the options set for the parser
func (p *Parser) handleWrappedError(u *Url, err *errors.ValidationError, cause error) error {
	err = err.Wrap(cause)
	return p.handleError(u, err)
}
