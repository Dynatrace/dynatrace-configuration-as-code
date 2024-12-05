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
	context "context"
	"fmt"
	"net/http"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	automationApi "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	buckets "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	documents "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	openpipeline "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	dtclient "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

var DummyClientSet = ClientSet{
	ConfigClient:       &dtclient.DummyConfigClient{},
	SettingsClient:     &dtclient.DummySettingsClient{},
	AutClient:          &DummyAutomationClient{},
	BucketClient:       &DummyBucketClient{},
	DocumentClient:     &DummyDocumentClient{},
	OpenPipelineClient: &DummyOpenPipelineClient{},
}

var _ AutomationClient = (*DummyAutomationClient)(nil)

type DummyAutomationClient struct {
}

// Create implements AutomationClient.
func (d *DummyAutomationClient) Create(ctx context.Context, resourceType automationApi.ResourceType, data []byte) (result coreapi.Response, err error) {
	panic("unimplemented")
}

// Delete implements AutomationClient.
func (d *DummyAutomationClient) Delete(ctx context.Context, resourceType automationApi.ResourceType, id string) (coreapi.Response, error) {
	panic("unimplemented")
}

// Get implements AutomationClient.
func (d *DummyAutomationClient) Get(ctx context.Context, resourceType automationApi.ResourceType, id string) (coreapi.Response, error) {
	panic("unimplemented")
}

// List implements AutomationClient.
func (d *DummyAutomationClient) List(ctx context.Context, resourceType automationApi.ResourceType) (coreapi.PagedListResponse, error) {
	panic("unimplemented")
}

// Update implements AutomationClient.
func (d *DummyAutomationClient) Update(ctx context.Context, resourceType automationApi.ResourceType, id string, data []byte) (coreapi.Response, error) {
	panic("unimplemented")
}

// Upsert implements AutomationClient.
func (d *DummyAutomationClient) Upsert(ctx context.Context, resourceType automationApi.ResourceType, id string, data []byte) (result coreapi.Response, err error) {
	return automation.Response{
		StatusCode: 200,
		Data:       []byte(fmt.Sprintf(`{"id" : "%s"}`, id)),
	}, nil
}

var _ BucketClient = (*DummyBucketClient)(nil)

type DummyBucketClient struct{}

// Create implements BucketClient.
func (d *DummyBucketClient) Create(ctx context.Context, bucketName string, data []byte) (coreapi.Response, error) {
	panic("unimplemented")
}

// Delete implements BucketClient.
func (d *DummyBucketClient) Delete(ctx context.Context, bucketName string) (coreapi.Response, error) {
	panic("unimplemented")
}

// Get implements BucketClient.
func (d *DummyBucketClient) Get(ctx context.Context, bucketName string) (coreapi.Response, error) {
	panic("unimplemented")
}

// List implements BucketClient.
func (d *DummyBucketClient) List(ctx context.Context) (coreapi.PagedListResponse, error) {
	panic("unimplemented")
}

// Update implements BucketClient.
func (d *DummyBucketClient) Update(ctx context.Context, bucketName string, data []byte) (coreapi.Response, error) {
	panic("unimplemented")
}

// Upsert implements BucketClient.
func (d *DummyBucketClient) Upsert(ctx context.Context, bucketName string, data []byte) (coreapi.Response, error) {
	return buckets.Response{
		StatusCode: http.StatusOK,
		Data:       data,
	}, nil
}

var _ DocumentClient = (*DummyDocumentClient)(nil)

type DummyDocumentClient struct{}

// Create implements Client.
func (c *DummyDocumentClient) Create(ctx context.Context, name string, isPrivate bool, externalId string, data []byte, documentType documents.DocumentType) (coreapi.Response, error) {
	return coreapi.Response{Data: []byte(`{}`)}, nil
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
func (c *DummyDocumentClient) Update(ctx context.Context, id string, name string, isPrivate bool, data []byte, documentType documents.DocumentType) (coreapi.Response, error) {
	return coreapi.Response{Data: []byte(`{}`)}, nil
}

// Delete implements DocumentClient.
func (c *DummyDocumentClient) Delete(ctx context.Context, id string) (coreapi.Response, error) {
	panic("unimplemented")
}

var _ OpenPipelineClient = (*DummyOpenPipelineClient)(nil)

type DummyOpenPipelineClient struct{}

// GetAll implements OpenPipelineClient.
func (c *DummyOpenPipelineClient) GetAll(ctx context.Context) ([]coreapi.Response, error) {
	panic("unimplemented")
}

func (c *DummyOpenPipelineClient) Update(_ context.Context, _ string, _ []byte) (openpipeline.Response, error) {
	return openpipeline.Response{}, nil
}