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

func newValidationError(failure bool, msg string) *ValidationError {
	return &ValidationError{
		msg:     msg,
		failure: failure,
	}
}

// ValidationError indicates that the url is not valid
type ValidationError struct {
	cause   error
	msg     string
	failure bool
}

func (e *ValidationError) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

// Failure returns true if the error is a failure, false if it is a warning
func (e *ValidationError) Failure() bool {
	return e.failure
}

func (e *ValidationError) Wrap(err error) *ValidationError {
	return &ValidationError{
		cause:   err,
		msg:     e.msg,
		failure: e.failure,
	}
}

func (e *ValidationError) Unwrap() error {
	return e.cause
}
