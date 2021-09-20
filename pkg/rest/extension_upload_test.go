//go:build unit
// +build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package rest

import (
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorrectlyIdentifiesLowerLocalVersion(t *testing.T) {
	localPayload := `{ "version": "1" }`
	remotePayload := `{ "version": "2" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionConfigOutdated)
}

func TestCorrectlyIdentifiesEqualVersion(t *testing.T) {
	localPayload := `{ "version": "1" }`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.NilError(t, err)
	assert.Equal(t, status, extensionUpToDate)
}

func TestCorrectlyIdentifiesNecessaryUpdate(t *testing.T) {
	localPayload := `{ "version": "1.5" }`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.NilError(t, err)
	assert.Equal(t, status, extensionNeedsUpdate)
}

func TestCorrectlyIdentifiesMissingExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", nil, "token")
	assert.NilError(t, err)
	assert.Equal(t, status, extensionNeedsUpdate)
}

func TestThrowsErrorOnRemoteParsingProblems(t *testing.T) {
	localPayload := `{ "version": "1.5" }`
	remotePayload := `{ "version "1" `

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionValidationError)
}

func TestThrowsErrorOnLocalParsingProblems(t *testing.T) {
	localPayload := `version": 1.5"}`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionValidationError)
}

func TestThrowsErrorOnRemoteMissingVersions(t *testing.T) {
	localPayload := `{ "version": "1" }`
	remotePayload := `{ }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionValidationError)
}

func TestThrowsErrorOnLocalMissingVersions(t *testing.T) {
	localPayload := `{ "something": "else" }`
	remotePayload := `{ "version": "1" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionValidationError)
}

func TestThrowsErrorOnRemoteNilReturn(t *testing.T) {
	localPayload := `{ "something": "else" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write(nil)
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", []byte(localPayload), "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionValidationError)
}

func TestThrowsErrorOnLocalNilPayload(t *testing.T) {
	remotePayload := `{ "something": "else" }`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(remotePayload))
	}))
	defer server.Close()

	status, err := validateIfExtensionShouldBeUploaded(server.Client(), server.URL, "name", nil, "token")
	assert.Assert(t, err != nil)
	assert.Equal(t, status, extensionValidationError)
}
