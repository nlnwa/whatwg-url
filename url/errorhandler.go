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
	"github.com/nlnwa/whatwg-url/errors"
)

func (p *parser) handleError(u *Url, code errors.ErrorCode) error {
	if p.opts.reportValidationErrors {
		u.validationErrors = append(u.validationErrors, errors.Error(code, u.inputUrl))
	}
	if p.opts.failOnValidationError {
		return errors.Error(code, u.inputUrl)
	}
	return nil
}

func (p *parser) handleFailure(u *Url, code errors.ErrorCode, err error) (*Url, error) {
	return nil, errors.Wrap(err, code, u.inputUrl)
}
