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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	projectv2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestDownloadConfigsBehaviour(t *testing.T) {
	tests := []struct {
		name              string
		givenOpts         downloadConfigsOptions
		expectedBehaviour func(client *client.MockDynatraceClient)
	}{
		{
			name: "Default opts: downloads Configs and Settings",
			givenOpts: downloadConfigsOptions{
				specificAPIs:    nil,
				specificSchemas: nil,
				onlyAPIs:        false,
				onlySettings:    false,
			},
			expectedBehaviour: func(c *client.MockDynatraceClient) {
				c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).AnyTimes().Return([]dtclient.Value{}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("{}"), nil) // singleton configs are always attempted
				c.EXPECT().ListSchemas().Return(dtclient.SchemaList{}, nil)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]dtclient.DownloadSettingsObject{}, nil)
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
			expectedBehaviour: func(c *client.MockDynatraceClient) {
				c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).Times(0)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Times(0)
				c.EXPECT().ListSchemas().AnyTimes().Return(dtclient.SchemaList{{SchemaId: "builtin:magic.secret"}}, nil)
				c.EXPECT().GetSchemaById(gomock.Any()).AnyTimes().Return(dtclient.Schema{SchemaId: "builtin:magic.secret"}, nil)
				c.EXPECT().ListSettings(gomock.Any(), "builtin:magic.secret", gomock.Any()).AnyTimes().Return([]dtclient.DownloadSettingsObject{}, nil)
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
			expectedBehaviour: func(c *client.MockDynatraceClient) {
				c.EXPECT().ListConfigs(gomock.Any(), api.NewAPIs()["alerting-profile"]).Return([]dtclient.Value{{Id: "42", Name: "profile"}}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), "42").AnyTimes().Return([]byte("{}"), nil)
				c.EXPECT().ListSchemas().Times(0)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			expectedBehaviour: func(c *client.MockDynatraceClient) {
				c.EXPECT().ListConfigs(gomock.Any(), api.NewAPIs()["alerting-profile"]).Return([]dtclient.Value{{Id: "42", Name: "profile"}}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), "42").AnyTimes().Return([]byte("{}"), nil)
				c.EXPECT().ListSchemas().AnyTimes().Return(dtclient.SchemaList{{SchemaId: "builtin:magic.secret"}}, nil)
				c.EXPECT().GetSchemaById(gomock.Any()).AnyTimes().Return(dtclient.Schema{SchemaId: "builtin:magic.secret"}, nil)
				c.EXPECT().ListSettings(gomock.Any(), "builtin:magic.secret", gomock.Any()).AnyTimes().Return([]dtclient.DownloadSettingsObject{}, nil)

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
			expectedBehaviour: func(c *client.MockDynatraceClient) {
				c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).AnyTimes().Return([]dtclient.Value{}, nil)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("{}"), nil) // singleton configs are always attempted
				c.EXPECT().ListSchemas().Times(0)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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
			expectedBehaviour: func(c *client.MockDynatraceClient) {
				c.EXPECT().ListConfigs(gomock.Any(), gomock.Any()).Times(0)
				c.EXPECT().ReadConfigById(gomock.Any(), gomock.Any()).Times(0)
				c.EXPECT().ListSchemas().Return(dtclient.SchemaList{}, nil)
				c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]dtclient.DownloadSettingsObject{}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockDynatraceClient(gomock.NewController(t))

			tt.givenOpts.downloadOptionsShared = downloadOptionsShared{
				environmentURL: "testurl.com",
				auth: manifest.Auth{
					Token: manifest.AuthSecret{
						Name:  "TEST_TOKEN_VAR",
						Value: "test.token",
					},
				},
				outputFolder:           "folder",
				projectName:            "project",
				forceOverwriteManifest: false,
			}

			tt.expectedBehaviour(c)

			_, err := downloadConfigs(&client.ClientSet{DTClient: c}, api.NewAPIs(), tt.givenOpts, defaultDownloadFn)
			assert.NoError(t, err)
		})
	}
}

func TestDownload_Options(t *testing.T) {
	type wantDownload struct {
		config, settings, bucket, automation, document, openpipeline bool
	}
	tests := []struct {
		name  string
		given downloadConfigsOptions
		want  wantDownload
	}{
		{
			"download all if options are not limiting",
			downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{
					auth: manifest.Auth{OAuth: &manifest.OAuth{}}, // OAuth required to be defined for platform types
				},
			},
			wantDownload{
				config:       true,
				settings:     true,
				bucket:       true,
				automation:   true,
				document:     true,
				openpipeline: true,
			},
		},
		{
			"only settings requested",
			downloadConfigsOptions{onlySettings: true},
			wantDownload{settings: true},
		},
		{
			"specific settings requested",
			downloadConfigsOptions{specificSchemas: []string{"some:schema"}},
			wantDownload{settings: true},
		},
		{
			"only documents requested",
			downloadConfigsOptions{onlyDocuments: true,
				downloadOptionsShared: downloadOptionsShared{
					auth: manifest.Auth{OAuth: &manifest.OAuth{}},
				}},
			wantDownload{document: true},
		},
		{
			"only openpipeline requested",
			downloadConfigsOptions{onlyOpenPipeline: true,
				downloadOptionsShared: downloadOptionsShared{
					auth: manifest.Auth{OAuth: &manifest.OAuth{}},
				}},
			wantDownload{openpipeline: true},
		},
		{
			"only apis requested",
			downloadConfigsOptions{onlyAPIs: true},
			wantDownload{config: true},
		},
		{
			"specific config apis requested",
			downloadConfigsOptions{specificAPIs: []string{"alerting-profile"}},
			wantDownload{config: true},
		},
		{
			"only automations requested",
			downloadConfigsOptions{
				downloadOptionsShared: downloadOptionsShared{
					auth: manifest.Auth{OAuth: &manifest.OAuth{}},
				},
				onlyAutomation: true,
			},
			wantDownload{automation: true},
		},
		{
			"specific APIs and schemas",
			downloadConfigsOptions{specificAPIs: []string{"alerting-profile"}, specificSchemas: []string{"some:schema"}},
			wantDownload{config: true, settings: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := downloadFn{
				classicDownload: func(client.ConfigClient, string, api.APIs, classic.ContentFilters) (projectv2.ConfigsPerType, error) {
					if !tt.want.config {
						t.Fatalf("classic config download was not meant to be called but was")
					}
					return nil, nil
				},
				settingsDownload: func(settingsClient client.SettingsClient, s string, filters settings.Filters, settingsType ...config.SettingsType) (projectv2.ConfigsPerType, error) {
					if !tt.want.settings {
						t.Fatalf("settings download was not meant to be called but was")
					}
					return nil, nil
				},
				automationDownload: func(a client.AutomationClient, s string, automationType ...config.AutomationType) (projectv2.ConfigsPerType, error) {
					if !tt.want.automation {
						t.Fatalf("automation download was not meant to be called but was")
					}
					return nil, nil
				},
				bucketDownload: func(b client.BucketClient, s string) (projectv2.ConfigsPerType, error) {
					if !tt.want.bucket {
						t.Fatalf("automation download was not meant to be called but was")
					}
					return nil, nil
				},
				documentDownload: func(b client.DocumentClient, s string) (projectv2.ConfigsPerType, error) {
					if !tt.want.document {
						t.Fatalf("document download was not meant to be called but was")
					}
					return nil, nil
				},
				openPipelineDownload: func(b client.OpenPipelineClient, s string) (projectv2.ConfigsPerType, error) {
					if !tt.want.openpipeline {
						t.Fatalf("openpipeline download was not meant to be called but was")
					}
					return nil, nil
				},
			}

			_, err := downloadConfigs(&client.ClientSet{DTClient: client.NewMockDynatraceClient(gomock.NewController(t))}, api.NewAPIs(), tt.given, fn)
			assert.NoError(t, err)
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

func TestDownloadConfigsExitsEarlyForUnknownSettingsSchema(t *testing.T) {
	c := client.NewMockDynatraceClient(gomock.NewController(t))

	givenOpts := downloadConfigsOptions{
		specificSchemas: []string{"UNKOWN SCHEMA"},
		onlySettings:    false,
		downloadOptionsShared: downloadOptionsShared{
			environmentURL: "testurl.com",
			auth: manifest.Auth{
				Token: manifest.AuthSecret{
					Name:  "TEST_TOKEN_VAR",
					Value: "test.token",
				},
			},
			outputFolder:           "folder",
			projectName:            "project",
			forceOverwriteManifest: false,
		},
	}

	c.EXPECT().ListSchemas().Return(dtclient.SchemaList{{"builtin:some.schema"}}, nil)

	err := doDownloadConfigs(afero.NewMemMapFs(), &client.ClientSet{DTClient: c}, nil, givenOpts)
	assert.ErrorContains(t, err, "not known", "expected download to fail for unkown Settings Schema")
	c.EXPECT().ListSettings(gomock.Any(), gomock.Any(), gomock.Any()).Times(0) // no downloads should even be attempted for unknown schema
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

func TestDownloadConfigs_OnlyAutomationWithoutAutomationCredentials(t *testing.T) {
	opts := downloadConfigsOptions{
		onlyAutomation: true,
	}

	err := doDownloadConfigs(testutils.CreateTestFileSystem(), &client.ClientSet{}, nil, opts)
	assert.ErrorContains(t, err, "no OAuth credentials configured")
}

func Test_downloadConfigsOptions_valid(t *testing.T) {
	t.Run("no error for konwn api", func(t *testing.T) {
		given := downloadConfigsOptions{specificAPIs: []string{"alerting-profile"}}

		errs := given.valid()

		assert.Len(t, errs, 0)
	})
	t.Run("report error for unknown", func(t *testing.T) {
		given := downloadConfigsOptions{specificAPIs: []string{"unknown api"}}

		errs := given.valid()

		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs[0], "unknown api")
	})
}

func Test_copyConfigs(t *testing.T) {
	t.Run("Copy configs to empty", func(t *testing.T) {
		dest := projectv2.ConfigsPerType{}
		copyConfigs(dest, projectv2.ConfigsPerType{
			"dashboard": []config.Config{
				{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}}},
			"notebook": []config.Config{
				{Coordinate: coordinate.Coordinate{ConfigId: "notebook-1"}}},
		})

		assert.Len(t, dest, 2)

		assert.Contains(t, dest, "notebook")
		assert.EqualValues(t, dest["notebook"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "notebook-1"}}})

		assert.Contains(t, dest, "dashboard")
		assert.EqualValues(t, dest["dashboard"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}}})
	})

	t.Run("Copying configs of same type should merge", func(t *testing.T) {
		dest := projectv2.ConfigsPerType{"dashboard": []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}},
		}}
		copyConfigs(dest, projectv2.ConfigsPerType{"dashboard": []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-2"}},
		}})

		assert.Len(t, dest, 1)

		assert.Contains(t, dest, "dashboard")
		assert.EqualValues(t, dest["dashboard"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}},
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-2"}},
		})
	})

	t.Run("Copy configs of different types", func(t *testing.T) {
		dest := projectv2.ConfigsPerType{"notebook": []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "notebook-1"}},
		}}
		copyConfigs(dest, projectv2.ConfigsPerType{"dashboard": []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}}}})

		assert.Len(t, dest, 2)

		assert.Contains(t, dest, "notebook")
		assert.EqualValues(t, dest["notebook"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "notebook-1"}},
		})

		assert.Contains(t, dest, "dashboard")
		assert.EqualValues(t, dest["dashboard"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}},
		})
	})

	t.Run("Merge configs of same and different types", func(t *testing.T) {
		dest := projectv2.ConfigsPerType{
			"notebook": []config.Config{
				{Coordinate: coordinate.Coordinate{ConfigId: "notebook-1"}},
			},
			"dashboard": []config.Config{
				{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}},
			},
		}

		copyConfigs(dest, projectv2.ConfigsPerType{"dashboard": []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-2"}},
		}})

		assert.Len(t, dest, 2)

		assert.Contains(t, dest, "notebook")
		assert.EqualValues(t, dest["notebook"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "notebook-1"}},
		})

		assert.Contains(t, dest, "dashboard")
		assert.EqualValues(t, dest["dashboard"], []config.Config{
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-1"}},
			{Coordinate: coordinate.Coordinate{ConfigId: "dashboard-2"}},
		})
	})
}
