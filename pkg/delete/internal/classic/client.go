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
		Client:    client,
		listCache: make(map[urlPath]listResponse),
	}
}

type (
	cachedDTClient struct {
		dtclient.Client
		listCache map[urlPath]listResponse
	}
	urlPath      = string
	listResponse = []dtclient.Value
)

func (client *cachedDTClient) ListConfigs(ctx context.Context, api api.API) ([]dtclient.Value, error) {
	if _, ok := client.listCache[api.URLPath]; !ok {
		v, err := client.Client.ListConfigs(ctx, api)
		if err != nil {
			return nil, err
		} else {
			client.listCache[api.URLPath] = v
		}
	}
	return client.listCache[api.URLPath], nil
}
