// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/concurrency"
)

type limitingClient struct {
	client  Client
	limiter *concurrency.Limiter
}

var _ Client = (*limitingClient)(nil)

// LimitClientParallelRequests utilizes the decorator pattern to limit parallel requests to the dynatrace API
func LimitClientParallelRequests(client Client, maxParallelRequests int) Client {
	return &limitingClient{
		client,
		concurrency.NewLimiter(maxParallelRequests),
	}
}

func (l limitingClient) List(a api.Api) (values []api.Value, err error) {
	l.limiter.ExecuteBlocking(func() {
		values, err = l.client.List(a)
	})

	return
}

func (l limitingClient) ReadById(a api.Api, id string) (json []byte, err error) {
	l.limiter.ExecuteBlocking(func() {
		json, err = l.client.ReadById(a, id)
	})

	return
}

func (l limitingClient) UpsertByName(a api.Api, name string, payload []byte) (entity api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		entity, err = l.client.UpsertByName(a, name, payload)
	})

	return
}

func (l limitingClient) UpsertByNonUniqueNameAndId(a api.Api, entityId string, name string, payload []byte) (entity api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		entity, err = l.client.UpsertByNonUniqueNameAndId(a, entityId, name, payload)
	})

	return
}

func (l limitingClient) DeleteById(a api.Api, id string) (err error) {
	l.limiter.ExecuteBlocking(func() {
		err = l.client.DeleteById(a, id)
	})

	return
}

func (l limitingClient) ExistsByName(a api.Api, name string) (exists bool, id string, err error) {
	l.limiter.ExecuteBlocking(func() {
		exists, id, err = l.client.ExistsByName(a, name)
	})

	return
}

func (l limitingClient) UpsertSettings(obj SettingsObject) (e api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		e, err = l.client.UpsertSettings(obj)
	})

	return
}

func (l limitingClient) ListSchemas() (s SchemaList, err error) {
	l.limiter.ExecuteBlocking(func() {
		s, err = l.client.ListSchemas()
	})

	return
}

func (l limitingClient) ListSettings(schemaId string, opts ListSettingsOptions) (o []DownloadSettingsObject, err error) {
	l.limiter.ExecuteBlocking(func() {
		o, err = l.client.ListSettings(schemaId, opts)
	})

	return
}

func (l limitingClient) DeleteSettings(objectID string) (err error) {
	l.limiter.ExecuteBlocking(func() {
		err = l.client.DeleteSettings(objectID)
	})

	return
}

func (l limitingClient) ListEntitiesTypes() (e []EntitiesType, err error) {

	l.limiter.ExecuteBlocking(func() {
		e, err = l.client.ListEntitiesTypes()
	})

	return
}

func (l limitingClient) ListEntities(entitiesType EntitiesType) (o EntitiesList, err error) {
	l.limiter.ExecuteBlocking(func() {
		o, err = l.client.ListEntities(entitiesType)
	})

	return
}
