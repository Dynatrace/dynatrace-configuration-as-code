//go:build unit

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

package automationutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
)

func TestDecodeResponse(t *testing.T) {
	tests := []struct {
		name    string
		given   api.Response
		want    automationutils.Response
		wantErr bool
	}{
		{
			"decodes simple response",
			api.Response{
				StatusCode: 200,
				Data:       []byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
			},
			automationutils.Response{
				ID:   "some-id",
				Data: []byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
			},
			false,
		},
		{
			"error if ID is missing",
			api.Response{
				StatusCode: 200,
				Data:       []byte(`{"workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
			},
			automationutils.Response{},
			true,
		},
		{
			"error if data empty",
			api.Response{
				StatusCode: 200,
				Data:       []byte{},
			},
			automationutils.Response{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := automationutils.DecodeResponse(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeListResponse(t *testing.T) {
	tests := []struct {
		name    string
		given   api.PagedListResponse
		want    []automationutils.Response
		wantErr bool
	}{
		{
			"decodes simple response",
			api.PagedListResponse{
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data:       []byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
					Objects: [][]byte{
						[]byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
				},
			},
			[]automationutils.Response{
				{
					ID:   "some-id",
					Data: []byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
				},
			},
			false,
		},
		{
			"decodes response list",
			api.PagedListResponse{
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data:       []byte(`count: 4, results: [{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}]`),
					},
					Objects: [][]byte{
						[]byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
				},
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data: []byte(`count: 4, results: [
{ "id": "some-other-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]},
{ "id": "some-other-id-2", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]},
{ "id": "some-other-id-3", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}]`),
					},
					Objects: [][]byte{
						[]byte(`{ "id": "some-other-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
						[]byte(`{ "id": "some-other-id-2", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
						[]byte(`{ "id": "some-other-id-3", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
				},
			},
			[]automationutils.Response{
				{
					ID:   "some-id",
					Data: []byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
				},
				{
					ID:   "some-other-id",
					Data: []byte(`{ "id": "some-other-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
				},
				{
					ID:   "some-other-id-2",
					Data: []byte(`{ "id": "some-other-id-2", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
				},
				{
					ID:   "some-other-id-3",
					Data: []byte(`{ "id": "some-other-id-3", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
				},
			},
			false,
		},
		{
			"error of one element is missing ID",
			api.PagedListResponse{
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data:       []byte(`count: 4, results: [{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}]`),
					},
					Objects: [][]byte{
						[]byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
				},
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data: []byte(`count: 4, results: [
{ "id": "some-other-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]},
{ "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]},
{ "id": "some-other-id-3", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}]`),
					},
					Objects: [][]byte{
						[]byte(`{ "id": "some-other-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
						[]byte(`{ "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
						[]byte(`{ "id": "some-other-id-3", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
				},
			},
			[]automationutils.Response{},
			true,
		},
		{
			"error of one response is empty",
			api.PagedListResponse{
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data:       []byte(`count: 4, results: [{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}]`),
					},
					Objects: [][]byte{
						[]byte(`{ "id": "some-id", "workflow-steps": [{"some": "value"},{"some": "value"},{"some": "value"}]}`),
					},
				},
				api.ListResponse{
					Response: api.Response{
						StatusCode: 200,
						Data:       []byte("{}"),
					},
					Objects: [][]byte{[]byte("{}")},
				},
			},
			[]automationutils.Response{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := automationutils.DecodeListResponse(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}
