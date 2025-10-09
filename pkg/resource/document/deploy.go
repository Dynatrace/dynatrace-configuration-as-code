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

package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

//go:generate mockgen -source=deploy.go -destination=document_mock.go -package=document DeploySource
type DeploySource interface {
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Create(ctx context.Context, name string, isPrivate bool, externalId string, data []byte, documentType documents.DocumentType) (api.Response, error)
	Update(ctx context.Context, id string, name string, isPrivate bool, data []byte, documentType documents.DocumentType) (api.Response, error)
}

var (
	ErrWrongPayloadType     = errors.New("can't deploy a Dynatrace classic dashboard using the 'documents' type. Either use 'api: dashboard' to deploy a Dynatrace classic dashboard or update your payload to a Dynatrace platform dashboard")
	ErrMissingNameParameter = errors.New("missing name parameter")
)

const errReadDataMsg = "error reading received data"

type DeployAPI struct {
	source DeploySource
}

func NewDeployAPI(source DeploySource) *DeployAPI {
	return &DeployAPI{source}
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	documentType, isPrivate, err := getDocumentAttributesFromConfigType(c.Type)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("cannot get document type for config: %w", err)
	}

	documentName, ok := properties[config.NameParameter].(string)
	if !ok {
		return entities.ResolvedEntity{}, ErrMissingNameParameter
	}

	if documentType == documents.Dashboard {
		if valErr := validateDashboardPayload(renderedConfig); valErr != nil {
			return entities.ResolvedEntity{}, valErr
		}
	}

	// strategy 1: if an origin id is available, try to update that document
	if c.OriginObjectId != "" {
		updateResponse, err := d.source.Update(ctx, c.OriginObjectId, documentName, isPrivate, []byte(renderedConfig), documentType)
		if err == nil {
			md, err := documents.UnmarshallMetadata(updateResponse.Data)
			if err != nil {
				return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, errReadDataMsg).WithError(err)
			}
			return createResolvedEntity(md.ID, c.Coordinate, properties), nil
		}

		if !api.IsNotFoundError(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}
	}

	// strategy 2: find and update document via external id
	externalId := idutils.GenerateExternalID(c.Coordinate)

	id, err := d.tryGetDocumentIDByExternalID(ctx, externalId)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error finding document with externalId='%s'", externalId)).WithError(err)
	}

	if id != "" {
		updateResponse, err := d.source.Update(ctx, id, documentName, isPrivate, []byte(renderedConfig), documentType)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", c.OriginObjectId)).WithError(err)
		}

		md, err := documents.UnmarshallMetadata(updateResponse.Data)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, errReadDataMsg).WithError(err)
		}
		return createResolvedEntity(md.ID, c.Coordinate, properties), nil
	}

	// strategy 3: try to create a new document
	createResponse, err := d.source.Create(ctx, documentName, isPrivate, externalId, []byte(renderedConfig), documentType)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to create document named '%s'", documentName)).WithError(err)
	}
	md, err := documents.UnmarshallMetadata(createResponse.Data)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, errReadDataMsg).WithError(err)
	}

	return createResolvedEntity(md.ID, c.Coordinate, properties), nil
}

func (d DeployAPI) tryGetDocumentIDByExternalID(ctx context.Context, externalId string) (string, error) {
	listResponse, err := d.source.List(ctx, fmt.Sprintf("externalId=='%s'", externalId))
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

func createResolvedEntity(id string, coordinate coordinate.Coordinate, properties parameter.Properties) entities.ResolvedEntity {
	properties[config.IdParameter] = id

	return entities.ResolvedEntity{
		Coordinate: coordinate,
		Properties: properties,
	}
}

func getDocumentAttributesFromConfigType(t config.Type) (docType string, private bool, err error) {
	documentType, ok := t.(config.DocumentType)
	if !ok {
		return "", false, fmt.Errorf("expected document config type but found %v", t)
	}

	docType, found := documentKindToType[documentType.Kind]
	if !found {
		return "", false, nil
	}

	return docType, documentType.Private, nil
}

// validateDashboardPayload returns an error if the JSON data is 1) malformed or 2) if the payload is not a Dynatrace platform dashboard payload.
func validateDashboardPayload(payload string) error {
	type DashboardKeys struct {
		Tiles any `json:"tiles"`
	}

	parsedPayload := DashboardKeys{}
	err := json.Unmarshal([]byte(payload), &parsedPayload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal dashboard payload: %w", err)
	}

	// Tiles should only be an array if Dynatrace classic dashboards configs are defined.
	// For Dynatrace platform dashboards, a map is used.
	if _, isArray := parsedPayload.Tiles.([]any); isArray {
		return ErrWrongPayloadType
	}

	return nil
}
