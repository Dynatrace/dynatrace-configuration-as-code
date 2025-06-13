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

package document_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
)

const documentName = "my dashboard"

const documentIdForExternalId = "document-id-for-external-id"

var documentConfigCoordinate = coordinate.Coordinate{
	Project:  "proj",
	Type:     string(config.DashboardKind),
	ConfigId: "my-dashboard",
}
var defaultParameters = testutils.ToParameterMap([]parameter.NamedParameter{{
	Name: config.NameParameter,
	Parameter: &parameter.DummyParameter{
		Value: documentName,
	},
}})

func TestDeployDocumentWrongType(t *testing.T) {
	client := &client.DummyDocumentClient{}

	conf := &config.Config{
		Type:     config.ClassicApiType{},
		Template: testutils.GenerateFaultyTemplate(t),
	}

	_, errs := document.NewDeployAPI(client).Deploy(t.Context(), nil, "", conf)
	assert.NotEmpty(t, errs)
}

func TestDeploy_ConfigWithOriginObjectId(t *testing.T) {

	const originObjectId = "document-id"

	documentConfig := &config.Config{
		Type:           config.DocumentType{Kind: config.DashboardKind},
		Coordinate:     documentConfigCoordinate,
		OriginObjectId: originObjectId,
		Template:       testutils.GenerateDummyTemplate(t),
		Parameters:     defaultParameters,
	}

	expectedExternalId := idutils.GenerateExternalID(documentConfigCoordinate)
	expectedFilterString := fmt.Sprintf("externalId=='%s'", expectedExternalId)

	t.Run("Update by originObjectId succeeds", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Eq(documentName), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, originObjectId))}, nil)

		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, originObjectId, result.Properties[config.IdParameter])
	})

	t.Run("Update by originObjectId fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Eq(documentName), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("connection error"))

		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with originObjectId doesnt exist, list and update by externalId succeeds", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().
			Update(gomock.Any(), originObjectId, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
			Return(api.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().
			List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).
			Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().
			Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
			Return(api.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document with originObjectId doesnt exist, list by externalId fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with originObjectId doesnt exist, list by externalId succeeds, update fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with originObjectId doesnt exist, document with externalId doesnt exist, create succeeds", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document with originObjectId doesnt exist, document with externalId doesnt exist, create fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq(originObjectId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, api.APIError{StatusCode: http.StatusNotFound})
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})
}

func TestDeploy_ConfigWithoutOriginObjectId(t *testing.T) {

	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: defaultParameters,
	}

	expectedExternalId := idutils.GenerateExternalID(documentConfigCoordinate)

	expectedFilterString := fmt.Sprintf("externalId=='%s'", expectedExternalId)

	t.Run("Document list and update by externalId succeeds", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document list by externalId fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document list by externalId succeeds, update fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
		client.EXPECT().Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})

	t.Run("Document with externalId doesnt exist, create succeeds", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, documentIdForExternalId))}, nil)
		result, err := runDeployTest(t, client, documentConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
		assert.Equal(t, documentIdForExternalId, result.Properties[config.IdParameter])
	})

	t.Run("Document with externalId doesnt exist, create fails", func(t *testing.T) {
		client := document.NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().List(gomock.Any(), gomock.Eq(expectedFilterString)).Times(1).Return(documents.ListResponse{}, nil)
		client.EXPECT().Create(gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Eq(expectedExternalId), gomock.Any(), gomock.Any()).Times(1).Return(api.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, documentConfig)
		assert.Error(t, err)
	})
}

func TestDeploy_WithV1Payload_Fails(t *testing.T) {
	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   template.NewInMemoryTemplate("dashboard-v1", `{"tiles": []}`),
		Parameters: defaultParameters,
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))
	_, err := runDeployTest(t, cl, documentConfig)
	assert.ErrorIs(t, err, document.ErrWrongPayloadType)
}

func TestDeploy_WithMissingNameParameter_Fails(t *testing.T) {
	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))
	_, err := runDeployTest(t, cl, documentConfig)
	assert.ErrorIs(t, err, document.ErrMissingNameParameter)
}

func TestDeploy_WithInvalidUpdateResponse_Fails(t *testing.T) {
	documentConfig := &config.Config{
		OriginObjectId: "object-id",
		Type:           config.DocumentType{Kind: config.DashboardKind},
		Coordinate:     documentConfigCoordinate,
		Template:       testutils.GenerateDummyTemplate(t),
		Parameters:     defaultParameters,
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))
	cl.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(api.Response{Data: make([]byte, 0)}, nil)

	_, err := runDeployTest(t, cl, documentConfig)
	synErr := &json.SyntaxError{}
	assert.ErrorAs(t, err, &synErr)
}

func TestDeploy_WithInvalidUpdateResponseViaExternalID_Fails(t *testing.T) {
	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: defaultParameters,
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))
	cl.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Times(1).
		Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)
	cl.EXPECT().
		Update(gomock.Any(), gomock.Eq(documentIdForExternalId), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(api.Response{Data: make([]byte, 0)}, nil)

	_, err := runDeployTest(t, cl, documentConfig)
	synErr := &json.SyntaxError{}
	assert.ErrorAs(t, err, &synErr)
}

func TestDeploy_WithInvalidUpdateResponseViaCreate_Fails(t *testing.T) {
	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: defaultParameters,
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))
	cl.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Times(1).
		Return(documents.ListResponse{Responses: []documents.Response{}}, nil)

	cl.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(api.Response{Data: make([]byte, 0)}, nil)

	_, err := runDeployTest(t, cl, documentConfig)
	synErr := &json.SyntaxError{}
	assert.ErrorAs(t, err, &synErr)
}

func TestDeploy_WithDuplicateExternalID_Fails(t *testing.T) {
	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: defaultParameters,
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))
	cl.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Times(1).
		Return(documents.ListResponse{Responses: []documents.Response{{Metadata: documents.Metadata{ID: documentIdForExternalId}}, {Metadata: documents.Metadata{ID: documentIdForExternalId}}}}, nil)

	_, err := runDeployTest(t, cl, documentConfig)
	assert.ErrorContains(t, err, "multiple documents")
}

func TestDeploy_WithInvalidPayload_Fails(t *testing.T) {
	documentConfig := &config.Config{
		Type:       config.DocumentType{Kind: config.DashboardKind},
		Coordinate: documentConfigCoordinate,
		Template:   template.NewInMemoryTemplate("dashboard-v1", ""),
		Parameters: defaultParameters,
	}

	cl := document.NewMockDeploySource(gomock.NewController(t))

	parameters, errs := documentConfig.ResolveParameterValues(entities.New())
	require.Empty(t, errs)
	// in prod code an impossible case, as config.Render already checks if unmarshal works
	_, err := document.NewDeployAPI(cl).Deploy(t.Context(), parameters, "", documentConfig)
	synErr := &json.SyntaxError{}
	assert.ErrorAs(t, err, &synErr)
}

// not existing kind falls back to empty string
func TestDeploy_WithNotExistingDocumentKind_Succeeds(t *testing.T) {
	objectId := "object-id"
	documentConfig := &config.Config{
		Type:           config.DocumentType{Kind: "new-not-existing-one"},
		Coordinate:     documentConfigCoordinate,
		OriginObjectId: objectId,
		Template:       testutils.GenerateDummyTemplate(t),
		Parameters:     defaultParameters,
	}
	client := document.NewMockDeploySource(gomock.NewController(t))
	client.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(documentName), gomock.Any(), gomock.Any(), "").
		Times(1).
		Return(api.Response{Data: []byte(fmt.Sprintf(`{"id":"%s"}`, objectId))}, nil)

	_, err := runDeployTest(t, client, documentConfig)
	assert.NoError(t, err)
}

func runDeployTest(t *testing.T, client *document.MockDeploySource, c *config.Config) (entities.ResolvedEntity, error) {
	parameters, errs := c.ResolveParameterValues(entities.New())
	require.Empty(t, errs)
	content, err := c.Render(parameters)
	require.NoError(t, err)

	return document.NewDeployAPI(client).Deploy(t.Context(), parameters, content, c)
}
