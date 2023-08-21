/*
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

package bucket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const endpoint = "platform/storage/management/v1/bucket-definitions"

type (
	// Response holds all necessary information to create and update Grail Buckets
	Response struct {
		BucketName string `json:"bucketName"`
		Status     string `json:"status"`
		Version    int    `json:"version"`
		Data       []byte `json:"-"`
	}

	// Client abstracts the API access for Grail Buckets
	Client struct {
		url     string
		client  *rest.Client
		limiter *concurrency.Limiter
	}
)

// NewClient creates a new client to interact with the Grail Bucket API
func NewClient(url string, client *rest.Client) *Client {
	return &Client{
		url:     url,
		client:  client,
		limiter: concurrency.NewLimiter(5),
	}
}

// Upsert create or updates a given Grail Bucket
func (c Client) Upsert(ctx context.Context, bucketName string, data []byte) (result Response, err error) {
	if bucketName == "" {
		return Response{}, fmt.Errorf("bucketName must be non empty")
	}
	c.limiter.ExecuteBlocking(func() {
		result, err = c.upsert(ctx, bucketName, data)
	})
	return
}

// upsert creates or updates a given bucket.
//
// Due to concurrency issues on the server side, we decided to do it as follows:
// First, we try to create the bucket. If we succeed, we return with the created bucket.
// If the creation fails, we fetch the existing bucket, and perform an update.
//
// This is done like this, as the server did not recognize the existing object immediately after creation.
// Retrying the GET request multiple times solves this issue, however, this leads to problems during creation, as
// the fetch fails multiple times, because the object has not been created yet.
func (c Client) upsert(ctx context.Context, bucketName string, data []byte) (Response, error) {
	r, err := c.create(ctx, bucketName, data)
	if err == nil {
		log.WithCtxFields(ctx).Debug("Created bucket with bucketName %q", bucketName)
		return r, nil
	}

	log.WithCtxFields(ctx).WithFields(field.Error(err)).Debug("Failed to create new object with bucketName %q. Trying to update existing object. Error: %s", bucketName, err)

	b, err := c.Get(ctx, bucketName)
	if err != nil {
		return Response{}, fmt.Errorf("failed to get object with bucketName %q: %w", bucketName, err)
	}

	r, err = c.update(ctx, b, data)
	log.WithCtxFields(ctx).Debug("Updated bucket with bucketName %q", bucketName)
	return r, err
}

func (c Client) create(ctx context.Context, bucketName string, data []byte) (Response, error) {
	u, err := url.JoinPath(c.url, endpoint)
	if err != nil {
		return Response{}, fmt.Errorf("faild to create sound url: %w", err)
	}

	err = setBucketName(bucketName, &data)
	if err != nil {
		return Response{}, err
	}

	r, err := c.client.Post(ctx, u, data)
	if err != nil {
		return Response{}, fmt.Errorf("failed to create object with bucketName %q: %w", bucketName, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to create object with bucketName %q (HTTP %d): %s", bucketName, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodPut, u)
	}

	b, err := unmarshalJSON(r.Body)
	if err != nil {
		return Response{}, fmt.Errorf("unable to read response: %w", err)
	}

	return b, nil
}

func (c Client) update(ctx context.Context, b Response, data []byte) (Response, error) {

	u, err := url.Parse(c.url)
	if err != nil {
		return Response{}, fmt.Errorf("failed to parse url: %w", err)
	}

	u.Path, err = url.JoinPath(u.Path, endpoint, b.BucketName)
	if err != nil {
		return Response{}, fmt.Errorf("failed to join url: %w", err)
	}

	q := u.Query()
	q.Add("optimistic-locking-version", strconv.Itoa(b.Version))
	u.RawQuery = q.Encode()

	var m map[string]any
	err = json.Unmarshal(data, &m)
	if err != nil {
		return Response{}, fmt.Errorf("unable to unmarshal template: %w", err)
	}
	m["bucketName"] = b.BucketName
	m["version"] = b.Version
	m["status"] = b.Status

	data, err = json.Marshal(m)
	if err != nil {
		return Response{}, fmt.Errorf("unable to marshal data: %w", err)
	}

	r, err := c.client.Put(ctx, u.String(), data)
	if err != nil {
		return Response{}, fmt.Errorf("unable to update object with bucketName %q: %w", b.BucketName, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to update object with bucketName %q (HTTP %d): %s", b.BucketName, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodPut, u.String())
	}

	return b, nil
}

func setBucketName(bucketName string, data *[]byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(*data, &m)
	if err != nil {
		return err
	}
	m["bucketName"] = bucketName
	*data, err = json.Marshal(m)
	if err != nil {
		return err
	}
	return nil
}

// Get fetches a single bucket based given the bucketName
func (c Client) Get(ctx context.Context, bucketName string) (Response, error) {
	u, err := url.JoinPath(c.url, endpoint, bucketName)
	if err != nil {
		return Response{}, fmt.Errorf("faild to create sound url: %w", err)
	}

	retry := rest.RetrySetting{
		WaitTime:   time.Second,
		MaxRetries: 3,
	}

	r, err := c.client.GetWithRetry(ctx, u, retry)
	if err != nil {
		return Response{}, fmt.Errorf("unable to get object with bucketName %q: %w", bucketName, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to get object with bucketName %q (HTTP %d): %s", bucketName, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodGet, u)
	}

	b, err := unmarshalJSON(r.Body)
	if err != nil {
		return Response{}, err
	}

	return b, nil
}

func unmarshalJSON(data []byte) (Response, error) {
	var r Response
	err := json.Unmarshal(data, &r)
	if err != nil {
		return Response{}, fmt.Errorf("fail to unmarshall response: %w", err)
	}
	r.Data = data
	return r, nil
}
