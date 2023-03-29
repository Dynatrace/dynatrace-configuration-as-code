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

package entities

import (
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
	testType := "SOMETHING"
	testType2 := "SOMETHINGELSE"
	uuid := idutils.GenerateUuidFromName(testType)

	type mockValues struct {
		EntitiesTypeList      func() ([]client.EntitiesType, error)
		EntitiesTypeListCalls int
		EntitiesList          func() (client.EntitiesList, error)
		EntitiesListCalls     int
	}
	tests := []struct {
		name       string
		mockValues mockValues
		want       v2.ConfigsPerType
	}{
		{
			name: "DownloadEntities - List Entity Types fails",
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, error) {
					return nil, client.RespError{Err: fmt.Errorf("oh no"), StatusCode: 0}
				},
				EntitiesTypeListCalls: 1,
				EntitiesList: func() (client.EntitiesList, error) {
					return client.EntitiesList{}, nil
				},
				EntitiesListCalls: 0,
			},
			want: nil,
		},
		{
			name: "DownloadEntities - List Entity fails",
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, error) {
					return []client.EntitiesType{{EntitiesTypeId: testType}, {EntitiesTypeId: testType2}}, nil
				},
				EntitiesTypeListCalls: 1,
				EntitiesList: func() (client.EntitiesList, error) {
					return client.EntitiesList{}, client.RespError{Err: fmt.Errorf("oh no"), StatusCode: 0}
				},
				EntitiesListCalls: 2,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name: "DownloadEntities",
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, error) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, nil
				},
				EntitiesTypeListCalls: 1,
				EntitiesList: func() (client.EntitiesList, error) {
					return client.EntitiesList{
						Entities: []string{""},
					}, nil
				},
				EntitiesListCalls: 1,
			},
			want: v2.ConfigsPerType{testType: {
				{
					Template: template.NewDownloadTemplate(testType, testType, "[]"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     testType,
						ConfigId: uuid,
					},
					Type: config.EntityType{
						EntitiesType: testType,
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: uuid},
					},
					Skip: false,
				},
			}},
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockClient(gomock.NewController(t))
			entityTypeList, err := tt.mockValues.EntitiesTypeList()
			c.EXPECT().ListEntitiesTypes().Times(tt.mockValues.EntitiesTypeListCalls).Return(entityTypeList, err)
			entities, err := tt.mockValues.EntitiesList()
			c.EXPECT().ListEntities(gomock.Any()).Times(tt.mockValues.EntitiesListCalls).Return(entities, err)
			res := NewEntitiesDownloader(c).DownloadAll("projectName")
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestDownload(t *testing.T) {
	testType := "SOMETHING"
	uuid := idutils.GenerateUuidFromName(testType)

	type mockValues struct {
		EntitiesTypeList      func() ([]client.EntitiesType, error)
		EntitiesTypeListCalls int
		EntitiesList          func() (client.EntitiesList, error)
		EntitiesListCalls     int
	}
	tests := []struct {
		name          string
		EntitiesTypes []string
		mockValues    mockValues
		want          v2.ConfigsPerType
	}{
		{
			name: "DownloadEntities - empty list of entities types",
			mockValues: mockValues{
				EntitiesTypeList:      func() ([]client.EntitiesType, error) { return []client.EntitiesType{}, nil },
				EntitiesTypeListCalls: 0,
				EntitiesList:          func() (client.EntitiesList, error) { return client.EntitiesList{}, nil },
				EntitiesListCalls:     0,
			},
			want: nil,
		},
		{
			name:          "DownloadEntities - entities list empty",
			EntitiesTypes: []string{testType},
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, error) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, nil
				},
				EntitiesTypeListCalls: 1,
				EntitiesList: func() (client.EntitiesList, error) {
					return client.EntitiesList{
						From:     "",
						To:       "",
						Entities: make([]string, 0, 1),
					}, nil
				},
				EntitiesListCalls: 1,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name:          "DownloadEntities - Not all entities found",
			EntitiesTypes: []string{testType, "SOMETHING_ELSE"},
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, error) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, nil
				},
				EntitiesTypeListCalls: 1,
				EntitiesList: func() (client.EntitiesList, error) {
					return client.EntitiesList{
						From:     "",
						To:       "",
						Entities: []string{""},
					}, nil
				},
				EntitiesListCalls: 0,
			},
			want: nil,
		},
		{
			name:          "DownloadEntities - entities found",
			EntitiesTypes: []string{testType},
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, error) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, nil
				},
				EntitiesTypeListCalls: 1,
				EntitiesList: func() (client.EntitiesList, error) {
					return client.EntitiesList{
						From:     "",
						To:       "",
						Entities: []string{""},
					}, nil
				},
				EntitiesListCalls: 1,
			},
			want: v2.ConfigsPerType{testType: {
				{
					Template: template.NewDownloadTemplate(testType, testType, "[]"),
					Coordinate: coordinate.Coordinate{
						Project:  "projectName",
						Type:     testType,
						ConfigId: uuid,
					},
					Type: config.EntityType{
						EntitiesType: testType,
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: uuid},
					},
					Skip: false,
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMockClient(gomock.NewController(t))
			entityTypeList, err := tt.mockValues.EntitiesTypeList()
			c.EXPECT().ListEntitiesTypes().Times(tt.mockValues.EntitiesTypeListCalls).Return(entityTypeList, err)
			entities, err := tt.mockValues.EntitiesList()
			c.EXPECT().ListEntities(gomock.Any()).Times(tt.mockValues.EntitiesListCalls).Return(entities, err)
			res := NewEntitiesDownloader(c).Download(tt.EntitiesTypes, "projectName")
			assert.Equal(t, tt.want, res)
		})
	}
}
