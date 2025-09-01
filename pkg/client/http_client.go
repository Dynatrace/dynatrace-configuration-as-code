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

package client

import (
	"context"
	"crypto/tls"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
)

// SetCustomHTTPClientInContext sets a custom HTTP client for the oauth2 lib without SSL certificate checks, if the SkipCertificateVerification FF is set
func SetCustomHTTPClientInContext(ctx context.Context) context.Context {
	if !featureflags.SkipCertificateVerification.Enabled() {
		return ctx
	}

	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: &http.Transport{
			// nosemgrep
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
			},
			Proxy: http.ProxyFromEnvironment,
		},
	})
}
