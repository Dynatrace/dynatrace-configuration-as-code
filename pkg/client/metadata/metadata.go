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
	"context"
	"encoding/json"
	"errors"
	"fmt"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
)

const ClassicEnvironmentDomainPath = "/platform/metadata/v1/classic-environment-domain"

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
func GetDynatraceClassicURL(ctx context.Context, platformClient corerest.Client) (string, error) {
	resp, err := coreapi.AsResponseOrError(platformClient.GET(ctx, ClassicEnvironmentDomainPath, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		apiErr := coreapi.APIError{}
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 401 && apiErr.StatusCode <= 403 {
			return "", fmt.Errorf("missing permissions to query classic environment URL: oAuth client may be missing required scope 'app-engine:apps:run': %w", err)
		}
		return "", fmt.Errorf("failed to query classic environment URL: %w", err)
	}

	var jsonResp classicEnvURL
	err = json.Unmarshal(resp.Data, &jsonResp)
	if err != nil {
		// for specific Dynatrace base URLs (e.g. https://env-id.live.dynatrace.com) we get back an HTTP 200 OK,
		// however the payload is not the expected JSON but HTML content.
		// At this point, best we can do is give the user a hint that the URL is not completely correct
		return "", fmt.Errorf("failed to parse classic environment response payload from %q. Please check your dynatrace environment URL to match the following pattern: https://<env-id>.apps.dynatrace.com", platformClient.BaseURL())
	}
	return jsonResp.GetURL(), nil
}
