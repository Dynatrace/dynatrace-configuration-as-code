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

package document

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

var documentMapping = map[string]config.DocumentKind{
	documents.Dashboard: config.DashboardKind,
	documents.Notebook:  config.NotebookKind,
	documents.Launchpad: config.LaunchpadKind,
}

type Source interface {
	List(ctx context.Context, filter string) (documents.ListResponse, error)
	Get(ctx context.Context, id string) (documents.Response, error)
}

type DownloadAPI struct {
	documentSource Source
}

func NewDownloadAPI(documentSource Source) *DownloadAPI {
	return &DownloadAPI{documentSource}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.Info("Downloading documents")
	// due to the current test setup, the types must be downloaded in order. This should be changed eventually
	var typesToDownload = []documents.DocumentType{
		documents.Dashboard,
		documents.Notebook,
		documents.Launchpad,
	}

	var allConfigs []config.Config
	for _, docKind := range typesToDownload {
		configs := downloadDocumentsOfType(ctx, a.documentSource, projectName, docKind)
		allConfigs = append(allConfigs, configs...)
	}

	return project.ConfigsPerType{
		string(config.DocumentTypeID): allConfigs,
	}, nil
}

func downloadDocumentsOfType(ctx context.Context, documentSource Source, projectName string, documentType string) []config.Config {
	log.WithFields(field.Type("document")).Debug("Downloading documents of type '%s'", documentType)

	listResponse, err := documentSource.List(ctx, fmt.Sprintf("type=='%s'", documentType))
	if err != nil {
		log.WithFields(field.Type("document"), field.Error(err)).Error("Failed to list all documents of type '%s': %v", documentType, err)
		return nil
	}

	var configs []config.Config

	for _, response := range listResponse.Responses {
		// skip downloading ready-made documents - these are presets that cannot be redeployed
		if isReadyMadeByAnApp(response.Metadata) {
			continue
		}

		cfg, err := convertDocumentResponse(ctx, documentSource, projectName, response)
		if err != nil {
			log.WithFields(field.Type("document"), field.Error(err)).Error("Failed to convert document '%s' of type '%s': %v", response.ID, documentType, err)
			continue
		}
		configs = append(configs, cfg)
	}

	log.WithFields(field.Type("document")).Debug("Downloaded %d documents of type '%s'", len(configs), documentType)

	return configs
}

func isReadyMadeByAnApp(metadata documents.Metadata) bool {
	return (metadata.OriginAppID != nil) && (len(*metadata.OriginAppID) > 0)
}

func convertDocumentResponse(ctx context.Context, documentSource Source, projectName string, response documents.Response) (config.Config, error) {
	documentType, err := validateDocumentType(response.Type)
	if err != nil {
		return config.Config{}, err
	}

	documentType.Private = response.IsPrivate

	documentResponse, err := documentSource.Get(ctx, response.ID)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to get document: %w", err)
	}

	params := map[string]parameter.Parameter{
		config.NameParameter: &value.ValueParameter{Value: documentResponse.Name},
	}

	templateFromResponse, err := createTemplateFromResponse(documentResponse)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to create template: %w", err)
	}

	return config.Config{
		Template: templateFromResponse,
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.DocumentTypeID),
			ConfigId: documentResponse.ID,
		},
		Type:           documentType,
		Parameters:     params,
		OriginObjectId: documentResponse.ID,
	}, nil
}

func createTemplateFromResponse(response documents.Response) (template.Template, error) {
	var data map[string]interface{}
	err := json.Unmarshal(response.Data, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return template.NewInMemoryTemplate(response.ID, string(bytes)), nil
}

func validateDocumentType(documentType string) (config.DocumentType, error) {
	kind, f := documentMapping[documentType]
	if !f {
		return config.DocumentType{}, fmt.Errorf("unsupported document type: %s", documentType)
	}

	return config.DocumentType{Kind: kind}, nil
}
