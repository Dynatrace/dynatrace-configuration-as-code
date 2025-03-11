/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
)

type customErr struct {
	StatusCode int
}

func (e customErr) Error() string {
	return "error"
}
func TestIsAPIErrorStatusNotFound(t *testing.T) {
	t.Run("is true if status is 404", func(t *testing.T) {
		err := api.APIError{StatusCode: http.StatusNotFound}
		got := IsAPIErrorStatusNotFound(err)
		assert.Equal(t, true, got)
	})

	t.Run("is false if status is not 404", func(t *testing.T) {
		testcases := []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusOK, http.StatusForbidden, http.StatusUnauthorized}

		for _, tt := range testcases {
			t.Run(fmt.Sprintf("is false if status is %d", tt), func(t *testing.T) {
				err := api.APIError{StatusCode: tt}
				got := IsAPIErrorStatusNotFound(err)
				assert.Equal(t, false, got)
			})
		}
	})

	t.Run("is false if error is not an apiError", func(t *testing.T) {
		testcases := []error{customErr{StatusCode: http.StatusNotFound}, errors.New("error")}
		for _, err := range testcases {
			t.Run(fmt.Sprintf("is false if status is %T", err), func(t *testing.T) {
				got := IsAPIErrorStatusNotFound(err)
				assert.Equal(t, false, got)
			})
		}
	})
}
