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

package rest

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/concurrency"
)

type limitingClient struct {
	client  DynatraceClient
	limiter *concurrency.Limiter
}

var _ DynatraceClient = (*limitingClient)(nil)

// LimitClientParallelRequests utilizes the decorator pattern to limit parallel requests to the dynatrace API
func LimitClientParallelRequests(client DynatraceClient, maxParallelRequests int) DynatraceClient {
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

func (l limitingClient) UpsertByEntityId(a api.Api, entityId string, name string, payload []byte) (entity api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		entity, err = l.client.UpsertByEntityId(a, entityId, name, payload)
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

func (l limitingClient) UpsertSettings(knownSettings KnownSettings, obj SettingsObject) (e api.DynatraceEntity, err error) {
	l.limiter.ExecuteBlocking(func() {
		e, err = l.client.UpsertSettings(knownSettings, obj)
	})

	return
}

func (l limitingClient) ListKnownSettings(schemaId string) (k KnownSettings, err error) {
	l.limiter.ExecuteBlocking(func() {
		k, err = l.client.ListKnownSettings(schemaId)
	})

	return
}

func (l limitingClient) ListSchemas() (s SchemaList, err error) {
	l.limiter.ExecuteBlocking(func() {
		s, err = l.client.ListSchemas()
	})

	return
}

func (l limitingClient) ListSettings(schema string) (o []DownloadSettingsObject, err error) {
	l.limiter.ExecuteBlocking(func() {
		o, err = l.client.ListSettings(schema)
	})

	return
}
