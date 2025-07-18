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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
)

var (
	// default account to permute
	validAccount = persistence.Account{
		Name: "name",
		AccountUUID: persistence.TypedValue{
			Value: uuid.New().String(),
		},
		ApiUrl: &persistence.TypedValue{
			Value: "https://example.com",
		},
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: &persistence.TypedValue{
				Value: "https://example.com",
			},
		},
	}
)

func TestValidAccounts(t *testing.T) {
	t.Setenv("SECRET", "secret")

	// full account 1
	acc := persistence.Account{
		Name: "name",
		AccountUUID: persistence.TypedValue{
			Type:  persistence.TypeValue,
			Value: uuid.New().String(),
		},
		ApiUrl: &persistence.TypedValue{
			Value: "https://example.com",
		},
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: &persistence.TypedValue{
				Value: "https://example.com",
			},
		},
	}

	// account 2 has no api url
	acc2 := persistence.Account{
		Name: "name2",
		AccountUUID: persistence.TypedValue{
			Value: uuid.New().String(),
		},
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: nil,
		},
	}

	//account 3 has UUID defined as env var
	envUUID := uuid.New().String()
	t.Setenv("ACC_3_UUID_ENV_VAR", envUUID)
	acc3 := persistence.Account{
		Name: "name3",
		AccountUUID: persistence.TypedValue{
			Type:  persistence.TypeEnvironment,
			Value: "ACC_3_UUID_ENV_VAR",
		},
		OAuth: persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Name: "SECRET",
			},
			ClientSecret: persistence.AuthSecret{
				Name: "SECRET",
			},
			TokenEndpoint: nil,
		},
	}

	t.Run("full account", func(t *testing.T) {
		v, err := parseAccounts(&Context{}, []persistence.Account{acc})
		assert.NoError(t, err)
		assert.Equal(t, v, map[string]manifest.Account{
			"name": {
				Name:        "name",
				AccountUUID: uuid.MustParse(acc.AccountUUID.Value),
				ApiUrl: &manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Value: "https://example.com",
				},
				OAuth: manifest.OAuth{
					ClientID:     manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					ClientSecret: manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					TokenEndpoint: &manifest.URLDefinition{
						Type:  manifest.ValueURLType,
						Value: "https://example.com",
					},
				},
			},
		})
	})

	t.Run("simple account", func(t *testing.T) {
		v, err := parseAccounts(&Context{}, []persistence.Account{acc2})
		assert.NoError(t, err)
		assert.Equal(t, v, map[string]manifest.Account{
			"name2": {
				Name:        "name2",
				AccountUUID: uuid.MustParse(acc2.AccountUUID.Value),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID:      manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					ClientSecret:  manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					TokenEndpoint: nil,
				},
			},
		})
	})

	t.Run("env var uuid account", func(t *testing.T) {
		v, err := parseAccounts(&Context{}, []persistence.Account{acc3})
		assert.NoError(t, err)
		assert.Equal(t, v, map[string]manifest.Account{
			"name3": {
				Name:        "name3",
				AccountUUID: uuid.MustParse(envUUID),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID:      manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					ClientSecret:  manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					TokenEndpoint: nil,
				},
			},
		})
	})

	t.Run("several accounts", func(t *testing.T) {
		v, err := parseAccounts(&Context{}, []persistence.Account{acc, acc2, acc3})
		assert.NoError(t, err)

		assert.Equal(t, v, map[string]manifest.Account{
			"name": {
				Name:        "name",
				AccountUUID: uuid.MustParse(acc.AccountUUID.Value),
				ApiUrl: &manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Value: "https://example.com",
				},
				OAuth: manifest.OAuth{
					ClientID:     manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					ClientSecret: manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					TokenEndpoint: &manifest.URLDefinition{
						Type:  manifest.ValueURLType,
						Value: "https://example.com",
					},
				},
			},
			"name2": {
				Name:        "name2",
				AccountUUID: uuid.MustParse(acc2.AccountUUID.Value),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID:      manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					ClientSecret:  manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					TokenEndpoint: nil,
				},
			},
			"name3": {
				Name:        "name3",
				AccountUUID: uuid.MustParse(envUUID),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID:      manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					ClientSecret:  manifest.AuthSecret{Name: "SECRET", Value: "secret"},
					TokenEndpoint: nil,
				},
			},
		})
	})

}

func TestInvalidAccounts(t *testing.T) {
	t.Setenv("SECRET", "secret")

	// validate that the default is valid
	_, err := parseAccounts(&Context{}, []persistence.Account{validAccount})
	assert.NoError(t, err)

	// tests
	t.Run("name is missing", func(t *testing.T) {
		a := validAccount
		a.Name = ""

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		assert.ErrorIs(t, err, errNameMissing)
	})

	t.Run("accountUUID is missing", func(t *testing.T) {
		a := validAccount
		a.AccountUUID.Value = ""

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		assert.ErrorIs(t, err, errAccUidMissing)
	})

	t.Run("accountUUID is invalid", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.AccountUUID.Value = "this-is-not-a-valid-uuid"

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		uuidErr := invalidUUIDError{}
		if assert.ErrorAs(t, err, &uuidErr) {
			assert.Equal(t, uuidErr.uuid, "this-is-not-a-valid-uuid")
		}
	})

	t.Run("accountUUID is invalid type", func(t *testing.T) {
		a := validAccount
		a.AccountUUID.Type = "this-is-not-a-type"

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		assert.Error(t, err)
	})

	t.Run("oAuth is set", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.OAuth = persistence.OAuth{}

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		assert.ErrorContains(t, err, "oAuth is invalid")
	})

	t.Run("oAuth.id is not set", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.OAuth.ClientID = persistence.AuthSecret{}

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		assert.ErrorContains(t, err, "ClientID: no name given or empty")

	})

	t.Run("oAuth.secret is not set", func(t *testing.T) {
		a := deepCopy(t, validAccount)
		a.OAuth.ClientSecret = persistence.AuthSecret{}

		_, err := parseAccounts(&Context{}, []persistence.Account{a})
		assert.ErrorContains(t, err, "ClientSecret: no name given or empty")
	})
}

func TestSelectedAccounts(t *testing.T) {
	a := deepCopy(t, validAccount)
	a.OAuth.ClientSecret = persistence.AuthSecret{
		Name: "SECRET_2",
	}
	b := deepCopy(t, validAccount)
	b.Name = "other"
	accounts := []persistence.Account{a, b}
	t.Setenv("SECRET", "secret")

	t.Run("Returns selected account", func(t *testing.T) {
		parsedAccounts, err := parseAccounts(&Context{Account: b.Name}, accounts)
		assert.NoError(t, err)
		require.Len(t, parsedAccounts, 1)

		account, ok := parsedAccounts[b.Name]

		assert.True(t, ok)
		assert.Equal(t, b.Name, account.Name)
	})

	t.Run("Returns all accounts", func(t *testing.T) {
		t.Setenv("SECRET_2", "secret")
		parsedAccounts, err := parseAccounts(&Context{}, accounts)

		assert.NoError(t, err)
		require.Len(t, parsedAccounts, 2)
	})

	t.Run("Returns an error if not account matches", func(t *testing.T) {
		notExistingAccount := "not-existing"
		_, err := parseAccounts(&Context{Account: notExistingAccount}, accounts)

		assert.ErrorContains(t, err, fmt.Sprintf("'%s' was not found", notExistingAccount))
	})
}

// deepCopy marshals and then marshals the payload, thus only works for public members, thus only private spaced
func deepCopy(t *testing.T, in persistence.Account) persistence.Account {
	d, e := json.Marshal(in)
	assert.NoError(t, e)

	var o persistence.Account
	e = json.Unmarshal(d, &o)
	assert.NoError(t, e)
	return o
}
