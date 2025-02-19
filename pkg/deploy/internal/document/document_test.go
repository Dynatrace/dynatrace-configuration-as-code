//go:build unit

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package document

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
)

const documentName = "my dashboard"

const documentIdForExternalId = "document-id-for-external-id"

var documentConfigCoordinate = coordinate.Coordinate{
	Project:  "proj",
	Type:     string(config.DashboardKind),
	ConfigId: "my-dashboard",
}

func TestDeployDocumentWrongType(t *testing.T) {
	client := &client.DryRunDocumentClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: testutils.GenerateFaultyTemplate(t),
	}

	_, errors := Deploy(context.TODO(), client, nil, "", conf)
	assert.NotEmpty(t, errors)
}

func TestDeploy_ConfigWithOriginObjectId(t *testing.T) {

	const originObjectId = "document-id"

	documentConfig := &config.Config{
		Type:           config.DocumentType{Kind: config.DashboardKind},
		Coordinate:     documentConfigCoordinate,
		OriginObjectId: originObjectId,
		Template:       testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap([]parameter.NamedParameter{{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: documentName,
			},
		}}),
	}

	expectedExternalId, err := idutils.GenerateExternalIDForDocument(documentConfigCoordinate)
	require.NoError(t, err)

	expectedFilterString := fmt.Sprintf("externalId=='%s'", expectedExternalId)

	t.Run("Update by originObjectId succeeds", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Eq(documentName), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, originObjectId))}, nil)

		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, originObjectId, result.Properties[config.IdParameter])
	})

	t.Run("Update by originObjectId fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Eq(documentName), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, errors.New("connection error"))

		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with originObjectId doesnt exist, list and update by externalId succeeds", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().
			Update(gomock.Any(), originObjectId, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
			Return(libAPI.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().
			List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).
			Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().
			Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
			Return(libAPI.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document with originObjectId doesnt exist, list by externalId fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with originObjectId doesnt exist, list by externalId succeeds, update fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with originObjectId doesnt exist, document with externalId doesnt exist, create succeeds", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document with originObjectId doesnt exist, document with externalId doesnt exist, create fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})
}

func TestDeploy_ConfigWithoutOriginObjectId(t *testing.T) {

	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: testutils.ToParameterMap([]parameter.NamedParameter{{
			Name: config.NameParameter,
			Parameter: &parameter.DummyParameter{
				Value: documentName,
			},
		}}),
	}

	expectedExternalId, err := idutils.GenerateExternalIDForDocument(documentConfigCoordinate)
	require.NoError(t, err)

	expectedFilterString := fmt.Sprintf("externalId=='%s'", expectedExternalId)

	t.Run("Document list and update by externalId succeeds", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document list by externalId fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document list by externalId succeeds, update fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with externalId doesnt exist, create succeeds", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document with externalId doesnt exist, create fails", func(t *testing.T) {
		client := NewMockClient(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(libAPI.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})
}

func runDeployTest(t *testing.T, client Client, c *config.Config) (entities.ResolvedEntity, error) {
	parameters, errs := c.ResolveParameterValues(entities.New())
	require.Empty(t, errs)

	return Deploy(context.TODO(), client, parameters, "", c)
}
