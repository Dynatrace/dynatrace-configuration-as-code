//go:build unit

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

package manifest

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultTokenEndpoint(t *testing.T) {
	t.Run("Token endpoint value is returned if set", func(t *testing.T) {
		o := OAuth{
			TokenEndpoint: &URLDefinition{
				Type:  ValueURLType,
				Value: "https://my-token-endpoint.com",
			},
		}
		assert.Equal(t, "https://my-token-endpoint.com", o.GetTokenEndpointValue())

	})

	t.Run("Default token endpoint is returned if none is set", func(t *testing.T) {
		o := OAuth{}
		assert.Equal(t, "https://sso.dynatrace.com/sso/oauth2/token", o.GetTokenEndpointValue())

		o2 := OAuth{
			TokenEndpoint: &URLDefinition{
				Type:  ValueURLType,
				Value: "",
			},
		}
		assert.Equal(t, "https://sso.dynatrace.com/sso/oauth2/token", o2.GetTokenEndpointValue())
	})
}
