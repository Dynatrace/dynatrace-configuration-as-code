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

package classic

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

func newCachedDTClient(client dtclient.Client) dtclient.Client {
	return &cachedDTClient{
		Client: client,
		cache:  make(map[urlPath]listResponse),
	}
}

type (
	cachedDTClient struct {
		dtclient.Client
		cache map[urlPath]listResponse
	}
	urlPath      = string
	listResponse = []dtclient.Value
)

func (c *cachedDTClient) ListConfigs(ctx context.Context, a api.API) ([]dtclient.Value, error) {
	if _, ok := c.cache[a.URLPath]; !ok {
		v, err := c.Client.ListConfigs(ctx, a)
		if err != nil {
			return nil, err
		} else {
			c.cache[a.URLPath] = v
		}
	}
	return c.cache[a.URLPath], nil
}

func (c *cachedDTClient) DeleteConfigById(a api.API, id string) error {
	if a.ID == api.ApplicationWeb {
		delete(c.cache, a.URLPath)
	}
	return c.Client.DeleteConfigById(a, id)
}
