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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
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

func (l limitingClient) ListConfigs(a *api.API) (values []api.Value, err error) {
	l.limiter.ExecuteBlocking(func() {
		values, err = l.client.ListConfigs(a)
	})

	return
}

func (l limitingClient) ReadConfigById(a *api.API, id string) (json []byte, err error) {
	l.limiter.ExecuteBlocking(func() {
		json, err = l.client.ReadConfigById(a, id)
	})

	return
}

func (l limitingClient) UpsertConfigByName(a *api.API, name string, payload []byte) (entity api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		entity, err = l.client.UpsertConfigByName(a, name, payload)
	})

	return
}

func (l limitingClient) UpsertConfigByNonUniqueNameAndId(a *api.API, entityId string, name string, payload []byte) (entity api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		entity, err = l.client.UpsertConfigByNonUniqueNameAndId(a, entityId, name, payload)
	})

	return
}

func (l limitingClient) DeleteConfigById(a *api.API, id string) (err error) {
	l.limiter.ExecuteBlocking(func() {
		err = l.client.DeleteConfigById(a, id)
	})

	return
}

func (l limitingClient) ConfigExistsByName(a *api.API, name string) (exists bool, id string, err error) {
	l.limiter.ExecuteBlocking(func() {
		exists, id, err = l.client.ConfigExistsByName(a, name)
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

func (l limitingClient) GetSettingById(objectId string) (o *DownloadSettingsObject, err error) {
	l.limiter.ExecuteBlocking(func() {
		o, err = l.client.GetSettingById(objectId)
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

func (l limitingClient) ListEntities(entitiesType EntitiesType) (o []string, err error) {
	l.limiter.ExecuteBlocking(func() {
		o, err = l.client.ListEntities(entitiesType)
	})

	return
}
