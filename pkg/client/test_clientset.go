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

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
)

// TestSegmentsClient is a fake client that returns an unimplemented error on every execution of any method.
type TestSegmentsClient struct{}

func (TestSegmentsClient) List(ctx context.Context) (segments.Response, error) {
	return segments.Response{}, fmt.Errorf("unimplemented")
}

func (TestSegmentsClient) GetAll(ctx context.Context) ([]segments.Response, error) {
	return []segments.Response{}, fmt.Errorf("unimplemented")
}

func (TestSegmentsClient) Delete(ctx context.Context, id string) (segments.Response, error) {
	return segments.Response{}, fmt.Errorf("unimplemented")
}

func (TestSegmentsClient) Update(ctx context.Context, id string, data []byte) (segments.Response, error) {
	return segments.Response{}, fmt.Errorf("unimplemented")
}

func (TestSegmentsClient) Create(ctx context.Context, data []byte) (segments.Response, error) {
	return segments.Response{}, nil
}

func (TestSegmentsClient) Get(ctx context.Context, id string) (segments.Response, error) {
	return segments.Response{}, fmt.Errorf("unimplemented")
}

// TestServiceLevelObjectiveClient is a fake client that returns an unimplemented error on every execution of any method.
type TestServiceLevelObjectiveClient struct{}

func (TestServiceLevelObjectiveClient) List(ctx context.Context) (api.PagedListResponse, error) {
	return api.PagedListResponse{}, fmt.Errorf("unimplemented")
}

func (TestServiceLevelObjectiveClient) Update(ctx context.Context, id string, data []byte) (api.Response, error) {
	return api.Response{}, fmt.Errorf("unimplemented")
}

func (TestServiceLevelObjectiveClient) Create(ctx context.Context, data []byte) (api.Response, error) {
	return api.Response{}, fmt.Errorf("unimplemented")
}

func (TestServiceLevelObjectiveClient) Delete(ctx context.Context, id string) (api.Response, error) {
	return api.Response{}, fmt.Errorf("unimplemented")
}
