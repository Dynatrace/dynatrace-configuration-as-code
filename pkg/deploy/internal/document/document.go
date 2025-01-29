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

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	libAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

//go:generate mockgen -source=document.go -destination=document_mock.go -package=document documentClient
type Client interface {
	Get(ctx context.Context, id string) (documents.Response, error)
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Create(ctx context.Context, name string, isPrivate bool, externalId string, data []byte, documentType documents.DocumentType) (libAPI.Response, error)
	Update(ctx context.Context, id string, name string, isPrivate bool, data []byte, documentType documents.DocumentType) (libAPI.Response, error)
}

func Deploy(ctx context.Context, client Client, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	// create new context to carry logger
	ctx = logr.NewContextWithSlogLogger(ctx, log.WithCtxFields(ctx).SLogger())

	documentType, isPrivate, err := getDocumentAttributesFromConfigType(c.Type)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("cannot get document type for config: %w", err)
	}

	documentName, ok := properties[config.NameParameter].(string)
	if !ok {
		return entities.ResolvedEntity{}, errors.New("missing name parameter")
	}

	// strategy 1: if an origin id is available, try to update that document
	if c.OriginObjectId != "" {
		updateResponse, err := client.Update(ctx, c.OriginObjectId, documentName, isPrivate, []byte(renderedConfig), documentType)
		if err == nil {
			md, err := documents.UnmarshallMetadata(updateResponse.Data)
			if err != nil {
				return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "error reading received data").WithError(err)
			}
			return createResolvedEntity(documentName, md.ID, c.Coordinate, properties), nil
		}

		if !isAPIErrorStatusNotFound(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}
	}

	// strategy 2: find and update document via external id
	externalId, err := idutils.GenerateExternalIDForDocument(c.Coordinate)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "error generating external id").WithError(err)
	}

	id, err := tryGetDocumentIDByExternalID(ctx, client, externalId)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error finding document with externalId='%s'", externalId)).WithError(err)
	}

	if id != "" {
		updateResponse, err := client.Update(ctx, id, documentName, isPrivate, []byte(renderedConfig), documentType)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}

		md, err := documents.UnmarshallMetadata(updateResponse.Data)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "error reading received data").WithError(err)
		}
		return createResolvedEntity(documentName, md.ID, c.Coordinate, properties), nil
	}

	// strategy 3: try to create a new document
	createResponse, err := client.Create(ctx, documentName, isPrivate, externalId, []byte(renderedConfig), documentType)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to create document named '%s'", documentName)).WithError(err)
	}
	md, err := documents.UnmarshallMetadata(createResponse.Data)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "error reading received data").WithError(err)
	}

	return createResolvedEntity(documentName, md.ID, c.Coordinate, properties), nil
}

func isAPIErrorStatusNotFound(err error) bool {
	var apiErr api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.StatusCode == http.StatusNotFound
}

func tryGetDocumentIDByExternalID(ctx context.Context, client Client, externalId string) (string, error) {
	listResponse, err := client.List(ctx, fmt.Sprintf("externalId=='%s'", externalId))
	if err != nil {
		return "", err
	}

	//  it is an error if more than one document was found with the same external id: it should be unique
	if len(listResponse.Responses) > 1 {
		return "", fmt.Errorf("multiple documents found with externalId='%s'", externalId)
	}

	if len(listResponse.Responses) == 0 {
		return "", nil
	}

	return listResponse.Responses[0].ID, nil
}

func createResolvedEntity(documentName string, id string, coordinate coordinate.Coordinate, properties parameter.Properties) entities.ResolvedEntity {
	properties[config.IdParameter] = id

	return entities.ResolvedEntity{
		EntityName: documentName,
		Coordinate: coordinate,
		Properties: properties,
	}
}

var documentMapping = map[config.DocumentKind]documents.DocumentType{
	config.DashboardKind: documents.Dashboard,
	config.NotebookKind:  documents.Notebook,
	config.LaunchpadKind: documents.Launchpad,
}

func getDocumentAttributesFromConfigType(t config.Type) (doctype string, private bool, err error) {
	documentType, ok := t.(config.DocumentType)
	if !ok {
		return "", false, fmt.Errorf("expected document config type but found %v", t)
	}

	kind, f := documentMapping[documentType.Kind]
	if !f {
		return "", false, nil
	}

	return kind, documentType.Private, nil
}
