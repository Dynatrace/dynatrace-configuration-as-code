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

package manifest_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

func TestDefaultTokenEndpoint(t *testing.T) {
	t.Run("Token endpoint value is returned if set", func(t *testing.T) {
		o := manifest.OAuth{
			TokenEndpoint: &manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: "https://my-token-endpoint.com",
			},
		}
		assert.Equal(t, "https://my-token-endpoint.com", o.GetTokenEndpointValue())

	})

	t.Run("Default token endpoint is returned if none is set", func(t *testing.T) {
		o := manifest.OAuth{}
		assert.Equal(t, "https://sso.dynatrace.com/sso/oauth2/token", o.GetTokenEndpointValue())

		o2 := manifest.OAuth{
			TokenEndpoint: &manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: "",
			},
		}
		assert.Equal(t, "https://sso.dynatrace.com/sso/oauth2/token", o2.GetTokenEndpointValue())
	})
}

func TestManifestLoading(t *testing.T) {
	fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())
	fs.Mkdir("./testdata/grouping", 0644)
	fs.Mkdir("./testdata/grouping/sub", 0644)

	t.Setenv("ENV_URL", "https://some.url")
	t.Setenv("ENV_TOKEN", "dt01.token")
	t.Setenv("ENV_CLIENT_ID", "dt02.id")
	t.Setenv("ENV_CLIENT_SECRET", "dt02.secret")
	t.Setenv("ENV_TOKEN_URL", "https://another-token.url")
	t.Setenv("ENV_API_URL", "https://api.url")
	t.Setenv("ENV_UUID", "8f9935ee-2068-455d-85ce-47447f19d5d5")

	mani, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: "./testdata/manifest_full.yaml",
		Opts: manifestloader.Options{
			DoNotResolveEnvVars:      false,
			RequireEnvironmentGroups: true,
		},
	})

	assert.NoError(t, errors.Join(errs...), "manifest loading should not produce any errors")

	assert.Equal(t, mani, manifest.Manifest{
		Projects: manifest.ProjectDefinitionByProjectID{
			"simple": manifest.ProjectDefinition{
				Name:  "simple",
				Group: "",
				Path:  "simple",
			},
			"with-path": manifest.ProjectDefinition{
				Name:  "with-path",
				Group: "",
				Path:  "here-i-am",
			},
			"grouping.sub": manifest.ProjectDefinition{
				Name:  "grouping.sub",
				Group: "grouping",
				Path:  filepath.FromSlash("grouping/sub"),
			},
			"grouping-without-path.grouping": manifest.ProjectDefinition{
				Name:  "grouping-without-path.grouping",
				Group: "grouping-without-path",
				Path:  "grouping",
			},
		},
		Environments: manifest.Environments{
			"test-env-1": manifest.EnvironmentDefinition{
				Enabled: true,
				Name:    "test-env-1",
				Group:   "dev",
				URL: manifest.URLDefinition{
					Type:  manifest.EnvironmentURLType,
					Name:  "ENV_URL",
					Value: "https://some.url",
				},
				Auth: manifest.Auth{
					Token: &manifest.AuthSecret{
						Name:  "ENV_TOKEN",
						Value: "dt01.token",
					},
					OAuth: &manifest.OAuth{
						ClientID: manifest.AuthSecret{
							Name:  "ENV_CLIENT_ID",
							Value: "dt02.id",
						},
						ClientSecret: manifest.AuthSecret{
							Name:  "ENV_CLIENT_SECRET",
							Value: "dt02.secret",
						},
						TokenEndpoint: &manifest.URLDefinition{
							Type:  manifest.ValueURLType,
							Name:  "",
							Value: "https://my-token.url",
						},
					},
				},
			},
			"test-env-2": manifest.EnvironmentDefinition{
				Enabled: true,
				Name:    "test-env-2",
				Group:   "dev",
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Name:  "",
					Value: "https://ddd.bbb.cc",
				},
				Auth: manifest.Auth{
					Token: &manifest.AuthSecret{
						Name:  "ENV_TOKEN",
						Value: "dt01.token",
					},
					OAuth: nil,
				},
			},
			"prod-env-1": manifest.EnvironmentDefinition{
				Enabled: true,
				Name:    "prod-env-1",
				Group:   "prod",
				URL: manifest.URLDefinition{
					Type:  manifest.EnvironmentURLType,
					Name:  "ENV_URL",
					Value: "https://some.url",
				},
				Auth: manifest.Auth{
					Token: &manifest.AuthSecret{
						Name:  "ENV_TOKEN",
						Value: "dt01.token",
					},
					OAuth: nil,
				},
			},
		},
		Accounts: map[string]manifest.Account{
			"my-account": {
				Name:        "my-account",
				AccountUUID: uuid.MustParse("8f9935ee-2068-455d-85ce-47447f19d5d5"),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "ENV_CLIENT_ID",
						Value: "dt02.id",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "ENV_CLIENT_SECRET",
						Value: "dt02.secret",
					},
					TokenEndpoint: nil,
				},
			},
			"other-account": {
				Name:        "other-account",
				AccountUUID: uuid.MustParse("c3f50f90-a1e2-4e7b-aadb-f3dea28e2294"),
				ApiUrl: &manifest.URLDefinition{
					Type:  manifest.EnvironmentURLType,
					Name:  "ENV_API_URL",
					Value: "https://api.url",
				},
				OAuth: manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "ENV_CLIENT_ID",
						Value: "dt02.id",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "ENV_CLIENT_SECRET",
						Value: "dt02.secret",
					},
					TokenEndpoint: &manifest.URLDefinition{
						Type:  manifest.EnvironmentURLType,
						Name:  "ENV_TOKEN_URL",
						Value: "https://another-token.url",
					},
				},
			},
			"account-full-uuid-type": {
				Name:        "account-full-uuid-type",
				AccountUUID: uuid.MustParse("c3f50f90-a1e2-4e7b-aadb-f3dea28e2294"),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "ENV_CLIENT_ID",
						Value: "dt02.id",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "ENV_CLIENT_SECRET",
						Value: "dt02.secret",
					},
					TokenEndpoint: nil,
				},
			},
			"account-environment-uuid-type": {
				Name:        "account-environment-uuid-type",
				AccountUUID: uuid.MustParse("8f9935ee-2068-455d-85ce-47447f19d5d5"),
				ApiUrl:      nil,
				OAuth: manifest.OAuth{
					ClientID: manifest.AuthSecret{
						Name:  "ENV_CLIENT_ID",
						Value: "dt02.id",
					},
					ClientSecret: manifest.AuthSecret{
						Name:  "ENV_CLIENT_SECRET",
						Value: "dt02.secret",
					},
					TokenEndpoint: nil,
				},
			},
		},
	})
}

func TestManifestLoading_AccountsInvalid(t *testing.T) {
	t.Setenv("SECRET", "secret")

	tc := []struct {
		name   string
		accDef string
	}{
		{
			name: "Empty name",
			accDef: `
manifestVersion: "1.0"
accounts:
- name: ""
  accountUUID: 8f9935ee-2068-455d-85ce-47447f19d5d5
  apiUrl:
    value: "https://[13::37]:42"
  oAuth:
    clientId:
      name: SECRET
    clientSecret:
      name: SECRET
`,
		},
		{
			name: "Missing name",
			accDef: `
manifestVersion: "1.0"
accounts:
- accountUUID: 8f9935ee-2068-455d-85ce-47447f19d5d5
  oAuth:
    clientId:
      name: SECRET
    clientSecret:
      name: SECRET
`,
		},
		{
			name: "Missing account uuid",
			accDef: `
manifestVersion: "1.0"
accounts:
- name: name
  oAuth:
    clientId:
      name: SECRET
    clientSecret:
      name: SECRET
`,
		},
		{
			name: "Empty account uuid",
			accDef: `
manifestVersion: "1.0"
accounts:
- name: name
  accountUUID: ""
  oAuth:
    clientId:
      name: SECRET
    clientSecret:
      name: SECRET
`,
		},
		{
			name: "Missing oauth",
			accDef: `
manifestVersion: "1.0"
accounts:
- name: name
  accountUUID: 8f9935ee-2068-455d-85ce-47447f19d5d5
`,
		},
		{
			name: "Missing client id",
			accDef: `
manifestVersion: "1.0"
accounts:
- name: name
  accountUUID: 8f9935ee-2068-455d-85ce-47447f19d5d5
  oAuth:
    clientSecret:
      name: SECRET
`,
		},
		{
			name: "Missing client secret",
			accDef: `
manifestVersion: "1.0"
accounts:
- name: name
  accountUUID: 8f9935ee-2068-455d-85ce-47447f19d5d5
  oAuth:
    clientId:
      name: SECRET
`,
		},
	}

	baseDef := `
projects: [{name: proj}]
environmentGroups:
- name: a
  environments:
  - name: b
    url: {value: "https://e.url"}
    auth: {token: {name: "SECRET"}}
`

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())

			fullDef := baseDef + tt.accDef

			afero.WriteFile(fs, "manifest.yaml", []byte(fullDef), 0644)
			fs.Mkdir("proj", 0644)

			mani, errs := manifestloader.Load(&manifestloader.Context{
				Fs:           fs,
				ManifestPath: "manifest.yaml",
			})

			assert.Equal(t, mani, manifest.Manifest{}, "manifest should not contain any info")
			assert.NotEmpty(t, errs, "errors should occur")
		})
	}
}
