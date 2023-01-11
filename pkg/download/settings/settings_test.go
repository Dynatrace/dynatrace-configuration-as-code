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
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	v2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownload(t *testing.T) {
	uuid := util.GenerateUuidFromName("oid1")

	type mockValues struct {
		Schemas           func() (rest.SchemaList, error)
		ListSchemasCalls  int
		Settings          func() ([]rest.DownloadSettingsObject, error)
		ListSettingsCalls int
	}
	tests := []struct {
		name       string
		mockValues mockValues
		want       v2.ConfigsPerType
	}{
		{
			name: "DownloadSettings - List Schemas fails",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (rest.SchemaList, error) {
					return nil, fmt.Errorf("oh no")
				},
				Settings: func() ([]rest.DownloadSettingsObject, error) {
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
				Schemas: func() (rest.SchemaList, error) {
					return rest.SchemaList{{SchemaId: "id1"}, {SchemaId: "id2"}}, nil
				},
				Settings: func() ([]rest.DownloadSettingsObject, error) {
					return nil, fmt.Errorf("oh no")
				},
				ListSettingsCalls: 2,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name: "DownloadSettings",
			mockValues: mockValues{
				ListSchemasCalls: 1,
				Schemas: func() (rest.SchemaList, error) {
					return rest.SchemaList{{SchemaId: "id1"}}, nil
				},
				Settings: func() ([]rest.DownloadSettingsObject, error) {
					return []rest.DownloadSettingsObject{{
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
			want: v2.ConfigsPerType{"id1": {
				{
					Template: template.NewDownloadTemplate(uuid, uuid, string(json.RawMessage{})),
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
					Skip: false,
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := rest.NewMockDynatraceClient(gomock.NewController(t))
			schemas, err := tt.mockValues.Schemas()
			c.EXPECT().ListSchemas().Times(tt.mockValues.ListSchemasCalls).Return(schemas, err)
			settings, err := tt.mockValues.Settings()
			c.EXPECT().ListSettings(gomock.Any(), gomock.Any()).Times(tt.mockValues.ListSettingsCalls).Return(settings, err)
			res := Download(c, "projectName")
			assert.Equal(t, tt.want, res)
		})
	}
}
