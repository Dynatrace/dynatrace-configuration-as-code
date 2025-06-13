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

package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	libAutomation "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

var DummyClientSet = ClientSet{
	ConfigClient:                &dtclient.DummyConfigClient{},
	SettingsClient:              &dtclient.DummySettingsClient{},
	AutClient:                   &DummyAutomationClient{},
	BucketClient:                &DummyBucketClient{},
	DocumentClient:              &DummyDocumentClient{},
	OpenPipelineClient:          &DummyOpenPipelineClient{},
	SegmentClient:               &DummySegmentClient{},
	ServiceLevelObjectiveClient: &DummyServiceLevelObjectClient{},
}

var _ AutomationClient = (*DummyAutomationClient)(nil)

type DummyAutomationClient struct {
}

// Create implements AutomationClient.
func (d *DummyAutomationClient) Create(ctx context.Context, resourceType libAutomation.ResourceType, data []byte) (result api.Response, err error) {
	panic("unimplemented")
}

// Delete implements AutomationClient.
func (d *DummyAutomationClient) Delete(ctx context.Context, resourceType libAutomation.ResourceType, id string) (api.Response, error) {
	panic("unimplemented")
}

// Get implements AutomationClient.
func (d *DummyAutomationClient) Get(ctx context.Context, resourceType libAutomation.ResourceType, id string) (api.Response, error) {
	panic("unimplemented")
}

// List implements AutomationClient.
func (d *DummyAutomationClient) List(ctx context.Context, resourceType libAutomation.ResourceType) (api.PagedListResponse, error) {
	panic("unimplemented")
}

// Update implements AutomationClient.
func (d *DummyAutomationClient) Update(ctx context.Context, resourceType libAutomation.ResourceType, id string, data []byte) (api.Response, error) {
	panic("unimplemented")
}

// Upsert implements AutomationClient.
func (d *DummyAutomationClient) Upsert(ctx context.Context, resourceType libAutomation.ResourceType, id string, data []byte) (result api.Response, err error) {
	return automation.Response{
		StatusCode: 200,
		Data:       []byte(fmt.Sprintf(`{"id" : "%s"}`, id)),
	}, nil
}

var _ BucketClient = (*DummyBucketClient)(nil)

type DummyBucketClient struct{}

// Create implements BucketClient.
func (d *DummyBucketClient) Create(ctx context.Context, bucketName string, data []byte) (api.Response, error) {
	panic("unimplemented")
}

// Delete implements BucketClient.
func (d *DummyBucketClient) Delete(ctx context.Context, bucketName string) (api.Response, error) {
	panic("unimplemented")
}

// Get implements BucketClient.
func (d *DummyBucketClient) Get(ctx context.Context, bucketName string) (api.Response, error) {
	panic("unimplemented")
}

// List implements BucketClient.
func (d *DummyBucketClient) List(ctx context.Context) (api.PagedListResponse, error) {
	panic("unimplemented")
}

// Update implements BucketClient.
func (d *DummyBucketClient) Update(ctx context.Context, bucketName string, data []byte) (api.Response, error) {
	panic("unimplemented")
}

// Upsert implements BucketClient.
func (d *DummyBucketClient) Upsert(ctx context.Context, bucketName string, data []byte) (api.Response, error) {
	return api.Response{
		StatusCode: http.StatusOK,
		Data:       data,
	}, nil
}

var _ DocumentClient = (*DummyDocumentClient)(nil)

type DummyDocumentClient struct{}

// Create implements Client.
func (c *DummyDocumentClient) Create(ctx context.Context, name string, isPrivate bool, externalId string, data []byte, documentType documents.DocumentType) (api.Response, error) {
	return api.Response{Data: []byte(`{}`)}, nil
}

// Get implements Client.
func (c *DummyDocumentClient) Get(ctx context.Context, id string) (documents.Response, error) {
	return documents.Response{}, nil
}

// List implements Client.
func (c *DummyDocumentClient) List(ctx context.Context, filter string) (documents.ListResponse, error) {
	return documents.ListResponse{}, nil
}

// Update implements Client.
func (c *DummyDocumentClient) Update(ctx context.Context, id string, name string, isPrivate bool, data []byte, documentType documents.DocumentType) (api.Response, error) {
	return api.Response{Data: []byte(`{}`)}, nil
}

// Delete implements DocumentClient.
func (c *DummyDocumentClient) Delete(ctx context.Context, id string) (api.Response, error) {
	panic("unimplemented")
}

var _ OpenPipelineClient = (*DummyOpenPipelineClient)(nil)

type DummyOpenPipelineClient struct{}

// GetAll implements OpenPipelineClient.
func (c *DummyOpenPipelineClient) GetAll(ctx context.Context) ([]api.Response, error) {
	panic("unimplemented")
}

func (c *DummyOpenPipelineClient) Update(_ context.Context, _ string, _ []byte) (openpipeline.Response, error) {
	return openpipeline.Response{}, nil
}

type DummySegmentClient struct{}

func (c *DummySegmentClient) List(_ context.Context) (api.Response, error) {
	return api.Response{}, nil
}

func (c *DummySegmentClient) GetAll(_ context.Context) ([]api.Response, error) {
	return []api.Response{}, nil
}

func (c *DummySegmentClient) Delete(_ context.Context, _ string) (api.Response, error) {
	return api.Response{}, nil
}

func (c *DummySegmentClient) Create(_ context.Context, _ []byte) (api.Response, error) {
	return api.Response{Data: []byte(`{}`)}, nil
}

func (c *DummySegmentClient) Update(_ context.Context, _ string, _ []byte) (api.Response, error) {
	return api.Response{}, nil
}

func (c *DummySegmentClient) Get(_ context.Context, _ string) (api.Response, error) {
	return api.Response{}, nil
}

type DummyServiceLevelObjectClient struct{}

func (c *DummyServiceLevelObjectClient) List(_ context.Context) (api.PagedListResponse, error) {
	return api.PagedListResponse{}, nil
}

func (c *DummyServiceLevelObjectClient) Update(_ context.Context, _ string, _ []byte) (api.Response, error) {
	return api.Response{}, nil
}

func (c *DummyServiceLevelObjectClient) Create(_ context.Context, _ []byte) (api.Response, error) {
	return api.Response{Data: []byte(`{}`)}, nil
}

func (c *DummyServiceLevelObjectClient) Delete(_ context.Context, _ string) (api.Response, error) {
	return api.Response{}, nil
}
