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

func validationError(kind kind, msg string) *ValidationError {
	return &ValidationError{kind: kind, msg: msg}
}

// ValidationError indicates that the url is not valid
type ValidationError struct {
	cause error
	kind  kind
	msg   string
}

func (e *ValidationError) Error() string {
	msg := e.msg
	if e.cause != nil {
		return msg + ": " + e.cause.Error()
	}
	return msg
}

func (e *ValidationError) Wrap(err error) *ValidationError {
	return &ValidationError{
		cause: err,
		kind:  e.kind,
		msg:   e.msg,
	}
}

func (e *ValidationError) Is(target error) bool {
	t, ok := target.(*ValidationError)
	if !ok {
		return false
	}
	return e.kind == t.kind
}

func (e *ValidationError) Unwrap() error {
	return e.cause
}
