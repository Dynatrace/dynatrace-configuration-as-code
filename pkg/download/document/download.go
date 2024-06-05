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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

func Download(client client.DocumentClient, projectName string) (v2.ConfigsPerType, error) {
	result := make(v2.ConfigsPerType)

	dashboards := downloadDocumentsOfType(client, projectName, documents.Dashboard)
	notebooks := downloadDocumentsOfType(client, projectName, documents.Notebook)
	result[string(config.DocumentTypeId)] = append(result[string(config.DocumentTypeId)], dashboards...)
	result[string(config.DocumentTypeId)] = append(result[string(config.DocumentTypeId)], notebooks...)
	return result, nil
}

func downloadDocumentsOfType(client client.DocumentClient, projectName string, documentType documents.DocumentType) []config.Config {
	listResponse, err := client.List(context.TODO(), fmt.Sprintf("type=='%s'", documentType))
	if err != nil {
		log.WithFields(field.Type("document"), field.Error(err)).Error("Failed to list all documents of type '%s': %v", documentType, err)
		return nil
	}

	var configs []config.Config
	for _, response := range listResponse.Responses {

		config, err := convertDocumentResponse(client, projectName, response)
		if err != nil {
			log.WithFields(field.Type("document"), field.Error(err)).Error("Failed to convert document '%s' of type '%s': %v", response.ID, documentType, err)
			continue
		}
		configs = append(configs, config)
	}

	return configs
}

func convertDocumentResponse(client client.DocumentClient, projectName string, response documents.Response) (config.Config, error) {
	documentType, err := validateDocumentType(response.Type)
	if err != nil {
		return config.Config{}, err
	}

	documentType.Private = response.IsPrivate

	documentResponse, err := client.Get(context.TODO(), response.ID)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to get document: %w", err)
	}

	params := map[string]parameter.Parameter{
		config.NameParameter: &value.ValueParameter{Value: documentResponse.Name},
	}

	template, err := createTemplateFromResponse(documentResponse)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to create template: %w", err)
	}

	return config.Config{
		Template: template,
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.DocumentTypeId),
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
	switch documentType {
	case string(documents.Dashboard):
		return config.DocumentType{Kind: config.DashboardKind}, nil
	case string(documents.Notebook):
		return config.DocumentType{Kind: config.NotebookKind}, nil
	default:
		return config.DocumentType{}, fmt.Errorf("unsupported document type: %s", documentType)
	}
}
