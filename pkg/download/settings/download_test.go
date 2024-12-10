//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package settings

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

func TestDownloadAll(t *testing.T) {
	uuid1 := idutils.GenerateUUIDFromString("oid1")
	uuid2 := idutils.GenerateUUIDFromString("oid2")
	uuid3 := idutils.GenerateUUIDFromString("oid3")

	type mockValues struct {
		Schemas           func() (dtclient.SchemaList, error)
		ListSchemasCalls  int
		Settings          func() ([]dtclient.DownloadSettingsObject, error)
		ListSettingsCalls int
		GetSchema         func(schemaID string) (dtclient.Schema, error)
		GetSchemaCalls    int
	}
	tests := []struct {
		name       string
		mockValues mockValues
		filters    map[string]Filter
		want       v2.ConfigsPerType
	}{
		{
			name: "DownloadSettings - List Schemas fails",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return nil, fmt.Errorf("oh no")
				},
				GetSchema:      func(schemaID string) (dtclient.Schema, error) { return dtclient.Schema{}, nil },
				GetSchemaCalls: 0,

				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return nil, nil
				},
				ListSettingsCalls: 0,
			},
			want: nil,
		},
		{
			name: "DownloadSettings - List Settings fails",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1"}, {SchemaId: "id2"}}, nil
				},
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return nil, coreapi.APIError{StatusCode: 0}
				},
				ListSettingsCalls: 2,
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{}, nil
				},
				GetSchemaCalls: 2,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name: "DownloadSettings - invalid (empty) value payload",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1"}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1"}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "sv1",
						SchemaId:      "sid1",
						ObjectId:      "oid1",
						Scope:         "tenant",
						Value:         json.RawMessage{},
					}}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {}},
		},
		{
			name: "DownloadSettings - valid value payload",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1"}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1"}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "sv1",
						SchemaId:      "sid1",
						ObjectId:      "oid1",
						Scope:         "tenant",
						Value:         json.RawMessage("{}"),
					}}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {
				{
					Template: template.NewInMemoryTemplate(uuid1, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid1,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
			}},
		},
		{
			name: "DownloadSettings - discard settings based on filter",
			filters: map[string]Filter{"sid1": {ShouldDiscard: func(settingsValue map[string]interface{}) (bool, string) {
				return settingsValue["skip"] == true, "skip is true"
			}},
			},
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1"}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1"}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "sv1",
						SchemaId:      "sid1",
						ObjectId:      "oid1",
						Scope:         "tenant",
						Value:         json.RawMessage(`{"skip" : true}`),
					}}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {}},
		},
		{
			name: "DownloadSettings - discard unmodifable settings",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1"}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1"}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{
						{
							ExternalId:    "ex1",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid1",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
						},
						{
							ExternalId:    "ex2",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid2",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: false,
							},
						},
					}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {
				{
					Template: template.NewInMemoryTemplate(uuid1, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid1,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
			}},
		},
		{
			name: "DownloadSettings - don't discard unmodifable settings that have single modifiable properties",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1"}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1"}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{
						{
							ExternalId:    "ex1",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid1",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable:      false,
								ModifiablePaths: []string{"enabled"},
							},
						},
					}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {
				{
					Template: template.NewInMemoryTemplate(uuid1, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid1,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
			}},
		},
		{
			name: "DownloadSettings - ordered settings with discard should not panic",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1", Ordered: true}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1", Ordered: true}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{
						{
							ExternalId:    "ex1",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid1",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: true,
								Movable:    true,
							},
						},
						{
							ExternalId:    "ex2",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid2",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: false,
								Movable:    true,
							},
						},
						{
							ExternalId:    "ex3",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid3",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: true,
								Movable:    true,
							},
						},
					}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {
				{
					Template: template.NewInMemoryTemplate(uuid1, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid1,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
				{
					Template: template.NewInMemoryTemplate(uuid3, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid3,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
						config.InsertAfterParameter: &reference.ReferenceParameter{
							ParameterReference: parameter.ParameterReference{
								Config: coordinate.Coordinate{
									Project:  "projectName",
									Type:     "sid1",
									ConfigId: uuid1,
								},
								Property: "id",
							},
						},
					},
					Skip:           false,
					OriginObjectId: "oid3",
				},
			}},
		},
		{
			name: "DownloadSettings - non-movable ordered settings should not receive insertAfter param",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "id1", Ordered: true}}, nil
				},
				GetSchema: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "id1", Ordered: true}, nil
				},
				GetSchemaCalls: 1,
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{
						{
							ExternalId:    "ex1",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid1",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: true,
								Movable:    true,
							},
						},
						{
							ExternalId:    "ex2",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid2",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: true,
								Movable:    true,
							},
						},
						{
							ExternalId:    "ex3",
							SchemaVersion: "sv1",
							SchemaId:      "sid1",
							ObjectId:      "oid3",
							Scope:         "tenant",
							Value:         json.RawMessage("{}"),
							ModificationInfo: &dtclient.SettingsModificationInfo{
								Modifiable: true,
								Movable:    false,
							},
						},
					}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"id1": {
				{
					Template: template.NewInMemoryTemplate(uuid1, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid1,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
				{
					Template: template.NewInMemoryTemplate(uuid2, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid2,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
						config.InsertAfterParameter: &reference.ReferenceParameter{
							ParameterReference: parameter.ParameterReference{
								Config: coordinate.Coordinate{
									Project:  "projectName",
									Type:     "sid1",
									ConfigId: uuid1,
								},
								Property: "id",
							},
						},
					},
					Skip:           false,
					OriginObjectId: "oid2",
				},
				{
					Template: template.NewInMemoryTemplate(uuid3, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid3,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid3",
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockSettingsClient(gomock.NewController(t))
			schemas, err := tt.mockValues.Schemas()
			c.EXPECT().ListSchemas(gomock.Any()).Times(tt.mockValues.ListSchemasCalls).Return(schemas, err)
			//c.EXPECT().GetSchemaById(gomock.Any()).Times(tt.mockValues.GetSchemaCalls).Return(tt.mockValues.GetSchema(""))
			settings, err := tt.mockValues.Settings()
			c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Times(tt.mockValues.ListSettingsCalls).Return(settings, err)
			res, _ := Download(c, "projectName", tt.filters)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestDownload(t *testing.T) {
	uuid := idutils.GenerateUUIDFromString("oid1")

	type mockValues struct {
		Schemas        func() (dtclient.SchemaList, error)
		Settings       func() ([]dtclient.DownloadSettingsObject, error)
		FetchedSchemas func(schemaID string) (dtclient.Schema, error)

		ListSchemasCalls  int
		ListSettingsCalls int
		GetSchemaCalls    int

		EnvVars map[string]string
	}
	tests := []struct {
		name       string
		Schemas    []config.SettingsType
		mockValues mockValues
		want       v2.ConfigsPerType
	}{
		{
			name: "DownloadSettings - empty list of schemas",
			mockValues: mockValues{
				Schemas: func() (dtclient.SchemaList, error) { return dtclient.SchemaList{}, nil },
				FetchedSchemas: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{}, nil
				},
				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{}, nil
				},
				ListSchemasCalls:  1,
				GetSchemaCalls:    0,
				ListSettingsCalls: 0,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name:    "DownloadSettings - Settings found",
			Schemas: []config.SettingsType{{SchemaId: "builtin:alerting-profile"}},
			mockValues: mockValues{
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "builtin:alerting-profile"}}, nil
				},
				FetchedSchemas: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "builtin:alerting-profile"}, nil
				},
				GetSchemaCalls: 1,

				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "sv1",
						SchemaId:      "sid1",
						ObjectId:      "oid1",
						Scope:         "tenant",
						Value:         json.RawMessage(`{}`),
					}}, nil
				},
				ListSchemasCalls:  1,
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"builtin:alerting-profile": {
				{
					Template: template.NewInMemoryTemplate(uuid, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid,
					},
					Type: config.SettingsType{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
			}},
		},
		{
			name:    "Downloading builtin:host.monitoring.mode discards all by default",
			Schemas: []config.SettingsType{{SchemaId: "builtin:host.monitoring.mode"}},
			mockValues: mockValues{
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "builtin:host.monitoring.mode"}}, nil
				},
				FetchedSchemas: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "builtin:host.monitoring.mode"}, nil
				},
				GetSchemaCalls: 1,

				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "1.2.3",
						SchemaId:      "builtin:host.monitoring.mode",
						ObjectId:      "oid1",
						Scope:         "HOST-1234567890ABCDEF",
						Value:         json.RawMessage(`{}`),
					}}, nil
				},
				ListSchemasCalls:  1,
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"builtin:host.monitoring.mode": {}},
		},
		{
			name:    "Downloading builtin:host.monitoring.mode does not discard them if the DownloadFilter FF is inactive",
			Schemas: []config.SettingsType{{SchemaId: "builtin:host.monitoring.mode"}},
			mockValues: mockValues{
				Schemas: func() (dtclient.SchemaList, error) {
					return dtclient.SchemaList{{SchemaId: "builtin:host.monitoring.mode"}}, nil
				},
				FetchedSchemas: func(schemaID string) (dtclient.Schema, error) {
					return dtclient.Schema{SchemaId: "builtin:host.monitoring.mode"}, nil
				},
				GetSchemaCalls: 1,
				EnvVars: map[string]string{
					featureflags.DownloadFilter: "false",
				},

				Settings: func() ([]dtclient.DownloadSettingsObject, error) {
					return []dtclient.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "1.2.3",
						SchemaId:      "builtin:host.monitoring.mode",
						ObjectId:      "oid1",
						Scope:         "HOST-1234567890ABCDEF",
						Value:         json.RawMessage(`{}`),
					}}, nil
				},
				ListSchemasCalls:  1,
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"builtin:host.monitoring.mode": {
				{
					Template: template.NewInMemoryTemplate(uuid, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "builtin:host.monitoring.mode",
						ConfigId: uuid,
					},
					Type: config.SettingsType{
						SchemaId:      "builtin:host.monitoring.mode",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "HOST-1234567890ABCDEF"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.mockValues.EnvVars {
				t.Setenv(k, v)
			}

			c := client.NewMockSettingsClient(gomock.NewController(t))
			schemas, err1 := tt.mockValues.Schemas()
			settings, err2 := tt.mockValues.Settings()
			c.EXPECT().ListSchemas(gomock.Any()).Times(tt.mockValues.ListSchemasCalls).Return(schemas, err1)
			c.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Times(tt.mockValues.ListSettingsCalls).Return(settings, err2)
			res, _ := Download(c, "projectName", DefaultSettingsFilters, tt.Schemas...)
			assert.Equal(t, tt.want, res)
		})
	}
}

func Test_validateSpecificSettings(t *testing.T) {
	type given struct {
		settingsOnEnvironment     []schema
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
				settingsOnEnvironment:     []schema{{id: "builtin:magic.setting"}},
				specificSettingsRequested: []string{"builtin:magic.setting"},
			},
			true,
			nil,
		},
		{
			"not valid if setting not found",
			given{
				settingsOnEnvironment:     []schema{{id: "builtin:magic.setting"}},
				specificSettingsRequested: []string{"builtin:unknown"},
			},
			false,
			[]string{"builtin:unknown"},
		},
		{
			"not valid if one setting not found",
			given{
				settingsOnEnvironment:     []schema{{id: "builtin:magic.setting"}},
				specificSettingsRequested: []string{"builtin:magic.setting", "builtin:unknown"},
			},
			false,
			[]string{"builtin:unknown"},
		},
		{
			"valid if no specific schemas requested (empty)",
			given{
				settingsOnEnvironment:     []schema{{id: "builtin:magic.setting"}},
				specificSettingsRequested: []string{},
			},
			true,
			nil,
		},
		{
			"valid if no specific schemas requested (nil)",
			given{
				settingsOnEnvironment:     []schema{{id: "builtin:magic.setting"}},
				specificSettingsRequested: nil,
			},
			true,
			nil,
		},
		{
			"valid if no specific schemas requested (empty) and none exist",
			given{
				settingsOnEnvironment:     []schema{},
				specificSettingsRequested: []string{},
			},
			true,
			nil,
		},
		{
			"valid if no specific schemas requested (nil) and none exist",
			given{
				settingsOnEnvironment:     []schema{},
				specificSettingsRequested: nil,
			},
			true,
			nil,
		},
		{
			"not valid if specific schemas requested but none exist",
			given{
				settingsOnEnvironment:     []schema{},
				specificSettingsRequested: []string{"builtin:magic.setting"},
			},
			false,
			[]string{"builtin:magic.setting"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, gotUnknownSchemas := validateSpecificSchemas(tt.given.settingsOnEnvironment, tt.given.specificSettingsRequested)
			assert.Equalf(t, tt.wantValid, gotValid, "validateSpecificSchemas(%v) for available settings %v", tt.given.specificSettingsRequested, tt.given.specificSettingsRequested)
			assert.Equalf(t, tt.wantUnknownSchemas, gotUnknownSchemas, "validateSpecificSchemas(%v) for available settings %v", tt.given.specificSettingsRequested, tt.given.specificSettingsRequested)
		})
	}
}

func Test_shouldFilterUnmodifiableSettings(t *testing.T) {
	type flags struct {
		downloadFilterFF                     bool
		downloadFilterSettingsFF             bool
		downloadFilterSettingsUnmodifiableFF bool
	}
	tests := []struct {
		name  string
		given flags
		want  bool
	}{
		{
			name: "applies filter if all feature flags are active",
			given: flags{
				downloadFilterFF:                     true,
				downloadFilterSettingsFF:             true,
				downloadFilterSettingsUnmodifiableFF: true,
			},
			want: true,
		},
		{
			name: "does not apply filters if base flag is OFF",
			given: flags{
				downloadFilterFF:                     false,
				downloadFilterSettingsFF:             true,
				downloadFilterSettingsUnmodifiableFF: true,
			},
			want: false,
		},
		{
			name: "does not apply filters if settings flag is OFF",
			given: flags{
				downloadFilterFF:                     true,
				downloadFilterSettingsFF:             false,
				downloadFilterSettingsUnmodifiableFF: true,
			},
			want: false,
		},
		{
			name: "does not apply filters if unmodifiable settings flag is OFF",
			given: flags{
				downloadFilterFF:                     true,
				downloadFilterSettingsFF:             true,
				downloadFilterSettingsUnmodifiableFF: false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN Feature Flags
			t.Setenv(featureflags.Permanent[featureflags.DownloadFilter].EnvName(), strconv.FormatBool(tt.given.downloadFilterFF))
			t.Setenv(featureflags.Permanent[featureflags.DownloadFilterSettings].EnvName(), strconv.FormatBool(tt.given.downloadFilterSettingsFF))
			t.Setenv(featureflags.Permanent[featureflags.DownloadFilterSettingsUnmodifiable].EnvName(), strconv.FormatBool(tt.given.downloadFilterSettingsUnmodifiableFF))

			assert.Equalf(t, tt.want, shouldFilterUnmodifiableSettings(), "shouldFilterUnmodifableSettings()")
		})
	}
}

func Test_shouldFilterSettings(t *testing.T) {
	type flags struct {
		downloadFilterFF         bool
		downloadFilterSettingsFF bool
	}
	tests := []struct {
		name  string
		given flags
		want  bool
	}{
		{
			name: "applies filter if all feature flags are active",
			given: flags{
				downloadFilterFF:         true,
				downloadFilterSettingsFF: true,
			},
			want: true,
		},
		{
			name: "does not apply filters if base flag is OFF",
			given: flags{
				downloadFilterFF:         false,
				downloadFilterSettingsFF: true,
			},
			want: false,
		},
		{
			name: "does not apply filters if settings flag is OFF",
			given: flags{
				downloadFilterFF:         true,
				downloadFilterSettingsFF: false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN Feature Flags
			t.Setenv(featureflags.Permanent[featureflags.DownloadFilter].EnvName(), strconv.FormatBool(tt.given.downloadFilterFF))
			t.Setenv(featureflags.Permanent[featureflags.DownloadFilterSettings].EnvName(), strconv.FormatBool(tt.given.downloadFilterSettingsFF))

			assert.Equalf(t, tt.want, shouldFilterSettings(), "shouldFilterSettings()")
		})
	}
}
