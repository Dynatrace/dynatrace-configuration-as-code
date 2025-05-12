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

package platform

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/oauth2/clientcredentials"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/classicheartbeat"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/metadata"
)

func GetDynatraceClassicURL(ctx context.Context, platformURL string, oauthCreds clientcredentials.Config) (string, error) {
	if featureflags.BuildSimpleClassicURL.Enabled() {
		if classicURL, ok := findSimpleClassicURL(ctx, platformURL); ok {
			return classicURL, nil
		}
	}

	client, err := clients.Factory().WithPlatformURL(platformURL).WithOAuthCredentials(oauthCreds).CreatePlatformClient(ctx)
	if err != nil {
		return "", err
	}
	return metadata.GetDynatraceClassicURL(ctx, *client)
}

func findSimpleClassicURL(ctx context.Context, platformURL string) (classicUrl string, ok bool) {
	if !strings.Contains(platformURL, ".apps.") {
		log.Debug("Environment URL not matching expected Platform URL pattern, unable to build Classic environment URL directly.")
		return "", false
	}

	replaceWith := ".live."
	if match, _ := regexp.Match(`(.*)\.(.*)\.apps\.`, []byte(platformURL)); match {
		replaceWith = "."
	}
	classicUrl = strings.Replace(platformURL, ".apps.", replaceWith, 1)

	parsedUrl, err := url.Parse(classicUrl)
	if err != nil {
		log.Debug("Invalid environment URL: %s", err)
		return "", false
	}
	cl := corerest.NewClient(parsedUrl, &http.Client{}, corerest.WithRateLimiter())

	if classicheartbeat.TestClassic(ctx, *cl) {
		log.Debug("Found classic environment URL based on Platform URL: %s", classicUrl)
		return classicUrl, true
	}

	return "", false
}
