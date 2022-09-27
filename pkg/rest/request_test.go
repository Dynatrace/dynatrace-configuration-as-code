//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package rest

import (
	"gotest.tools/assert"
	"net/http"
	"testing"
)

func TestGetWithStatus429AndWithoutRetryHeaders(t *testing.T) {
	client, url := newDynatraceTestServer(t, func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusTooManyRequests)
	})

	_, err := get(client, url, "token")

	assert.ErrorContains(t, err, "X-RateLimit-Limit")
}

func TestGetWithStatus429AndWithoutXRateLimitReset(t *testing.T) {
	client, url := newDynatraceTestServer(t, func(res http.ResponseWriter, req *http.Request) {
		// setting directly to not use the canonical name - it will be used anyway somewhere in Go
		res.Header()["X-RateLimit-Limit"] = []string{"some-limit"}

		http.Error(res, "", http.StatusTooManyRequests)
	})

	_, err := get(client, url, "token")

	assert.ErrorContains(t, err, "X-RateLimit-Reset")
}
