//go:build unit

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package settings_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/settings"
)

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
	resolvedEntity, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "", conf)
	assert.Equal(t, entities.ResolvedEntity{
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
	resolvedEntity, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "", conf)
	assert.Equal(t, entities.ResolvedEntity{
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
	resolvedEntity, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "", conf)
	assert.Zero(t, resolvedEntity)
	assert.Error(t, err)
}

func TestDeploy_InsertAfter_NotDefined(t *testing.T) {

	t.Parallel()

	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().
		Upsert(context.TODO(), gomock.Any(), dtclient.UpsertSettingsOptions{OverrideRetry: nil, InsertAfter: nil}).
		Times(1).
		Return(dtclient.DynatraceEntity{}, nil)

	conf := config.Config{
		Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
	}

	props := map[string]any{
		"scope": "environment",
	}

	_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "{}", &conf)
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
				Upsert(context.TODO(), gomock.Any(), gomock.Eq(dtclient.UpsertSettingsOptions{OverrideRetry: nil, InsertAfter: &test.expectedInsertAfter})).
				Times(1).
				Return(dtclient.DynatraceEntity{}, nil)

			conf := config.Config{
				Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
			}

			props := map[string]any{
				"scope":       "environment",
				"insertAfter": test.insertAfterProperty,
			}

			_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "{}", &conf)
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

			_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "{}", &conf)
			assert.ErrorContains(t, err, test.errorContains)
		})
	}
}

func TestDeploy_FailsWithInvalidConfigType(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))

	conf := config.Config{
		Type: config.ClassicApiType{},
	}

	_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), nil, "{}", &conf)
	assert.ErrorContains(t, err, "config was not of expected type")
}

func TestDeploy_FailsWithoutScope(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))

	conf := config.Config{
		Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
	}

	_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), nil, "{}", &conf)
	assert.ErrorContains(t, err, fmt.Sprintf("'%s' not found", config.ScopeParameter))
}

func TestDeploy_WithBucketRetrySetting(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().
		Upsert(context.TODO(), gomock.Any(), gomock.Eq(dtclient.UpsertSettingsOptions{OverrideRetry: &dtclient.RetrySetting{WaitTime: 10 * time.Second, MaxRetries: 12}})).
		Times(1).
		Return(dtclient.DynatraceEntity{}, nil)

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config: coordinate.Coordinate{
							Type: string(config.BucketTypeID),
						},
						Property: "name",
					},
				},
			},
		},
	}

	conf := config.Config{
		Type:       config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
		Parameters: testutils.ToParameterMap(parameters),
	}
	props := map[string]any{
		"scope": "environment",
	}

	_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "{}", &conf)
	assert.NoError(t, err)
}

func TestDeploy_WithVeryLongRetrySetting(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))
	c.EXPECT().
		Upsert(context.TODO(), gomock.Any(), gomock.Eq(dtclient.UpsertSettingsOptions{OverrideRetry: &dtclient.DefaultRetrySettings.VeryLong})).
		Times(1).
		Return(dtclient.DynatraceEntity{}, nil)

	parameters := []parameter.NamedParameter{
		{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config: coordinate.Coordinate{
							Type: api.ApplicationWeb,
						},
						Property: "name",
					},
				},
			},
		},
	}

	conf := config.Config{
		Type:       config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
		Parameters: testutils.ToParameterMap(parameters),
	}
	props := map[string]any{
		"scope": "environment",
	}

	_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "{}", &conf)
	assert.NoError(t, err)
}

func TestDeploy_WithFailedRequest(t *testing.T) {
	c := client.NewMockSettingsClient(gomock.NewController(t))
	wantErrMsg := "custom error"
	customErr := errors.New(wantErrMsg)
	c.EXPECT().
		Upsert(context.TODO(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(dtclient.DynatraceEntity{}, customErr)

	conf := config.Config{
		Type: config.SettingsType{SchemaId: "builtin:monaco-test", SchemaVersion: "1.2.3"},
	}
	props := map[string]any{
		"scope": "environment",
	}

	_, err := settings.NewDeployAPI(c).Deploy(context.TODO(), props, "{}", &conf)
	assert.ErrorContains(t, err, wantErrMsg)
}
