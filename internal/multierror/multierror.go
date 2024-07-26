/*
 * @license
 * Copyright 2023 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package multierror

import (
	"errors"
	"fmt"
	"strings"
)

// MultiError is an error containing several errors
// Unlike errors.Join() it produces an error with exported fields and can be included in structured logging
type MultiError struct {
	// Errors is a list of errors grouped into this MultiError
	Errors []error `json:"errors"`
}

func (m MultiError) Error() string {
	s := make([]string, len(m.Errors))
	for i, e := range m.Errors {
		s[i] = e.Error()
	}
	return fmt.Sprintf("encountered multiple errors: [ %s ]", strings.Join(s, ", "))
}

func New(errs ...error) error {
	if len(errs) == 1 {
		// callers might not always check this beforehand, but building a MultiError for a single error is useless
		return errs[0]
	}

	m := MultiError{}
	for _, e := range errs {
		if e != nil {
			var me MultiError
			if errors.As(e, &me) {
				m.Errors = append(m.Errors, me.Errors...)
			} else {
				m.Errors = append(m.Errors, e)
			}
		}
	}

	return m
}
