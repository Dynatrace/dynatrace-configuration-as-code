//go:build unit

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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockAPI = api.API{ID: "mock-api", SingleConfiguration: true}
var mockAPINotSingle = api.API{ID: "mock-api", SingleConfiguration: false}

func TestNewClassicClient(t *testing.T) {
	t.Run("Client has correct urls and settings api path", func(t *testing.T) {
		client, err := NewClassicClient("https://some-url.com", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "https://some-url.com", client.environmentURL)
		assert.Equal(t, "https://some-url.com", client.environmentURLClassic)
		assert.Equal(t, settingsSchemaAPIPathClassic, client.settingsSchemaAPIPath)
		assert.Equal(t, settingsObjectAPIPathClassic, client.settingsObjectAPIPath)

	})

	t.Run("URL is empty - should throw an error", func(t *testing.T) {
		_, err := NewClassicClient("", nil)
		assert.ErrorContains(t, err, "empty url")

	})

	t.Run("invalid URL - should throw an error", func(t *testing.T) {
		_, err := NewClassicClient("INVALID_URL", nil)
		assert.ErrorContains(t, err, "not valid")

	})

	t.Run("URL suffix is trimmed", func(t *testing.T) {
		client, err := NewClassicClient("http://some-url.com/", nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://some-url.com", client.environmentURL)
		assert.Equal(t, "http://some-url.com", client.environmentURLClassic)
	})

	t.Run("URL with leading space - should return an error", func(t *testing.T) {
		_, err := NewClassicClient(" https://my-environment.live.dynatrace.com/", nil)
		assert.Error(t, err)

	})

	t.Run("URL starts with http", func(t *testing.T) {
		client, err := NewClassicClient("http://some-url.com", nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://some-url.com", client.environmentURL)
		assert.Equal(t, "http://some-url.com", client.environmentURLClassic)

	})

	t.Run("URL is without scheme - should throw an error", func(t *testing.T) {
		_, err := NewClassicClient("some-url.com", nil)
		assert.ErrorContains(t, err, "not valid")

	})

	t.Run("URL is without valid local path - should return an error", func(t *testing.T) {
		_, err := NewClassicClient("/my-environment/live/dynatrace.com/", nil)
		assert.ErrorContains(t, err, "no host specified")

	})

	t.Run("without valid protocol - should return an error", func(t *testing.T) {
		var err error

		_, err = NewClassicClient("https//my-environment.live.dynatrace.com/", nil)
		assert.ErrorContains(t, err, "not valid")
	})
}

func TestNewPlatformClient(t *testing.T) {

	t.Run("Client has correct urls and settings api path", func(t *testing.T) {
		client, err := NewPlatformClient("https://some-url.com", "https://some-url2.com", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "https://some-url.com", client.environmentURL)
		assert.Equal(t, "https://some-url2.com", client.environmentURLClassic)
		assert.Equal(t, settingsSchemaAPIPathPlatform, client.settingsSchemaAPIPath)
		assert.Equal(t, settingsObjectAPIPathPlatform, client.settingsObjectAPIPath)

	})

	t.Run("URL is empty - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient("", "", nil, nil)
		assert.ErrorContains(t, err, "empty url")

		_, err = NewPlatformClient("http://some-url.com", "", nil, nil)
		assert.ErrorContains(t, err, "empty url")
	})

	t.Run("invalid URL - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient("INVALID_URL", "", nil, nil)
		assert.ErrorContains(t, err, "not valid")

		_, err = NewPlatformClient("http://some-url.com", "INVALID_URL", nil, nil)
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL suffix is trimmed", func(t *testing.T) {
		client, err := NewPlatformClient("http://some-url.com/", "http://some-url2.com/", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://some-url.com", client.environmentURL)
		assert.Equal(t, "http://some-url2.com", client.environmentURLClassic)
	})

	t.Run("URL with leading space - should return an error", func(t *testing.T) {
		_, err := NewPlatformClient(" https://my-environment.live.dynatrace.com/", "", nil, nil)
		assert.Error(t, err)

		_, err = NewPlatformClient("https://my-environment.live.dynatrace.com/", " https://my-environment.live.dynatrace.com/\"", nil, nil)
		assert.Error(t, err)
	})

	t.Run("URL starts with http", func(t *testing.T) {
		client, err := NewPlatformClient("http://some-url.com", "https://some-url.com", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://some-url.com", client.environmentURL)

		client, err = NewPlatformClient("https://my-environment.live.dynatrace.com/", "http://some-url.com", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://some-url.com", client.environmentURLClassic)
	})

	t.Run("URL is without scheme - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient("some-url.com", "", nil, nil)
		assert.ErrorContains(t, err, "not valid")

		_, err = NewPlatformClient("https://some-url.com", "some-url.com", nil, nil)
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL is without valid local path - should return an error", func(t *testing.T) {
		_, err := NewPlatformClient("/my-environment/live/dynatrace.com/", "https://some-url.com", nil, nil)
		assert.ErrorContains(t, err, "no host specified")

		_, err = NewPlatformClient("https://some-url.com", "/my-environment/live/dynatrace.com/", nil, nil)
		assert.ErrorContains(t, err, "no host specified")
	})

	t.Run("without valid protocol - should return an error", func(t *testing.T) {
		var err error

		_, err = NewPlatformClient("https//my-environment.live.dynatrace.com/", "", nil, nil)
		assert.ErrorContains(t, err, "not valid")

		_, err = NewPlatformClient("http//my-environment.live.dynatrace.com/", "", nil, nil)
		assert.ErrorContains(t, err, "not valid")
	})
}

func TestCreateDynatraceClientWithAutoServerVersion(t *testing.T) {
	t.Run("Server version is correctly set to determined value", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(`{"version" : "1.262.0.20230214-193525"}`))
		}))

		dcl, err := NewClassicClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()), WithAutoServerVersion())

		server.Close()
		assert.NoError(t, err)
		assert.Equal(t, version.Version{Major: 1, Minor: 262}, dcl.serverVersion)
	})

	t.Run("Server version is correctly set to unknown", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(`{}`))
		}))

		dcl, err := NewClassicClient(server.URL, rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()), WithAutoServerVersion())
		server.Close()
		assert.NoError(t, err)
		assert.Equal(t, version.UnknownVersion, dcl.serverVersion)
	})
}
