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

package download

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetApisToDownload(t *testing.T) {
	type given struct {
		apis         api.APIs
		specificAPIs []string
	}
	type expected struct {
		apis []string
	}
	tests := []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name: "filter all specific defined api",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				specificAPIs: []string{"api_1"},
			},
			expected: expected{
				apis: []string{"api_1"},
			},
		}, {
			name: "if deprecated api is defined, do not filter it",
			given: given{
				apis: api.APIs{
					"api_1":          api.API{ID: "api_1"},
					"api_2":          api.API{ID: "api_2"},
					"deprecated_api": api.API{ID: "deprecated_api", DeprecatedBy: "new_api"},
				},
				specificAPIs: []string{"api_1", "deprecated_api"},
			},
			expected: expected{
				apis: []string{"api_1", "deprecated_api"},
			},
		},
		{
			name: "if specific api is not requested, filter deprecated apis",
			given: given{
				apis: api.APIs{
					"api_1":          api.API{ID: "api_1"},
					"api_2":          api.API{ID: "api_2"},
					"deprecated_api": api.API{ID: "deprecated_api", DeprecatedBy: "new_api"},
				},
				specificAPIs: []string{},
			},
			expected: expected{
				apis: []string{"api_1", "api_2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getApisToDownload(tt.given.apis, tt.given.specificAPIs)
			for _, e := range tt.expected.apis {
				assert.Contains(t, actual, e)
			}
		})
	}
}

func TestDownloadConfigsBehaviour(t *testing.T) {
	tests := []struct {
		name              string
		givenOpts         downloadConfigsOptions
		expectedBehaviour func(client *client.MockClient)
	}{
		{
			name: "Default opts: downloads Configs and Settings",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    nil,
				specificSchemas: nil,
				onlyAPIs:        false,
				onlySettings:    false,
			},
			expectedBehaviour: func(c *client.MockClient) {
				c.EXPECT().ListConfigs(gomock.Any()).AnyTimes().Return([]client.Value{}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("{}"), nil) // singleton configs are always attempted
				c.EXPECT().ListSchemas().Return(client.SchemaList{}, nil)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).AnyTimes().Return([]client.DownloadSettingsObject{}, nil)
			},
		},
		{
			name: "Specific Settings: downloads defined Settings only",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    nil,
				specificSchemas: []string{"builtin:magic.secret"},
				onlyAPIs:        false,
				onlySettings:    false,
			},
			expectedBehaviour: func(c *client.MockClient) {
				c.EXPECT().ListConfigs(gomock.Any()).Times(0)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Times(0)
				c.EXPECT().ListSettings("builtin:magic.secret", gomock.Any()).AnyTimes().Return([]client.DownloadSettingsObject{}, nil)
			},
		},
		{
			name: "Specific APIs: downloads defined APIs only",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    []string{"alerting-profile"},
				specificSchemas: nil,
				onlyAPIs:        false,
				onlySettings:    false,
			},
			expectedBehaviour: func(c *client.MockClient) {
				c.EXPECT().ListConfigs(api.NewAPIs()["alerting-profile"]).Return([]client.Value{{Id: "42", Name: "profile"}}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), "42").AnyTimes().Return([]byte("{}"), nil)
				c.EXPECT().ListSchemas().Times(0)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "Specific APIs and Settings: downloads defined APIs and Schemas",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    []string{"alerting-profile"},
				specificSchemas: []string{"builtin:magic.secret"},
				onlyAPIs:        false,
				onlySettings:    false,
			},
			expectedBehaviour: func(c *client.MockClient) {
				c.EXPECT().ListConfigs(api.NewAPIs()["alerting-profile"]).Return([]client.Value{{Id: "42", Name: "profile"}}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), "42").AnyTimes().Return([]byte("{}"), nil)
				c.EXPECT().ListSettings("builtin:magic.secret", gomock.Any()).AnyTimes().Return([]client.DownloadSettingsObject{}, nil)

			},
		},
		{
			name: "Only APIs: downloads APIs only",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    nil,
				specificSchemas: nil,
				onlyAPIs:        true,
				onlySettings:    false,
			},
			expectedBehaviour: func(c *client.MockClient) {
				c.EXPECT().ListConfigs(gomock.Any()).AnyTimes().Return([]client.Value{}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("{}"), nil) // singleton configs are always attempted
				c.EXPECT().ListSchemas().Times(0)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "Only Settings: downloads Settings only",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    nil,
				specificSchemas: nil,
				onlyAPIs:        false,
				onlySettings:    true,
			},
			expectedBehaviour: func(c *client.MockClient) {
				c.EXPECT().ListConfigs(gomock.Any()).Times(0)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Times(0)
				c.EXPECT().ListSchemas().Return(client.SchemaList{}, nil)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).AnyTimes().Return([]client.DownloadSettingsObject{}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockClient(gomock.NewController(t))

			tt.givenOpts.downloadOptionsShared = downloadOptionsShared{
				environmentURL: "testurl.com",
				auth: manifest.Auth{
					Token: manifest.AuthSecret{
						Name:  "TEST_TOKEN_VAR",
						Value: "test.token",
					},
				},
				outputFolder:            "folder",
				projectName:             "project",
				forceOverwriteManifest:  false,
				concurrentDownloadLimit: 1,
			}

			tt.expectedBehaviour(c)

			_, err := downloadConfigs(c, api.NewAPIs(), tt.givenOpts)
			assert.NoError(t, err)
		})
	}
}

func Test_shouldDownloadAPIs(t *testing.T) {
	tests := []struct {
		name  string
		given downloadConfigsOptions
		want  bool
	}{
		{
			name: "true if not 'onlySettings'",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       nil,
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "true if 'onlyAPIs'",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       nil,
				onlyAPIs:              true,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "true if 'specificAPIs' defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          []string{"some-api", "other-api"},
				specificSchemas:       nil,
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "false if just 'specificSchemas' defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       []string{"some-schema", "other-schema"},
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: false,
		},
		{
			name: "true if 'specificAPIs' and 'specificSchemas' defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          []string{"some-api", "other-api"},
				specificSchemas:       []string{"some-schema", "other-schema"},
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "false if 'specificSchemas' and onlySettings defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       []string{"some-schema", "other-schema"},
				onlyAPIs:              false,
				onlySettings:          true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, shouldDownloadClassicConfigs(tt.given), "shouldDownloadApis(%v)", tt.given)
		})
	}
}

func Test_shouldDownloadSettings(t *testing.T) {
	tests := []struct {
		name  string
		given downloadConfigsOptions
		want  bool
	}{
		{
			name: "true if not 'onlyAPIs'",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       nil,
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "true if 'onlySettings'",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       nil,
				onlyAPIs:              false,
				onlySettings:          true,
			},
			want: true,
		},
		{
			name: "true if only 'specificSettings' defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          nil,
				specificSchemas:       []string{"some-schema", "other-schema"},
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "false if 'specificAPIs' defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          []string{"some-api", "other-api"},
				specificSchemas:       nil,
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: false,
		},
		{
			name: "true if 'specificAPIs' and 'specificSchemas' defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          []string{"some-api", "other-api"},
				specificSchemas:       []string{"some-schema", "other-schema"},
				onlyAPIs:              false,
				onlySettings:          false,
			},
			want: true,
		},
		{
			name: "false if 'specificAPIs' and onlyAPIs defined",
			given: downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{},
				specificAPIs:          []string{"some-api", "other-api"},
				specificSchemas:       nil,
				onlyAPIs:              true,
				onlySettings:          false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, shouldDownloadSettings(tt.given), "shouldDownloadSettings(%v)", tt.given)
		})
	}
}

func Test_validateSpecificSettings(t *testing.T) {
	type given struct {
		settingsOnEnvironment     client.SchemaList
		specificSettingsRequested []string
	}
	tests := []struct {
		name               string
		given              given
		wantValid          bool
		wantUnknownSchemas []string
	}{
		{
			"valid if setting is found",
			given{
				settingsOnEnvironment:     client.SchemaList{{"builtin:magic.setting"}},
				specificSettingsRequested: []string{"builtin:magic.setting"},
			},
			true,
			nil,
		},
		{
			"not valid if setting not found",
			given{
				settingsOnEnvironment:     client.SchemaList{{"builtin:magic.setting"}},
				specificSettingsRequested: []string{"builtin:unknown"},
			},
			false,
			[]string{"builtin:unknown"},
		},
		{
			"not valid if one setting not found",
			given{
				settingsOnEnvironment:     client.SchemaList{{"builtin:magic.setting"}},
				specificSettingsRequested: []string{"builtin:magic.setting", "builtin:unknown"},
			},
			false,
			[]string{"builtin:unknown"},
		},
		{
			"valid if no specific schemas requested (empty)",
			given{
				settingsOnEnvironment:     client.SchemaList{{"builtin:magic.setting"}},
				specificSettingsRequested: []string{},
			},
			true,
			nil,
		},
		{
			"valid if no specific schemas requested (nil)",
			given{
				settingsOnEnvironment:     client.SchemaList{{"builtin:magic.setting"}},
				specificSettingsRequested: nil,
			},
			true,
			nil,
		},
		{
			"valid if no specific schemas requested (empty) and none exist",
			given{
				settingsOnEnvironment:     client.SchemaList{},
				specificSettingsRequested: []string{},
			},
			true,
			nil,
		},
		{
			"valid if no specific schemas requested (nil) and none exist",
			given{
				settingsOnEnvironment:     client.SchemaList{},
				specificSettingsRequested: nil,
			},
			true,
			nil,
		},
		{
			"not valid if specific schemas requested but none exist",
			given{
				settingsOnEnvironment:     client.SchemaList{},
				specificSettingsRequested: []string{"builtin:magic.setting"},
			},
			false,
			[]string{"builtin:magic.setting"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockClient(gomock.NewController(t))
			c.EXPECT().ListSchemas().AnyTimes().Return(tt.given.settingsOnEnvironment, nil)

			gotValid, gotUnknownSchemas := validateSpecificSchemas(c, tt.given.specificSettingsRequested)
			assert.Equalf(t, tt.wantValid, gotValid, "validateSpecificSchemas(%v) for available settings %v", tt.given.specificSettingsRequested, tt.given.specificSettingsRequested)
			assert.Equalf(t, tt.wantUnknownSchemas, gotUnknownSchemas, "validateSpecificSchemas(%v) for available settings %v", tt.given.specificSettingsRequested, tt.given.specificSettingsRequested)
		})
	}
}

func TestDownloadConfigsExitsEarlyForUnknownAPI(t *testing.T) {
	c := client.NewMockClient(gomock.NewController(t))

	givenOpts := downloadConfigsOptions{
		specificAPIs:    []string{"UNKOWN API"},
		specificSchemas: nil,
		onlyAPIs:        false,
		onlySettings:    false,
		downloadOptionsShared: downloadOptionsShared{
			environmentURL: "testurl.com",
			auth: manifest.Auth{
				Token: manifest.AuthSecret{
					Name:  "TEST_TOKEN_VAR",
					Value: "test.token",
				},
			},
			outputFolder:            "folder",
			projectName:             "project",
			forceOverwriteManifest:  false,
			concurrentDownloadLimit: 1,
		},
	}

	givenDefaultAPIs := api.NewAPIs()
	err := doDownloadConfigs(afero.NewMemMapFs(), c, givenDefaultAPIs, givenOpts)
	assert.ErrorContains(t, err, "not known", "expected download to fail for unkown API")
}

func TestDownloadConfigsExitsEarlyForUnknownSettingsSchema(t *testing.T) {
	c := client.NewMockClient(gomock.NewController(t))

	givenOpts := downloadConfigsOptions{
		specificAPIs:    nil,
		specificSchemas: []string{"UNKOWN SCHEMA"},
		onlyAPIs:        false,
		onlySettings:    false,
		downloadOptionsShared: downloadOptionsShared{
			environmentURL: "testurl.com",
			auth: manifest.Auth{
				Token: manifest.AuthSecret{
					Name:  "TEST_TOKEN_VAR",
					Value: "test.token",
				},
			},
			outputFolder:            "folder",
			projectName:             "project",
			forceOverwriteManifest:  false,
			concurrentDownloadLimit: 1,
		},
	}

	c.EXPECT().ListSchemas().Return(client.SchemaList{{"builtin:some.schema"}}, nil)

	givenDefaultAPIs := api.NewAPIs()
	err := doDownloadConfigs(afero.NewMemMapFs(), c, givenDefaultAPIs, givenOpts)
	assert.ErrorContains(t, err, "not known", "expected download to fail for unkown Settings Schema")
	c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Times(0) // no downloads should even be attempted for unknown schema
}

func TestMapToAuth(t *testing.T) {
	t.Run("Best case scenario only with token", func(t *testing.T) {
		t.Setenv("TOKEN", "token_value")

		expected := &manifest.Auth{Token: manifest.AuthSecret{Name: "TOKEN", Value: "token_value"}}

		actual, errs := auth{token: "TOKEN"}.mapToAuth()

		assert.Empty(t, errs)
		assert.Equal(t, expected, actual)
	})
	t.Run("Best case scenario with OAuth", func(t *testing.T) {
		t.Setenv("TOKEN", "token_value")
		t.Setenv("CLIENT_ID", "client_id_value")
		t.Setenv("CLIENT_SECRET", "client_secret_value")

		expected := &manifest.Auth{
			Token: manifest.AuthSecret{Name: "TOKEN", Value: "token_value"},
			OAuth: &manifest.OAuth{
				ClientID:      manifest.AuthSecret{Name: "CLIENT_ID", Value: "client_id_value"},
				ClientSecret:  manifest.AuthSecret{Name: "CLIENT_SECRET", Value: "client_secret_value"},
				TokenEndpoint: nil,
			},
		}

		actual, errs := auth{
			token:        "TOKEN",
			clientID:     "CLIENT_ID",
			clientSecret: "CLIENT_SECRET",
		}.mapToAuth()

		assert.Empty(t, errs)
		assert.Equal(t, expected, actual)
	})
	t.Run("Token is missing", func(t *testing.T) {
		_, errs := auth{
			token: "TOKEN",
		}.mapToAuth()

		assert.Len(t, errs, 1)
		assert.Contains(t, errs, errors.New("the content of the environment variable \"TOKEN\" is not set"))
	})
	t.Run("Token is missing", func(t *testing.T) {
		_, errs := auth{
			token:        "TOKEN",
			clientID:     "CLIENT_ID",
			clientSecret: "CLIENT_SECRET",
		}.mapToAuth()

		assert.Len(t, errs, 3)
		assert.Contains(t, errs, errors.New("the content of the environment variable \"TOKEN\" is not set"))
		assert.Contains(t, errs, errors.New("the content of the environment variable \"CLIENT_ID\" is not set"))
		assert.Contains(t, errs, errors.New("the content of the environment variable \"CLIENT_SECRET\" is not set"))
	})
}
