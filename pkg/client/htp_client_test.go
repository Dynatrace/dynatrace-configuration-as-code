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

package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
)

func TestSetCustomHTTPClientInContext(t *testing.T) {
	t.Run("Sets a custom client if skip SSL env is 'true'", func(t *testing.T) {
		t.Setenv(featureflags.SkipCertificateVerification.EnvName(), "true")

		ctx := client.SetCustomHTTPClientInContext(t.Context())
		val := ctx.Value(oauth2.HTTPClient)

		require.NotNil(t, val)
		assert.IsType(t, val, &http.Client{})
	})

	t.Run("Doesn't set a custom client if skip SSL env isn't set", func(t *testing.T) {
		ctx := client.SetCustomHTTPClientInContext(t.Context())
		val := ctx.Value(oauth2.HTTPClient)

		assert.Nil(t, val)
	})
}
