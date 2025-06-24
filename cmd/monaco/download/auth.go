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

package download

import (
	"fmt"
	"os"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

type auth struct {
	accessToken, clientID, clientSecret, platformToken string
}

func (a auth) mapToAuth() (*manifest.Auth, []error) {
	errs := make([]error, 0)
	mAuth := manifest.Auth{}

	if a.accessToken != "" {
		if token, err := readAuthSecretFromEnvVariable(a.accessToken); err != nil {
			errs = append(errs, err)
		} else {
			mAuth.AccessToken = &token
		}
	}

	if a.clientID != "" && a.clientSecret != "" {
		mAuth.OAuth = &manifest.OAuth{}
		if clientId, err := readAuthSecretFromEnvVariable(a.clientID); err != nil {
			errs = append(errs, err)
		} else {
			mAuth.OAuth.ClientID = clientId
		}
		if clientSecret, err := readAuthSecretFromEnvVariable(a.clientSecret); err != nil {
			errs = append(errs, err)
		} else {
			mAuth.OAuth.ClientSecret = clientSecret
		}
	}

	if a.platformToken != "" {
		if platformToken, err := readAuthSecretFromEnvVariable(a.platformToken); err != nil {
			errs = append(errs, err)
		} else {
			mAuth.PlatformToken = &platformToken
		}
	}
	return &mAuth, errs
}

func readAuthSecretFromEnvVariable(envVar string) (manifest.AuthSecret, error) {
	var content string
	if envVar == "" {
		return manifest.AuthSecret{}, fmt.Errorf("unknown environment variable name")
	} else if content = os.Getenv(envVar); content == "" {
		return manifest.AuthSecret{}, fmt.Errorf("the content of the environment variable %q is not set", envVar)
	}
	return manifest.AuthSecret{Name: envVar, Value: secret.MaskedString(content)}, nil
}
