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
	Create(ctx context.Context, name string, isPrivate bool, customId string, data []byte, documentType documents.DocumentType) (api.Response, error)
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
	documentType, err := getDocumentType(c.Type)
	if err != nil {
		return entities.ResolvedEntity{}, fmt.Errorf("cannot get document type: %w", err)
	}

	documentKind, isPrivate := getDocumentAttributesFromConfigType(documentType)
	documentName, ok := properties[config.NameParameter].(string)
	if !ok {
		return entities.ResolvedEntity{}, ErrMissingNameParameter
	}

	if documentKind == documents.Dashboard {
		if err := validateDashboardPayload(renderedConfig); err != nil {
			return entities.ResolvedEntity{}, err
		}
	}

	customID := resolveCustomID(documentType.CustomID, c.Coordinate)
	response, err := d.upsertDocument(ctx, c, documentName, isPrivate, customID, []byte(renderedConfig), documentKind)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	return handleResponse(response, c, properties)
}

func resolveCustomID(configCustomID string, coord coordinate.Coordinate) string {
	// customID is either the one defined in the config, or a generated external ID
	if configCustomID != "" {
		return configCustomID
	}
	return idutils.GenerateExternalID(coord)
}

func (d DeployAPI) upsertDocument(ctx context.Context, c *config.Config, name string, isPrivate bool, customID string, payload []byte, kind documents.DocumentType) (api.Response, error) {
	// We try both IDs in order: originObjectId (UUID) first, then customID
	idsToTry := []string{c.OriginObjectId, customID}
	for _, id := range idsToTry {
		if id == "" {
			continue
		}
		response, err := d.source.Update(ctx, id, name, isPrivate, payload, kind)
		if err == nil {
			return response, nil
		}
		if !api.IsNotFoundError(err) {
			return api.Response{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update document '%s'", id)).WithError(err)
		}
	}

	// If not found, create a new document with custom ID or generated external ID
	response, err := d.source.Create(ctx, name, isPrivate, customID, payload, kind)
	if err != nil {
		return api.Response{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to create document named '%s'", name)).WithError(err)
	}
	return response, nil
}

func handleResponse(response api.Response, c *config.Config, properties parameter.Properties) (entities.ResolvedEntity, error) {
	md, err := documents.UnmarshallMetadata(response.Data)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, errReadDataMsg).WithError(err)
	}
	return createResolvedEntity(md.ID, c.Coordinate, properties), nil
}

func createResolvedEntity(id string, coordinate coordinate.Coordinate, properties parameter.Properties) entities.ResolvedEntity {
	properties[config.IdParameter] = id

	return entities.ResolvedEntity{
		Coordinate: coordinate,
		Properties: properties,
	}
}

func getDocumentType(t config.Type) (config.DocumentType, error) {
	documentType, ok := t.(config.DocumentType)
	if !ok {
		return config.DocumentType{}, fmt.Errorf("expected document config type but found %v", t)
	}

	return documentType, nil
}

func getDocumentAttributesFromConfigType(documentType config.DocumentType) (docType string, private bool) {
	docType, found := documentKindToType[documentType.Kind]
	if !found {
		return "", false
	}

	return docType, documentType.Private
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
