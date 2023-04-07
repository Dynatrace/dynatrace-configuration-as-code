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

package client

import (
	version2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"net/http"
	"runtime"
)

// TokenAuthTransport should be used to enable a client
// to use dynatrace token authorization
type TokenAuthTransport struct {
	http.RoundTripper
	header http.Header
}

// NewTokenAuthTransport creates a new http transport to be used for
// token authorization
func NewTokenAuthTransport(baseTransport http.RoundTripper, token string) *TokenAuthTransport {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	t := &TokenAuthTransport{
		RoundTripper: baseTransport,
		header:       http.Header{},
	}
	t.setHeader("Authorization", "Api-Token "+token)
	t.setHeader("User-Agent", "Dynatrace Monitoring as Code/"+version2.MonitoringAsCode+" "+(runtime.GOOS+" "+runtime.GOARCH))
	return t
}

func (t *TokenAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the custom headers to the request
	for k, v := range t.header {
		req.Header[k] = v
	}
	return t.RoundTripper.RoundTrip(req)
}

func (t *TokenAuthTransport) setHeader(key, value string) {
	t.header.Set(key, value)
}
