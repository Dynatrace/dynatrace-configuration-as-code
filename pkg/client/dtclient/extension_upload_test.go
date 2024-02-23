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
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorrectlyIdentifiesLowerLocalVersion(t *testing.T) {
	localPayload := `{ "version": "1" }`
	remotePayload := `{ "version": "2" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.Error(t, err)
	assert.Equal(t, extensionConfigOutdated, status)
}

func TestCorrectlyIdentifiesEqualVersion(t *testing.T) {
	localPayload := `{ "version": "1" }`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.NoError(t, err)
	assert.Equal(t, extensionUpToDate, status)
}

func TestCorrectlyIdentifiesNecessaryUpdate(t *testing.T) {
	localPayload := `{ "version": "1.5" }`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()
	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.NoError(t, err)
	assert.Equal(t, extensionNeedsUpdate, status)
}

func TestCorrectlyIdentifiesMissingExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", nil)
	require.NoError(t, err)
	assert.Equal(t, extensionNeedsUpdate, status)
}

func TestThrowsErrorOnRemoteParsingProblems(t *testing.T) {
	localPayload := `{ "version": "1.5" }`
	remotePayload := `{ "version "1" `

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.Error(t, err)
	assert.Equal(t, extensionValidationError, status)
}

func TestThrowsErrorOnLocalParsingProblems(t *testing.T) {
	localPayload := `version": 1.5"}`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())

	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.Error(t, err)
	assert.Equal(t, extensionValidationError, status)
}

func TestThrowsErrorOnRemoteMissingVersions(t *testing.T) {
	localPayload := `{ "version": "1" }`
	remotePayload := `{ }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.Error(t, err)
	assert.Equal(t, extensionValidationError, status)
}

func TestThrowsErrorOnLocalMissingVersions(t *testing.T) {
	localPayload := `{ "something": "else" }`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.Error(t, err)
	assert.Equal(t, extensionValidationError, status)
}

func TestThrowsErrorOnRemoteNilReturn(t *testing.T) {
	localPayload := `{ "something": "else" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write(nil)
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", []byte(localPayload))
	require.Error(t, err)
	assert.Equal(t, extensionValidationError, status)
}

func TestThrowsErrorOnLocalNilPayload(t *testing.T) {
	remotePayload := `{ "something": "else" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	dtClient, _ := NewDynatraceClientForTesting(server.URL, server.Client())
	status, err := dtClient.validateIfExtensionShouldBeUploaded(context.TODO(), server.URL, "name", nil)
	require.Error(t, err)
	assert.Equal(t, extensionValidationError, status)
}
