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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"net/http"
	"net/url"
)

const classicEnvironmentDomainPath = "/platform/core/v1/environment-api-info" // NOTE: once available, change this to /platform/metadata/v1/classic-environment-domain

type classicEnvURL struct {
	Endpoint string `json:"endpoint"`
}

// GetDynatraceClassicURL tries to fetch the URL of the classic environment using the API of a platform enabled
// environment
func GetDynatraceClassicURL(client *http.Client, environmentURL string) (string, error) {
	endpointURL, err := url.JoinPath(environmentURL, classicEnvironmentDomainPath)
	if err != nil {
		return "", fmt.Errorf("failed to build URL for API %q on environment URL %q", classicEnvironmentDomainPath, environmentURL)
	}

	resp, err := rest.Get(client, endpointURL)
	if err != nil {
		return "", fmt.Errorf("failed to query classic environment url %w", err)
	}

	if !resp.IsSuccess() {
		return "", RespError{
			Err:        fmt.Errorf("failed to query classic environment URL: (HTTP %v) %v", resp.StatusCode, string(resp.Body)),
			StatusCode: resp.StatusCode,
		}
	}

	var jsonResp classicEnvURL
	if err := json.Unmarshal(resp.Body, &jsonResp); err != nil {
		return "", fmt.Errorf("failed to parse Dynatrace version JSON: %w", err)
	}
	return jsonResp.Endpoint, nil
}
