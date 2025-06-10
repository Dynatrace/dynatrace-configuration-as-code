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

package auth

import (
	"net/http"
)

// ApiTokenAuthTransport should be used to enable a client to use Dynatrace API token authorization
type ApiTokenAuthTransport struct {
	http.RoundTripper
	header http.Header
}

// NewTokenAuthTransport creates a new http transport to be used for
// token authorization
func NewTokenAuthTransport(baseTransport http.RoundTripper, apiToken string) *ApiTokenAuthTransport {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	t := &ApiTokenAuthTransport{
		RoundTripper: baseTransport,
		header:       http.Header{},
	}
	t.setHeader("Authorization", "Api-Token "+apiToken)
	return t
}

func (t *ApiTokenAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the custom headers to the request
	for k, v := range t.header {
		req.Header[k] = v
	}
	return t.RoundTripper.RoundTrip(req)
}

func (t *ApiTokenAuthTransport) setHeader(key, value string) {
	t.header.Set(key, value)
}
