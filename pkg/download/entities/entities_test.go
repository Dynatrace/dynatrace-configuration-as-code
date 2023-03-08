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
	respError "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/errors"
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
		EntitiesTypeList      func() ([]client.EntitiesType, respError.RespError)
		EntitiesTypeListCalls int
		EntitiesList          func() ([]string, respError.RespError)
		EntitiesListCalls     int
	}
	tests := []struct {
		name       string
		mockValues mockValues
		want       v2.ConfigsPerType
	}{
		{
			name: "DownloadEntities - List Entities Types fails",
			mockValues: mockValues{
				EntitiesTypeListCalls: 1,
				EntitiesTypeList: func() ([]client.EntitiesType, respError.RespError) {
					return nil, respError.RespError{WrappedError: fmt.Errorf("oh no"), StatusCode: 0}
				},
				EntitiesList: func() ([]string, respError.RespError) {
					return nil, respError.RespError{}
				},
				EntitiesListCalls: 0,
			},
			want: nil,
		},
		{
			name: "DownloadEntities - List Entities fails",
			mockValues: mockValues{
				EntitiesTypeListCalls: 1,
				EntitiesTypeList: func() ([]client.EntitiesType, respError.RespError) {
					return []client.EntitiesType{{EntitiesTypeId: testType}, {EntitiesTypeId: testType2}}, respError.RespError{}
				},
				EntitiesList: func() ([]string, respError.RespError) {
					return nil, respError.RespError{WrappedError: fmt.Errorf("oh no"), StatusCode: 0}
				},
				EntitiesListCalls: 2,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name: "DownloadEntities",
			mockValues: mockValues{
				EntitiesTypeListCalls: 1,
				EntitiesTypeList: func() ([]client.EntitiesType, respError.RespError) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, respError.RespError{}
				},
				EntitiesList: func() ([]string, respError.RespError) {
					return []string{""}, respError.RespError{}
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
					Type: config.Type{
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
		EntitiesTypeList  func() ([]client.EntitiesType, respError.RespError)
		EntitiesList      func() ([]string, respError.RespError)
		EntitiesListCalls int
	}
	tests := []struct {
		name          string
		EntitiesTypes []client.EntitiesType
		mockValues    mockValues
		want          v2.ConfigsPerType
	}{
		{
			name: "DownloadEntities - empty list of entities types",
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, respError.RespError) {
					return []client.EntitiesType{}, respError.RespError{}
				},
				EntitiesList:      func() ([]string, respError.RespError) { return []string{}, respError.RespError{} },
				EntitiesListCalls: 0,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name:          "DownloadEntities - entities list empty",
			EntitiesTypes: []client.EntitiesType{{EntitiesTypeId: testType}},
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, respError.RespError) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, respError.RespError{}
				},
				EntitiesList: func() ([]string, respError.RespError) {
					return make([]string, 0, 1), respError.RespError{}
				},
				EntitiesListCalls: 1,
			},
			want: v2.ConfigsPerType{},
		},
		{
			name:          "DownloadEntities - entities found",
			EntitiesTypes: []client.EntitiesType{{EntitiesTypeId: testType}},
			mockValues: mockValues{
				EntitiesTypeList: func() ([]client.EntitiesType, respError.RespError) {
					return []client.EntitiesType{{EntitiesTypeId: testType}}, respError.RespError{}
				},
				EntitiesList: func() ([]string, respError.RespError) {
					return []string{""}, respError.RespError{}
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
					Type: config.Type{
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
			entities, err := tt.mockValues.EntitiesList()
			c.EXPECT().ListEntities(gomock.Any()).Times(tt.mockValues.EntitiesListCalls).Return(entities, err)
			res := NewEntitiesDownloader(c).Download(tt.EntitiesTypes, "projectName")
			assert.Equal(t, tt.want, res)
		})
	}
}
