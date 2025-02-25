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

package setting

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
)

func TestDeploySettingShouldFailCyclicParameterDependencies(t *testing.T) {
	ownerParameterName := "owner"
	configCoordinates := coordinate.Coordinate{}

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   configCoordinates,
						Property: ownerParameterName,
					},
				},
			},
		},
		{
			Name: ownerParameterName,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   configCoordinates,
						Property: config.NameParameter,
					},
				},
			},
		},
	}

	client := &dtclient.DummySettingsClient{}

	conf := &config.Config{
		Type:       config.ClassicApiType{},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
	}
	_, errors := Deploy(t.Context(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeploySettingShouldFailRenderTemplate(t *testing.T) {
	client := &dtclient.DummySettingsClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: testutils.GenerateFaultyTemplate(t),
	}

	_, errors := Deploy(t.Context(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeploySetting_ManagementZone_MZoneIDGetsEncoded(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
		Name: "mzname"}, nil)

	parameters := []parameter.NamedParameter{}

	conf := &config.Config{
		Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:management-zones", ConfigId: "abcde"},
		Type:       config.SettingsType{SchemaId: "builtin:management-zones", SchemaVersion: "1.2.3"},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
	}
	props := map[string]interface{}{"scope": "environment"}
	resolvedEntity, err := Deploy(t.Context(), c, props, "", conf)
	assert.Equal(t, entities.ResolvedEntity{
		EntityName: "[UNKNOWN NAME]vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
		Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:management-zones", ConfigId: "abcde"},
		Properties: map[string]any{"scope": "environment", "id": "-4292415658385853785", "name": "[UNKNOWN NAME]vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ"},
		Skip:       false,
	}, resolvedEntity)
	assert.NoError(t, err)
}

func TestDeploySetting_ManagementZone_NameGetsExtracted_ifPresent(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "abcdefghijk",
		Name: "mzname"}, nil)

	parameters := []parameter.NamedParameter{}

	conf := &config.Config{
		Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:some-setting", ConfigId: "abcde"},
		Type:       config.SettingsType{SchemaId: "builtin:management-zones", SchemaVersion: "1.2.3"},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
	}
	props := map[string]interface{}{"scope": "environment", "name": "the-name"}
	resolvedEntity, err := Deploy(t.Context(), c, props, "", conf)
	assert.Equal(t, entities.ResolvedEntity{
		EntityName: "the-name",
		Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:some-setting", ConfigId: "abcde"},
		Properties: map[string]any{"scope": "environment", "id": "abcdefghijk", "name": "the-name"},
		Skip:       false,
	}, resolvedEntity)
	assert.NoError(t, err)
}

func TestDeploySetting_ManagementZone_FailToDecodeMZoneID(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(dtclient.DynatraceEntity{
		Id:   "INVALID MANAGEMENT ZONE ID",
		Name: "mzanme"}, nil)

	parameters := []parameter.NamedParameter{}

	conf := &config.Config{
		Coordinate: coordinate.Coordinate{Project: "p", Type: "builtin:management-zones", ConfigId: "abcde"},
		Type:       config.SettingsType{SchemaId: "builtin:management-zones", SchemaVersion: "1.2.3"},
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap(parameters),
	}
	props := map[string]interface{}{"scope": "environment"}
	resolvedEntity, err := Deploy(t.Context(), c, props, "", conf)
	assert.Zero(t, resolvedEntity)
	assert.Error(t, err)
}

func TestDeploy_InsertAfter_NotDefined(t *testing.T) {

	t.Parallel()

	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().
		Upsert(t.Context(), gomock.Any(), dtclient.UpsertSettingsOptions{OverrideRetry: nil, InsertAfter: nil}).
		Times(1).
		Return(dtclient.DynatraceEntity{}, nil)

	conf := config.Config{
		Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
	}

	props := map[string]any{
		"scope": "environment",
	}

	_, err := Deploy(t.Context(), c, props, "{}", &conf)
	assert.NoError(t, err)

}
func TestDeploy_InsertAfter_ValidCases(t *testing.T) {
	tests := []struct {
		name                string
		insertAfterProperty string
		expectedInsertAfter string
	}{
		{
			name:                "an arbitrary ID is forwarded",
			insertAfterProperty: "ID-12345",
			expectedInsertAfter: "ID-12345",
		},
		{
			name:                "arbitrary ID most not be uppercased",
			insertAfterProperty: "id-12345",
			expectedInsertAfter: "id-12345",
		},
		{
			name:                "front is uppercased",
			insertAfterProperty: "front",
			expectedInsertAfter: dtclient.InsertPositionFront,
		},
		{
			name:                "back is uppercased",
			insertAfterProperty: "baCK",
			expectedInsertAfter: dtclient.InsertPositionBack,
		},
		{
			name:                "simple FRONT",
			insertAfterProperty: "FRONT",
			expectedInsertAfter: dtclient.InsertPositionFront,
		},
		{
			name:                "simple BACK",
			insertAfterProperty: "BACK",
			expectedInsertAfter: dtclient.InsertPositionBack,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := client.NewMockSettingsClient(gomock.NewController(t))
			c.EXPECT().
				Upsert(t.Context(), gomock.Any(), gomock.Eq(dtclient.UpsertSettingsOptions{OverrideRetry: nil, InsertAfter: &test.expectedInsertAfter})).
				Times(1).
				Return(dtclient.DynatraceEntity{}, nil)

			conf := config.Config{
				Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
			}

			props := map[string]any{
				"scope":       "environment",
				"insertAfter": test.insertAfterProperty,
			}

			_, err := Deploy(t.Context(), c, props, "{}", &conf)
			assert.NoError(t, err)
		})
	}
}

func TestDeploy_InsertAfter_InvalidCases(t *testing.T) {

	const errorTemplate = "'insertAfter' parameter must be a string of either an ID, 'FRONT', or 'BACK', got '%s'"

	tests := []struct {
		name                string
		insertAfterProperty any
		errorContains       string
	}{
		{
			name:                "empty array",
			insertAfterProperty: []string{},
			errorContains:       fmt.Sprintf(errorTemplate, "[]"),
		},
		{
			name:                "filled array",
			insertAfterProperty: []string{"test"},
			errorContains:       fmt.Sprintf(errorTemplate, "[test]"),
		},
		{
			name:                "map",
			insertAfterProperty: map[string]any{"test": "test"},
			errorContains:       fmt.Sprintf(errorTemplate, "map[test:test]"),
		},
		{
			name:                "object",
			insertAfterProperty: struct{ name string }{"test"},
			errorContains:       fmt.Sprintf(errorTemplate, "{test}"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := client.NewMockSettingsClient(gomock.NewController(t))

			conf := config.Config{
				Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
			}

			props := map[string]any{
				"scope":       "environment",
				"insertAfter": test.insertAfterProperty,
			}

			_, err := Deploy(t.Context(), c, props, "{}", &conf)
			assert.ErrorContains(t, err, test.errorContains)
		})
	}
}
