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

package useragent

import "net/http"

// CustomUserAgentTransport is a http.RoundTripper that ensures that a custom user-agent string is set for each request
type CustomUserAgentTransport struct {
	http.RoundTripper
	userAgentString string
}

// NewCustomUserAgentTransport creates a new http transport that sets a custom user-agent string for each request
func NewCustomUserAgentTransport(baseTransport http.RoundTripper, userAgent string) *CustomUserAgentTransport {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	t := &CustomUserAgentTransport{
		RoundTripper:    baseTransport,
		userAgentString: userAgent,
	}
	return t
}

func (t *CustomUserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgentString)
	return t.RoundTripper.RoundTrip(req)
}
