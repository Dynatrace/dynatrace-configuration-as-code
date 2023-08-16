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
)

const endpoint = "platform/storage/management/v1/bucket-definitions"

type Response struct {
	Data []byte
}

type Client struct {
	Url    string
	Client *rest.Client
}

func (c Client) Upsert(ctx context.Context, id string, data []byte) (Response, error) {

	u, err := url.JoinPath(c.Url, endpoint)
	if err != nil {
		return Response{}, fmt.Errorf("faild to create sound url: %w", err)
	}

	err = setBucketName(id, &data)
	if err != nil {
		return Response{}, err
	}

	r, err := c.Client.Post(ctx, u, data)
	if err != nil {
		return Response{}, fmt.Errorf("unable to create object with ID %s: %w", id, err)
	}
	if !r.IsSuccess() {
		return Response{}, rest.NewRespErr(fmt.Sprintf("failed to update object with ID %s (HTTP %d): %s", id, r.StatusCode, string(r.Body)), r).WithRequestInfo(http.MethodPut, u)
	}

	return Response{
		Data: r.Body,
	}, nil
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
