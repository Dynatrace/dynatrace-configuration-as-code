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

package metadata

import (
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
)

const ClassicEnvironmentDomainPath = "/platform/metadata/v1/classic-environment-domain"
const DeprecatedClassicEnvDomainPath = "/platform/core/v1/environment-api-info"

type classicEnvURL struct {
	// Domain is the URL of the classic environment
	Domain string `json:"domain"`
	// Endpoint is the URL of the classic environment
	// Note: newer environments return the classic environment URL in the Domain field
	Endpoint string `json:"endpoint"`
}

func (u classicEnvURL) GetURL() string {
	if u.Domain == "" {
		return u.Endpoint
	}
	return u.Domain
}

// GetDynatraceClassicURL tries to fetch the URL of the classic environment using the API of a platform enabled
// environment
func GetDynatraceClassicURL(ctx context.Context, client *http.Client, environmentURL string) (string, error) {
	endpointURL, err := url.JoinPath(environmentURL, ClassicEnvironmentDomainPath)
	if err != nil {
		return "", fmt.Errorf("failed to build URL for API %q on environment URL %q", ClassicEnvironmentDomainPath, environmentURL)
	}

	resp, err := rest.Get(ctx, client, endpointURL)
	if !resp.IsSuccess() || err != nil {
		log.Debug("failed to query classic environment url from %q, falling back to deprecated endpoint %q: %v (HTTP %v)", ClassicEnvironmentDomainPath, DeprecatedClassicEnvDomainPath, err, resp.StatusCode)

		deprecatedEndpointURL, err := url.JoinPath(environmentURL, DeprecatedClassicEnvDomainPath)
		if err != nil {
			return "", fmt.Errorf("failed to build URL for API %q on environment URL %q", DeprecatedClassicEnvDomainPath, environmentURL)
		}
		resp, err = rest.Get(ctx, client, deprecatedEndpointURL)
		if err != nil {
			return "", fmt.Errorf("failed to query classic environment url after fallback to %q: %w", DeprecatedClassicEnvDomainPath, err)
		}
	}

	if !resp.IsSuccess() {
		return "", rest.NewRespErr(
			fmt.Sprintf("failed to query classic environment URL: (HTTP %v) %v", resp.StatusCode, string(resp.Body)),
			resp)
	}

	var jsonResp classicEnvURL
	if err := json.Unmarshal(resp.Body, &jsonResp); err != nil {
		// for specific Dynatrace base URLs (e.g. https://env-id.live.dynatrace.com) we get back an HTTP 200 OK,
		// however the payload is not the expected JSON but HTML content.
		// At this point, best we can do is give the user a hint that the URL is not completely correct
		return "", fmt.Errorf("failed to parse classic environment response payload from %q. Please check your dynatrace environment URL to match the following pattern: https://<env-id>.apps.dynatrace.com", endpointURL)
	}
	return jsonResp.GetURL(), nil
}
