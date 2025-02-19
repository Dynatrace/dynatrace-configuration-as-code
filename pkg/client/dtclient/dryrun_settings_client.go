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

package dtclient

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"
)

type DryRunSettingsClient struct{}

func (c *DryRunSettingsClient) Cache(context.Context, string) error {
	return nil
}

func (c *DryRunSettingsClient) Upsert(_ context.Context, obj SettingsObject, _ UpsertSettingsOptions) (DynatraceEntity, error) {

	id := obj.Coordinate.ConfigId

	// to ensure decoding of Management Zone Numeric IDs works for dry-runs the dry-run client needs to produce a fake but validly formated objectID
	if obj.SchemaId == "builtin:management-zones" {
		uuid := uuid.New().String()
		id = base64.RawURLEncoding.EncodeToString([]byte(uuid))
	}

	return DynatraceEntity{
		Id:   id,
		Name: obj.Coordinate.ConfigId,
	}, nil
}

func (c *DryRunSettingsClient) ListSchemas(_ context.Context) (SchemaList, error) {
	return make(SchemaList, 0), nil
}

func (c *DryRunSettingsClient) GetSchema(_ context.Context, _ string) (schema Schema, err error) {
	return Schema{}, nil
}

func (c *DryRunSettingsClient) Get(_ context.Context, _ string) (*DownloadSettingsObject, error) {
	return &DownloadSettingsObject{}, nil
}
func (c *DryRunSettingsClient) List(_ context.Context, _ string, _ ListSettingsOptions) ([]DownloadSettingsObject, error) {
	return make([]DownloadSettingsObject, 0), nil
}

func (c *DryRunSettingsClient) Delete(_ context.Context, _ string) error {
	return nil
}
