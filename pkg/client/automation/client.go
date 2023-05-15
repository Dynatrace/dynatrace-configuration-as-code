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

package automation

import (
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"net/http"
)

// Response is a "general" Response type holding the ID and the response payload
type Response struct {
	// Id is the identifier that will be used when creating a new automation object
	Id string `json:"id"`
	// Data is the whole body of an automation object
	Data []byte `json:"-"`
}

// UnarshalJSON de-serializes JSON payload into [Response] type
func (r *Response) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}
	if err := json.Unmarshal(rawMap["id"], &r.Id); err != nil {
		return err
	}
	r.Data = data
	return nil
}

// ListResponse Response is a "general" List of Response values holding the ID and the response payload each
type ListResponse struct {
	Count   int        `json:"count"`
	Results []Response `json:"results"`
}

// Resource specifies information about a specific resource
type Resource struct {
	// Path is the API path to be used for this resource
	Path string
}

// ResourceType enumerates different kind of resources
type ResourceType int

const (
	Workflows ResourceType = iota
	BusinessCalendars
	SchedulingRules
)

var resources = map[ResourceType]Resource{
	Workflows:         {Path: "/platform/automation/v1/workflows"},
	BusinessCalendars: {Path: "/platform/automation/v1/business-calendars"},
	SchedulingRules:   {Path: "/platform/automation/v1/scheduling-rules"},
}

// Client can be used to interact with the Automation API
type Client struct {
	url       string
	limiter   *concurrency.Limiter
	client    *http.Client
	resources map[ResourceType]Resource
}

// ClientOption are (optional) additional parameter passed to the creation of
// an automation client
type ClientOption func(*Client)

// NewClient creates a new client to interact with the Automation API
func NewClient(url string, client *http.Client, opts ...ClientOption) *Client {
	c := &Client{
		url:       url,
		limiter:   concurrency.NewLimiter(5),
		client:    client,
		resources: resources,
	}

	for _, o := range opts {
		o(c)
	}
	return c
}

// WithClientRequestLimiter specifies that a specifies the limiter to be used for
// limiting parallel client requests
func WithClientRequestLimiter(limiter *concurrency.Limiter) func(client *Client) {
	return func(d *Client) {
		d.limiter = limiter
	}
}

// List returns all automation objects
func (a Client) List(resourceType ResourceType) (result *ListResponse, err error) {
	a.limiter.ExecuteBlocking(func() {
		result, err = a.list(resourceType)
	})
	return
}

func (a Client) list(resourceType ResourceType) (*ListResponse, error) {
	// try to get the list of resources
	resp, err := rest.Get(a.client, a.url+a.resources[resourceType].Path)
	if err != nil {
		return nil, fmt.Errorf("unable to list automation resources: %w", err)
	}

	// handle http error
	if !resp.IsSuccess() {
		return nil, ResponseErr{
			StatusCode: resp.StatusCode,
			Message:    "Failed to list automation objects",
			Data:       resp.Body,
		}
	}

	// unmarshal and return result
	var result ListResponse
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// Upsert creates or updates a given automation object
func (a Client) Upsert(resourceType ResourceType, id string, data []byte) (result *Response, err error) {
	if id == "" {
		return nil, fmt.Errorf("id must be non empty")
	}
	a.limiter.ExecuteBlocking(func() {
		result, err = a.upsert(resourceType, id, append([]byte(nil), data...))
	})
	return
}

func (a Client) upsert(resourceType ResourceType, id string, data []byte) (*Response, error) {
	if err := rmIDField(&data); err != nil {
		return nil, fmt.Errorf("unable to remove id field from payload in order to update object with ID %s: %w", id, err)
	}
	// try update via HTTP PUT
	resp, err := rest.Put(a.client, a.url+a.resources[resourceType].Path+"/"+id, data)
	if err != nil {
		return nil, fmt.Errorf("unable to update object with ID %s: %w", id, err)
	}

	// It worked? great, return
	if resp.IsSuccess() {
		log.Debug("Updated object with ID %s", id)
		return &Response{
			Id:   id,
			Data: resp.Body,
		}, nil
	}

	// check if we get an error except 404
	if !resp.IsSuccess() && resp.StatusCode != http.StatusNotFound {
		return nil, ResponseErr{
			StatusCode: resp.StatusCode,
			Message:    "Failed to upsert automation object",
			Data:       resp.Body,
		}
	}

	// at this point we need to create a new object using HTTP POST
	return a.create(id, data, resourceType)
}

func (a Client) create(id string, data []byte, resourceType ResourceType) (*Response, error) {
	// make sure actual "id" field is set in payload
	if err := setIDField(id, &data); err != nil {
		return nil, fmt.Errorf("unable to set the id field in order to crate object with id %s: %w", id, err)
	}

	// try to create a new object using HTTP POST
	resp, err := rest.Post(a.client, a.url+a.resources[resourceType].Path, data)
	if err != nil {
		return nil, err
	}

	// handle response err
	if !resp.IsSuccess() {
		return nil, ResponseErr{
			StatusCode: resp.StatusCode,
			Message:    "Failed to upsert automation object",
			Data:       resp.Body,
		}
	}

	// de-serialize response
	var e Response
	err = json.Unmarshal(resp.Body, &e)
	if err != nil {
		return nil, err
	}

	// check if id from response is indeed the same as desired
	if e.Id != id {
		return nil, fmt.Errorf("returned object ID does not match with the ID used when creating the object")
	}
	log.Debug("Created object with ID %s", id)
	return &e, nil
}

// Delete removes a given automation object by ID
func (a Client) Delete(resourceType ResourceType, id string) (err error) {
	if id == "" {
		return fmt.Errorf("id must be non empty")
	}
	a.limiter.ExecuteBlocking(func() {
		err = a.delete(resourceType, id)
	})
	return
}

func (a Client) delete(resourceType ResourceType, id string) error {
	err := rest.DeleteConfig(a.client, a.url+a.resources[resourceType].Path, id)
	if err != nil {
		return fmt.Errorf("unable to delete object with ID %s: %w", id, err)
	}

	return nil
}

func setIDField(id string, data *[]byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(*data, &m)
	if err != nil {
		return err
	}
	m["id"] = id
	*data, err = json.Marshal(m)
	if err != nil {
		return err
	}
	return nil
}

func rmIDField(data *[]byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(*data, &m)
	if err != nil {
		return err
	}
	delete(m, "id")
	*data, err = json.Marshal(m)
	if err != nil {
		return err
	}
	return nil
}
