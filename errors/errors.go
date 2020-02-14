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
	gerrs "github.com/pkg/errors"
)

// UrlError is the struct of url error
type UrlError struct {
	code  ErrorCode
	url   string
	cause error // the root cause for this error
}

func (e *UrlError) Error() string {
	errMsg := fmt.Sprintf("Error: %s, Url: %s", e.code, e.url)
	if nil == e.cause {
		return errMsg
	}

	return errMsg + ", Cause: " + e.cause.Error()
}

func (e *UrlError) Cause() error {
	return e.cause
}

func (e *UrlError) Code() ErrorCode {
	return e.code
}

func (e *UrlError) Url() string {
	return e.url
}

// Cause returns the cause error of this error
func Cause(err error) error {
	return gerrs.Cause(err)
}

// Code returns the error code
func Code(err error) ErrorCode {
	type coder interface {
		Code() ErrorCode
	}

	cd, ok := err.(coder)
	if !ok {
		return 0
	}
	return cd.Code()
}

// Message returns the error message
func Url(err error) string {
	type url interface {
		Url() string
	}

	m, ok := err.(url)
	if !ok {
		return ""
	}
	return m.Url()
}

// Error constructs a new error
func Error(code ErrorCode, url string) error {
	return &UrlError{
		code: code,
		url:  url,
	}
}

// Wrap waps an error with an error and a message
func Wrap(err error, code ErrorCode, url string) error {
	return &UrlError{
		code:  code,
		url:   url,
		cause: err,
	}
}
