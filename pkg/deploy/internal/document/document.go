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

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/go-logr/logr"
)

//go:generate mockgen -source=document.go -destination=document_mock.go -package=document documentClient
type Client interface {
	Get(ctx context.Context, id string) (documents.Response, error)
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Create(ctx context.Context, name string, externalId string, data []byte, documentType documents.DocumentType) (documents.Response, error)
	Update(ctx context.Context, id string, name string, data []byte, documentType documents.DocumentType) (documents.Response, error)
}

var _ Client = (*DummyClient)(nil)

type DummyClient struct{}

// Create implements Client.
func (c DummyClient) Create(ctx context.Context, name string, externalId string, data []byte, documentType documents.DocumentType) (documents.Response, error) {
	return documents.Response{}, nil
}

// Get implements Client.
func (c *DummyClient) Get(ctx context.Context, id string) (documents.Response, error) {
	return documents.Response{}, nil
}

// List implements Client.
func (c *DummyClient) List(ctx context.Context, filter string) (documents.ListResponse, error) {
	return documents.ListResponse{}, nil
}

// Update implements Client.
func (c *DummyClient) Update(ctx context.Context, id string, name string, data []byte, documentType documents.DocumentType) (documents.Response, error) {
	return documents.Response{}, nil
}

func Deploy(ctx context.Context, client Client, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	// create new context to carry logger
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())

	documentName, ok := properties[config.NameParameter].(string)
	if !ok {
		return entities.ResolvedEntity{}, errors.New("missing name parameter")
	}

	documentType, err := getDocumentTypeFromConfigType(c.Type)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("cannot get document type for config: %w", err)
	}

	// strategy 1: if an origin id is available, try to update that document
	if c.OriginObjectId != "" {
		_, err := client.Update(ctx, c.OriginObjectId, documentName, []byte(renderedConfig), documentType)
		if err == nil {
			properties[config.IdParameter] = c.OriginObjectId

			return entities.ResolvedEntity{
				EntityName: documentName,
				Coordinate: c.Coordinate,
				Properties: properties,
			}, nil
		}

		// error status not found means other deployment strategies should be tried, all other errors should stop deployment
		var apiErr api.APIError
		if !errors.As(err, &apiErr) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}

		if apiErr.StatusCode != http.StatusNotFound {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}
	}

	// strategy 2: find and update document via external id
	externalId, err := idutils.GenerateExternalIDForDocument(c.Coordinate)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "error generating external id").WithError(err)
	}

	// look for a document with the external id
	response, err := client.List(ctx, fmt.Sprintf("externalId=='%s'", externalId))
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error listing documents with externalId='%s'", externalId)).WithError(err)
	}

	//  it is an error if more than one document was found with the same external id (this should not happen as external id should be unique)
	if len(response.Responses) > 1 {
		return entities.ResolvedEntity{}, fmt.Errorf("multiple documents found with externalId='%s'", externalId)
	}

	// try to update the document if just one was found
	if len(response.Responses) == 1 {
		_, err := client.Update(ctx, response.Responses[0].ID, documentName, []byte(renderedConfig), documentType)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}

		properties[config.IdParameter] = response.Responses[0].ID

		return entities.ResolvedEntity{
			EntityName: documentName,
			Coordinate: c.Coordinate,
			Properties: properties,
		}, nil
	}

	// strategy 3: try to create a new document
	createResponse, err := client.Create(ctx, documentName, externalId, []byte(renderedConfig), documentType)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to create document named '%s'", documentName)).WithError(err)
	}

	properties[config.IdParameter] = createResponse.ID

	return entities.ResolvedEntity{
		EntityName: documentName,
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}

func getDocumentTypeFromConfigType(t config.Type) (documents.DocumentType, error) {
	documentType, ok := t.(config.DocumentType)
	if !ok {
		return "", fmt.Errorf("expected document config type but found %v", t)
	}
	switch documentType {
	case config.DashboardType:
		return documents.Dashboard, nil
	case config.NotebookType:
		return documents.Notebook, nil
	}

	return "", nil
}
