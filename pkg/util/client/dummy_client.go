// @license
// Copyright 2021 Dynatrace LLC
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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

type DataEntry struct {
	Name    string
	Id      string
	Owner   string
	Payload []byte
}

type DummyClient struct {
	Entries          map[api.Api][]DataEntry
	Fs               afero.Fs
	RequestOutputDir string
}

var (
	_ rest.DynatraceClient = (*DummyClient)(nil)
)

func (c *DummyClient) List(a api.Api) (values []api.Value, err error) {
	entries, found := c.Entries[a]

	if !found {
		return nil, nil
	}

	result := make([]api.Value, len(entries))

	for i, entry := range entries {
		result[i] = api.Value{
			Id:    entry.Id,
			Name:  entry.Name,
			Owner: &entry.Owner,
		}
	}

	return result, nil
}

func (c *DummyClient) ReadByName(a api.Api, name string) ([]byte, error) {
	entries, found := c.Entries[a]

	if !found {
		return nil, nil
	}

	for _, entry := range entries {
		if entry.Name == name {
			return entry.Payload, nil
		}
	}

	return nil, fmt.Errorf("nothing found for name %s in api %s", name, a.GetId())
}

func (c *DummyClient) ReadById(a api.Api, id string) ([]byte, error) {
	entries, found := c.Entries[a]

	if !found {
		return nil, nil
	}

	for _, entry := range entries {
		if entry.Id == id {
			return json.Marshal(entry.Payload)
		}
	}

	return nil, fmt.Errorf("nothing found for id %s in api %s", id, a.GetId())
}

func (c *DummyClient) UpsertByName(a api.Api, name string, data []byte) (entity api.DynatraceEntity, err error) {
	entries, found := c.Entries[a]

	if c.Entries == nil {
		c.Entries = make(map[api.Api][]DataEntry)
	}

	if !found {
		c.Entries[a] = make([]DataEntry, 0)
		entries = c.Entries[a]
	}

	var dataEntry *DataEntry

	for i, entry := range entries {
		if entry.Name == name {
			dataEntry = &entries[i]
			break
		}
	}

	if dataEntry == nil {
		dataEntry = &DataEntry{
			Name:  name,
			Id:    uuid.NewString(),
			Owner: "owner",
		}

		c.Entries[a] = append(c.Entries[a], *dataEntry)
		dataEntry = &c.Entries[a][len(c.Entries[a])-1]
	}

	dataEntry.Payload = data
	c.writeRequest(a, name, data)

	return api.DynatraceEntity{
		Id:   dataEntry.Id,
		Name: dataEntry.Name,
	}, nil
}

func (c *DummyClient) writeRequest(a api.Api, name string, payload []byte) {
	if c.Fs == nil {
		return
	}

	filename := fmt.Sprintf("%s-%s-%d.json", a.GetId(), name, time.Now().UnixNano())
	dir := c.RequestOutputDir

	if dir == "" {
		dir = "."
	}

	err := afero.WriteFile(c.Fs, filepath.Join(dir, filename), payload, 0664)

	if err != nil {
		util.Log.Error(err.Error())
	}
}

func (c *DummyClient) BulkDeleteByName(a api.Api, names []string) error {
	for _, name := range names {
		err := c.DeleteByName(a, name)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *DummyClient) DeleteByName(a api.Api, name string) error {
	entries, found := c.Entries[a]

	if !found {
		return nil
	}

	var foundIndex = -1

	for i, entry := range entries {
		if entry.Name == name {
			foundIndex = i
			break
		}
	}

	if foundIndex >= 0 {
		c.Entries[a] = append(entries[:foundIndex], entries[foundIndex+1:]...)
	}

	return nil
}

func (c *DummyClient) ExistsByName(a api.Api, name string) (exists bool, id string, err error) {
	entries, found := c.Entries[a]

	if !found {
		return false, "", errors.New("not found")
	}

	for _, entry := range entries {
		if entry.Name == name {
			return true, entry.Id, nil
		}
	}

	return false, "", nil
}
