package dtclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"

	"github.com/google/uuid"
	"github.com/spf13/afero"
)

type DataEntry struct {
	Name    string
	Id      string
	Owner   string
	Payload []byte
}

type DummyConfigClient struct {
	entries          map[string][]DataEntry
	entriesLock      sync.RWMutex
	Fs               afero.Fs
	RequestOutputDir string
}

func (c *DummyConfigClient) GetEntries(a api.API) ([]DataEntry, bool) {
	c.entriesLock.RLock()
	defer c.entriesLock.RUnlock()

	v, found := c.entries[a.ID]
	if !found {
		return []DataEntry{}, false
	}
	return v, true
}

func (c *DummyConfigClient) storeEntry(a api.API, e DataEntry) {
	c.entriesLock.Lock()
	defer c.entriesLock.Unlock()

	if c.entries == nil {
		c.entries = make(map[string][]DataEntry)
	}

	entries, exists := c.entries[a.ID]
	if !exists {
		entries = make([]DataEntry, 0)
	}
	entries = append(entries, e)
	c.entries[a.ID] = entries
}

func (c *DummyConfigClient) CreatedObjects() int {
	c.entriesLock.RLock()
	defer c.entriesLock.RUnlock()

	objects := 0
	for _, entries := range c.entries {
		objects += len(entries)
	}
	return objects
}

func (c *DummyConfigClient) Cache(ctx context.Context, a api.API) error {
	return nil
}

func (c *DummyConfigClient) List(_ context.Context, a api.API) (values []Value, err error) {
	entries, found := c.GetEntries(a)

	if !found {
		return nil, nil
	}

	result := make([]Value, len(entries))

	for i, entry := range entries {
		owner := entry.Owner
		result[i] = Value{
			Id:    entry.Id,
			Name:  entry.Name,
			Owner: &owner,
		}
	}

	return result, nil
}

func (c *DummyConfigClient) Get(_ context.Context, a api.API, id string) ([]byte, error) {
	entries, found := c.GetEntries(a)

	if !found {
		return nil, nil
	}

	for _, entry := range entries {
		if entry.Id == id {
			return json.Marshal(entry.Payload)
		}
	}

	return nil, fmt.Errorf("nothing found for id %s in api %s", id, a.ID)
}

func (c *DummyConfigClient) UpsertConfigByName(_ context.Context, a api.API, name string, data []byte) (entity DynatraceEntity, err error) {
	entries, _ := c.GetEntries(a)

	var dataEntry DataEntry
	var entryFound bool

	for i, entry := range entries {
		if entry.Name == name {
			dataEntry = entries[i]
			entryFound = true
			break
		}
	}

	if !entryFound {
		dataEntry = DataEntry{
			Name:  name,
			Id:    uuid.NewString(),
			Owner: "owner",
		}

		c.storeEntry(a, dataEntry)
	}

	dataEntry.Payload = data
	c.writeRequest(a, name, data)

	return DynatraceEntity{
		Id:   dataEntry.Id,
		Name: dataEntry.Name,
	}, nil
}

func (c *DummyConfigClient) UpsertConfigByNonUniqueNameAndId(_ context.Context, a api.API, entityId string, name string, data []byte, _ bool) (entity DynatraceEntity, err error) {
	entries, _ := c.GetEntries(a)

	var dataEntry DataEntry
	var entryFound bool

	for i, entry := range entries {
		if entry.Id == entityId {
			dataEntry = entries[i]
			entryFound = true
			break
		}
	}

	if !entryFound {
		dataEntry = DataEntry{
			Name:  name,
			Id:    entityId,
			Owner: "owner",
		}

		c.storeEntry(a, dataEntry)
	}

	dataEntry.Payload = data
	c.writeRequest(a, name, data)

	return DynatraceEntity{
		Id:   dataEntry.Id,
		Name: dataEntry.Name,
	}, nil
}

func (c *DummyConfigClient) writeRequest(a api.API, name string, payload []byte) {
	if c.Fs == nil {
		return
	}

	filename := fmt.Sprintf("%s-%s-%d.json", a.ID, name, time.Now().UnixNano())
	dir := c.RequestOutputDir

	if dir == "" {
		dir = "."
	}

	err := afero.WriteFile(c.Fs, filepath.Join(dir, filename), payload, 0664)

	if err != nil {
		log.Error(err.Error())
	}
}

func (c *DummyConfigClient) DeleteConfigById(_ context.Context, a api.API, id string) error {

	c.entriesLock.Lock()
	defer c.entriesLock.Unlock()

	entries, found := c.entries[a.ID]

	if !found {
		return nil
	}

	var foundIndex = -1

	for i, entry := range entries {
		if entry.Id == id {
			foundIndex = i
			break
		}
	}

	if foundIndex >= 0 {
		newEntries := append(entries[:foundIndex], entries[foundIndex+1:]...)
		c.entries[a.ID] = newEntries
	}

	return nil
}

func (c *DummyConfigClient) ExistsWithName(_ context.Context, a api.API, name string) (exists bool, id string, err error) {
	entries, found := c.GetEntries(a)

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
