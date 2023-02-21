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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownloadAll(t *testing.T) {
	uuid := idutils.GenerateUuidFromName("oid1")

	type mockValues struct {
		Schemas           func() (client.SchemaList, error)
		ListSchemasCalls  int
		Settings          func() ([]client.DownloadSettingsObject, error)
		ListSettingsCalls int
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
				Schemas: func() (client.SchemaList, error) {
					return nil, fmt.Errorf("oh no")
				},
				Settings: func() ([]client.DownloadSettingsObject, error) {
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
				Schemas: func() (client.SchemaList, error) {
					return client.SchemaList{{SchemaId: "id1"}, {SchemaId: "id2"}}, nil
				},
				Settings: func() ([]client.DownloadSettingsObject, error) {
					return nil, fmt.Errorf("oh no")
				},
				ListSettingsCalls: 2,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name: "DownloadSettings - invalid (empty) value payload",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (client.SchemaList, error) {
					return client.SchemaList{{SchemaId: "id1"}}, nil
				},
				Settings: func() ([]client.DownloadSettingsObject, error) {
					return []client.DownloadSettingsObject{{
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
				Schemas: func() (client.SchemaList, error) {
					return client.SchemaList{{SchemaId: "id1"}}, nil
				},
				Settings: func() ([]client.DownloadSettingsObject, error) {
					return []client.DownloadSettingsObject{{
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
					Template: template.NewDownloadTemplate(uuid, uuid, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid,
					},
					Type: config.Type{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter:  &value.ValueParameter{Value: uuid},
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
				Schemas: func() (client.SchemaList, error) {
					return client.SchemaList{{SchemaId: "id1"}}, nil
				},
				Settings: func() ([]client.DownloadSettingsObject, error) {
					return []client.DownloadSettingsObject{{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockClient(gomock.NewController(t))
			schemas, err := tt.mockValues.Schemas()
			c.EXPECT().ListSchemas().Times(tt.mockValues.ListSchemasCalls).Return(schemas, err)
			settings, err := tt.mockValues.Settings()
			c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Times(tt.mockValues.ListSettingsCalls).Return(settings, err)
			res := NewSettingsDownloader(c, WithFilters(tt.filters)).DownloadAll("projectName")
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestDownload(t *testing.T) {
	uuid := idutils.GenerateUuidFromName("oid1")

	type mockValues struct {
		Schemas           func() (client.SchemaList, error)
		Settings          func() ([]client.DownloadSettingsObject, error)
		ListSettingsCalls int
	}
	tests := []struct {
		name       string
		Schemas    []string
		mockValues mockValues
		want       v2.ConfigsPerType
	}{
		{
			name: "DownloadSettings - empty list of schemas",
			mockValues: mockValues{
				Schemas:           func() (client.SchemaList, error) { return client.SchemaList{}, nil },
				Settings:          func() ([]client.DownloadSettingsObject, error) { return []client.DownloadSettingsObject{}, nil },
				ListSettingsCalls: 0,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name:    "DownloadSettings - Settings found",
			Schemas: []string{"builtin:alerting-profile"},
			mockValues: mockValues{
				Schemas: func() (client.SchemaList, error) {
					return client.SchemaList{{SchemaId: "id1"}}, nil
				},
				Settings: func() ([]client.DownloadSettingsObject, error) {
					return []client.DownloadSettingsObject{{
						ExternalId:    "ex1",
						SchemaVersion: "sv1",
						SchemaId:      "sid1",
						ObjectId:      "oid1",
						Scope:         "tenant",
						Value:         json.RawMessage(`{}`),
					}}, nil
				},
				ListSettingsCalls: 1,
			},
			want: v2.ConfigsPerType{"builtin:alerting-profile": {
				{
					Template: template.NewDownloadTemplate(uuid, uuid, "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     "sid1",
						ConfigId: uuid,
					},
					Type: config.Type{
						SchemaId:      "sid1",
						SchemaVersion: "sv1",
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter:  &value.ValueParameter{Value: uuid},
						config.ScopeParameter: &value.ValueParameter{Value: "tenant"},
					},
					Skip:           false,
					OriginObjectId: "oid1",
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockClient(gomock.NewController(t))
			settings, err := tt.mockValues.Settings()
			c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Times(tt.mockValues.ListSettingsCalls).Return(settings, err)
			res := NewSettingsDownloader(c).Download(tt.Schemas, "projectName")
			assert.Equal(t, tt.want, res)
		})
	}
}
