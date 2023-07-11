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

package dtclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient/internal/response"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"net/http"
	"net/url"
)

type (
	SchemaList []struct {
		SchemaId string `json:"schemaId"`
	}

	// SchemaListResponse is the response type returned by the ListSchemas operation
	SchemaListResponse struct {
		Items      SchemaList `json:"items"`
		TotalCount int        `json:"totalCount"`
	}

	// Schema represents the definition of a specific settings schema
	Schema struct {
		ID               string
		UniqueProperties [][]string
	}

	// SchemaDetail is the response type returned by the schemaDetails operation
	schemaDetailsResponse struct {
		SchemaId          string `json:"schemaId"`
		SchemaConstraints []struct {
			Type             string   `json:"type"`
			UniqueProperties []string `json:"uniqueProperties"`
		} `json:"schemaConstraints"`
	}
)

func (d *DynatraceClient) ListSchemas() (schemas SchemaList, err error) {
	d.limiter.ExecuteBlocking(func() {
		schemas, err = d.listSchemas(context.TODO())
	})
	return
}

func (d *DynatraceClient) listSchemas(ctx context.Context) (SchemaList, error) {
	u, err := url.Parse(d.environmentURL + d.settingsSchemaAPIPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// getting all schemas does not have pagination
	resp, err := rest.Get(ctx, d.client, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to GET schemas: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, rest.NewRespErr(fmt.Sprintf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodGet, u.String())
	}

	var result SchemaListResponse
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, rest.NewRespErr("failed to unmarshal response", resp).WithRequestInfo(http.MethodGet, u.String()).WithErr(err)
	}

	if result.TotalCount != len(result.Items) {
		log.Warn("Total count of settings 2.0 schemas (=%d) does not match with count of actually downloaded settings 2.0 schemas (=%d)", result.TotalCount, len(result.Items))
	}

	return result.Items, nil
}

func (d *DynatraceClient) Schema(schemaID string) (schema Schema, err error) {
	d.limiter.ExecuteBlocking(func() {
		schema, err = d.schemaDetails(context.TODO(), schemaID)
	})
	return
}

func (d *DynatraceClient) schemaDetails(ctx context.Context, schemaID string) (Schema, error) {
	s := Schema{ID: schemaID}
	u, err := url.JoinPath(d.environmentURL, d.settingsSchemaAPIPath, schemaID)
	if err != nil {
		return s, fmt.Errorf("failed to parse url: %w", err)
	}

	r, err := rest.Get(ctx, d.client, u)
	if err != nil {
		return s, fmt.Errorf("failed to GET schema details for %q: %w", schemaID, err)
	}

	var sd response.SchemaDetail
	err = json.Unmarshal(r.Body, &sd)
	if err != nil {
		return s, rest.NewRespErr("failed to unmarshal response", r).WithRequestInfo(http.MethodGet, u).WithErr(err)
	}

	for _, sc := range sd.SchemaConstraints {
		if sc.Type == "UNIQUE" {
			s.UniqueProperties = append(s.UniqueProperties, sc.UniqueProperties)
		}
	}

	return s, nil
}
