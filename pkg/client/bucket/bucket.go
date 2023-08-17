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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const endpoint = "platform/storage/management/v1/bucket-definitions"

type Response struct {
	BucketName string `json:"bucketName"`
	Status     string `json:"status"`
	Version    int    `json:"version"`
	Data       []byte `json:"-"`
}

type Client struct {
	url    string
	client *rest.Client
}

func NewClient(url string, client *rest.Client) *Client {
	return &Client{
		url:    url,
		client: client,
	}
}

func (c Client) Upsert(ctx context.Context, id string, data []byte) (Response, error) {

	time.Sleep(5 * time.Second)
	b, err := c.get(ctx, id)

	if err != nil {
		return c.create(ctx, id, data)
	}

	return c.update(ctx, b, data)
}

func (c Client) create(ctx context.Context, id string, data []byte) (Response, error) {
	u, err := url.JoinPath(c.url, endpoint)
	if err != nil {
		return Response{}, fmt.Errorf("faild to create sound url: %w", err)
	}

	err = setBucketName(id, &data)
	if err != nil {
		return Response{}, err
	}

	r, err := c.client.Post(ctx, u, data)
	if err != nil {
		return Response{}, fmt.Errorf("unable to create object with ID %q: %w", id, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to update object with ID %q (HTTP %d): %s", id, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodPut, u)
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
		return Response{}, fmt.Errorf("unable to update object with ID %q: %w", b.BucketName, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to update object with ID %q (HTTP %d): %s", b.BucketName, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodPut, u.String())
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

func (c Client) get(ctx context.Context, id string) (Response, error) {
	u, err := url.JoinPath(c.url, endpoint, id)
	if err != nil {
		return Response{}, fmt.Errorf("faild to create sound url: %w", err)
	}

	r, err := c.client.Get(ctx, u)
	if err != nil {
		return Response{}, fmt.Errorf("unable to get object with id %q: %w", id, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to get object with ID %q (HTTP %d): %s", id, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodPut, u)
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
