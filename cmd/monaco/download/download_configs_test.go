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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"net/http"
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
				environmentUrl:          "testurl.com",
				token:                   "test.token",
				tokenEnvVarName:         "TEST_TOKEN_VAR",
				outputFolder:            "folder",
				projectName:             "project",
				forceOverwriteManifest:  false,
				concurrentDownloadLimit: 1,
				clientProvider: func(h *http.Client, s string, f ...func(*client.DynatraceClient)) (client.Client, error) {
					return c, nil
				},
			}

			tt.expectedBehaviour(c)

			_, err := downloadConfigs(api.NewAPIs(), tt.givenOpts)
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
