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

package client

import (
	"context"
	"fmt"
	"net/http"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	automationApi "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

var DryRunClientSet = ClientSet{
	ConfigClient:                &dtclient.DryRunConfigClient{},
	SettingsClient:              &dtclient.DryRunSettingsClient{},
	AutClient:                   &DryRunAutomationClient{},
	BucketClient:                &DryRunBucketClient{},
	DocumentClient:              &DryRunDocumentClient{},
	OpenPipelineClient:          &DryRunOpenPipelineClient{},
	SegmentClient:               &DryRunSegmentClient{},
	ServiceLevelObjectiveClient: &DryRunServiceLevelObjectClient{},
}

var _ AutomationClient = (*DryRunAutomationClient)(nil)

type DryRunAutomationClient struct {
}

// Create implements AutomationClient.
func (d *DryRunAutomationClient) Create(ctx context.Context, resourceType automationApi.ResourceType, data []byte) (result coreapi.Response, err error) {
	panic("unimplemented")
}

// Delete implements AutomationClient.
func (d *DryRunAutomationClient) Delete(ctx context.Context, resourceType automationApi.ResourceType, id string) (coreapi.Response, error) {
	panic("unimplemented")
}

// Get implements AutomationClient.
func (d *DryRunAutomationClient) Get(ctx context.Context, resourceType automationApi.ResourceType, id string) (coreapi.Response, error) {
	panic("unimplemented")
}

// List implements AutomationClient.
func (d *DryRunAutomationClient) List(ctx context.Context, resourceType automationApi.ResourceType) (coreapi.PagedListResponse, error) {
	panic("unimplemented")
}

// Update implements AutomationClient.
func (d *DryRunAutomationClient) Update(ctx context.Context, resourceType automationApi.ResourceType, id string, data []byte) (coreapi.Response, error) {
	panic("unimplemented")
}

// Upsert implements AutomationClient.
func (d *DryRunAutomationClient) Upsert(ctx context.Context, resourceType automationApi.ResourceType, id string, data []byte) (result coreapi.Response, err error) {
	return automation.Response{
		StatusCode: 200,
		Data:       []byte(fmt.Sprintf(`{"id" : "%s"}`, id)),
	}, nil
}

var _ BucketClient = (*DryRunBucketClient)(nil)

type DryRunBucketClient struct{}

// Create implements BucketClient.
func (d *DryRunBucketClient) Create(ctx context.Context, bucketName string, data []byte) (coreapi.Response, error) {
	panic("unimplemented")
}

// Delete implements BucketClient.
func (d *DryRunBucketClient) Delete(ctx context.Context, bucketName string) (coreapi.Response, error) {
	panic("unimplemented")
}

// Get implements BucketClient.
func (d *DryRunBucketClient) Get(ctx context.Context, bucketName string) (coreapi.Response, error) {
	panic("unimplemented")
}

// List implements BucketClient.
func (d *DryRunBucketClient) List(ctx context.Context) (coreapi.PagedListResponse, error) {
	panic("unimplemented")
}

// Update implements BucketClient.
func (d *DryRunBucketClient) Update(ctx context.Context, bucketName string, data []byte) (coreapi.Response, error) {
	panic("unimplemented")
}

// Upsert implements BucketClient.
func (d *DryRunBucketClient) Upsert(ctx context.Context, bucketName string, data []byte) (coreapi.Response, error) {
	return buckets.Response{
		StatusCode: http.StatusOK,
		Data:       data,
	}, nil
}

var _ DocumentClient = (*DryRunDocumentClient)(nil)

type DryRunDocumentClient struct{}

// Create implements Client.
func (c *DryRunDocumentClient) Create(ctx context.Context, name string, isPrivate bool, externalId string, data []byte, documentType documents.DocumentType) (coreapi.Response, error) {
	return coreapi.Response{Data: []byte(`{}`)}, nil
}

// Get implements Client.
func (c *DryRunDocumentClient) Get(ctx context.Context, id string) (documents.Response, error) {
	return documents.Response{}, nil
}

// List implements Client.
func (c *DryRunDocumentClient) List(ctx context.Context, filter string) (documents.ListResponse, error) {
	return documents.ListResponse{}, nil
}

// Update implements Client.
func (c *DryRunDocumentClient) Update(ctx context.Context, id string, name string, isPrivate bool, data []byte, documentType documents.DocumentType) (coreapi.Response, error) {
	return coreapi.Response{Data: []byte(`{}`)}, nil
}

// Delete implements DocumentClient.
func (c *DryRunDocumentClient) Delete(ctx context.Context, id string) (coreapi.Response, error) {
	panic("unimplemented")
}

var _ OpenPipelineClient = (*DryRunOpenPipelineClient)(nil)

type DryRunOpenPipelineClient struct{}

// GetAll implements OpenPipelineClient.
func (c *DryRunOpenPipelineClient) GetAll(ctx context.Context) ([]coreapi.Response, error) {
	panic("unimplemented")
}

func (c *DryRunOpenPipelineClient) Update(_ context.Context, _ string, _ []byte) (openpipeline.Response, error) {
	return openpipeline.Response{}, nil
}

type DryRunSegmentClient struct{}

func (c *DryRunSegmentClient) List(_ context.Context) (segments.Response, error) {
	return segments.Response{}, nil
}

func (c *DryRunSegmentClient) GetAll(_ context.Context) ([]segments.Response, error) {
	return []segments.Response{}, nil
}

func (c *DryRunSegmentClient) Delete(_ context.Context, _ string) (segments.Response, error) {
	return segments.Response{}, nil
}

func (c *DryRunSegmentClient) Create(_ context.Context, _ []byte) (segments.Response, error) {
	return segments.Response{Data: []byte(`{}`)}, nil
}

func (c *DryRunSegmentClient) Update(_ context.Context, _ string, _ []byte) (segments.Response, error) {
	return segments.Response{}, nil
}

func (c *DryRunSegmentClient) Get(_ context.Context, _ string) (segments.Response, error) {
	return segments.Response{}, nil
}

type DryRunServiceLevelObjectClient struct{}

func (c *DryRunServiceLevelObjectClient) List(_ context.Context) (coreapi.PagedListResponse, error) {
	return coreapi.PagedListResponse{}, nil
}

func (c *DryRunServiceLevelObjectClient) Update(_ context.Context, _ string, _ []byte) (coreapi.Response, error) {
	return coreapi.Response{}, nil
}

func (c *DryRunServiceLevelObjectClient) Create(_ context.Context, _ []byte) (coreapi.Response, error) {
	return coreapi.Response{Data: []byte(`{}`)}, nil
}
