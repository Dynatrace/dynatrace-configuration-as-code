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
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"net/http"
	"net/url"
)

const classicEnvironmentDomainPath = "/platform/metadata/v1/classic-environment-domain"
const deprecatedClassicEnvDomainPath = "/platform/core/v1/environment-api-info"

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
func GetDynatraceClassicURL(client *http.Client, environmentURL string) (string, error) {
	endpointURL, err := url.JoinPath(environmentURL, classicEnvironmentDomainPath)
	if err != nil {
		return "", fmt.Errorf("failed to build URL for API %q on environment URL %q", classicEnvironmentDomainPath, environmentURL)
	}

	resp, err := rest.Get(client, endpointURL)
	if !resp.IsSuccess() || err != nil {
		log.Debug("failed to query classic environment url from %q, falling back to deprecated endpoint %q: %v (HTTP %v)", classicEnvironmentDomainPath, deprecatedClassicEnvDomainPath, err, resp.StatusCode)

		deprecatedEndpointURL, err := url.JoinPath(environmentURL, deprecatedClassicEnvDomainPath)
		if err != nil {
			return "", fmt.Errorf("failed to build URL for API %q on environment URL %q", deprecatedClassicEnvDomainPath, environmentURL)
		}
		resp, err = rest.Get(client, deprecatedEndpointURL)
		if err != nil {
			return "", fmt.Errorf("failed to query classic environment url after fallback to %q: %w", deprecatedClassicEnvDomainPath, err)
		}
	}

	if !resp.IsSuccess() {
		return "", RespError{
			Err:        fmt.Errorf("failed to query classic environment URL: (HTTP %v) %v", resp.StatusCode, string(resp.Body)),
			StatusCode: resp.StatusCode,
		}
	}

	var jsonResp classicEnvURL
	if err := json.Unmarshal(resp.Body, &jsonResp); err != nil {
		return "", fmt.Errorf("failed to parse classic environment response payload: %w", err)
	}
	return jsonResp.GetURL(), nil
}
