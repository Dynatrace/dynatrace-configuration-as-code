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

package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
)

func TestValidateDynatraceDomain(t *testing.T) {
	t.Run("tokenEndpoint", func(t *testing.T) {
		tests := []struct {
			name    string
			url     string
			wantErr bool
		}{
			{"subdomain of dynatrace.com", "https://sso.dynatrace.com/sso/oauth2/token", false},
			{"subdomain of dynatracelabs.com", "https://sso.dynatracelabs.com/token", false},
			{"deep subdomain of dynatracelabs.com", "https://a.b.dynatracelabs.com/token", false},
			{"deep subdomain of dynatrace.com", "https://a.b.dynatrace.com/token", false},
			{"subdomain without path", "https://b.dynatrace.com", false},
			{"unrelated domain", "https://evil.com/steal", true},
			{"lookalike - not a subdomain", "https://evil-dynatrace.com/token", true},
			{"dynatrace.com embedded in path", "https://evil.com/dynatrace.com/token", true},
			{"dynatrace.com as subdomain of attacker domain", "https://sso.dynatrace.com.evil.com/token", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateDynatraceDomain(tt.url)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("apiUrl", func(t *testing.T) {
		t.Run("accepts valid dynatrace.com host", func(t *testing.T) {
			assert.NoError(t, validateDynatraceDomain("https://api.dynatrace.com"))
		})

		t.Run("accepts valid dynatracelabs.com host", func(t *testing.T) {
			assert.NoError(t, validateDynatraceDomain("https://sso.dynatracelabs.com"))
		})

		t.Run("accepts subdomain on allowed domain", func(t *testing.T) {
			assert.NoError(t, validateDynatraceDomain("https://abc.def.dynatrace.com"))
		})

		t.Run("accepts uppercase host (case-insensitive)", func(t *testing.T) {
			assert.NoError(t, validateDynatraceDomain("https://API.DYNATRACE.COM"))
		})

		t.Run("accepts host with port", func(t *testing.T) {
			assert.NoError(t, validateDynatraceDomain("https://api.dynatrace.com:8443/account"))
		})

		t.Run("rejects attacker-controlled host", func(t *testing.T) {
			err := validateDynatraceDomain("https://attacker.example.com/steal")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not allowed")
		})

		t.Run("rejects look-alike suffix without leading dot", func(t *testing.T) {
			err := validateDynatraceDomain("https://evil-dynatrace.com")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not allowed")
		})

		t.Run("rejects domain that only ends in dynatrace.com via path", func(t *testing.T) {
			err := validateDynatraceDomain("https://attacker.com/api.dynatrace.com")
			require.Error(t, err)
		})

		t.Run("rejects RFC1918 SSRF target", func(t *testing.T) {
			err := validateDynatraceDomain("http://10.0.0.1:8080/admin")
			require.Error(t, err)
		})

		t.Run("rejects cloud metadata SSRF target", func(t *testing.T) {
			err := validateDynatraceDomain("http://169.254.169.254/latest/meta-data/")
			require.Error(t, err)
		})

		t.Run("rejects userinfo prefix masquerade", func(t *testing.T) {
			// https://api.dynatrace.com@attacker.com -> actual host is attacker.com
			err := validateDynatraceDomain("https://api.dynatrace.com@attacker.com/")
			require.Error(t, err)
		})

		t.Run("rejects malformed URL", func(t *testing.T) {
			err := validateDynatraceDomain("::::not a url")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not a valid URL")
		})
	})
}

func TestParseSingleAccount_RejectsNonDynatraceApiUrl(t *testing.T) {
	ctx := &Context{Opts: Options{}}
	a := persistence.Account{
		Name:        "name",
		AccountUUID: persistence.TypedValue{Value: "8f9935ee-2068-455d-85ce-47447f19d5d5"},
		ApiUrl:      &persistence.TypedValue{Value: "https://attacker.example.com/steal"},
		OAuth: persistence.OAuth{
			ClientID:     persistence.AuthSecret{Name: "ID"},
			ClientSecret: persistence.AuthSecret{Name: "SECRET"},
		},
	}
	t.Setenv("ID", "id")
	t.Setenv("SECRET", "secret")

	_, err := parseSingleAccount(ctx, a)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestParseSingleAccount_AcceptsDynatraceApiUrl(t *testing.T) {
	ctx := &Context{Opts: Options{}}
	a := persistence.Account{
		Name:        "name",
		AccountUUID: persistence.TypedValue{Value: "8f9935ee-2068-455d-85ce-47447f19d5d5"},
		ApiUrl:      &persistence.TypedValue{Value: "https://api.dynatrace.com"},
		OAuth: persistence.OAuth{
			ClientID:     persistence.AuthSecret{Name: "ID"},
			ClientSecret: persistence.AuthSecret{Name: "SECRET"},
		},
	}
	t.Setenv("ID", "id")
	t.Setenv("SECRET", "secret")

	acc, err := parseSingleAccount(ctx, a)
	require.NoError(t, err)
	require.NotNil(t, acc.ApiUrl)
	assert.Equal(t, "https://api.dynatrace.com", acc.ApiUrl.Value)
}
