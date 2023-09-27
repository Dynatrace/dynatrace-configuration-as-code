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
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"strings"
)

// OauthCredentials holds information for authenticating to Dynatrace
// using Oauth2.0 client credential flow
type OauthCredentials struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// NewTokenAuthClient creates a new HTTP client that supports token based authorization
func NewTokenAuthClient(token string) *http.Client {
	if !isNewDynatraceTokenFormat(token) {
		log.Warn("The supplied token does not match the expected format and may be invalid. If authentication fails, please check your manifest and environment variable configuration.\nIf you are using a token created before Dynatrace 1.205, please consider generating a new token: https://www.dynatrace.com/support/help/shortlink/api-authentication")
	}
	return &http.Client{Transport: NewTokenAuthTransport(nil, token)}
}

// NewOAuthClient creates a new HTTP client that supports OAuth2 client credentials based authorization
func NewOAuthClient(ctx context.Context, oauthConfig OauthCredentials) *http.Client {
	config := clientcredentials.Config{
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		TokenURL:     oauthConfig.TokenURL,
		Scopes:       oauthConfig.Scopes,
	}
	return config.Client(ctx)
}

func isNewDynatraceTokenFormat(token string) bool {
	return strings.HasPrefix(token, "dt0c01.") && strings.Count(token, ".") == 2
}
