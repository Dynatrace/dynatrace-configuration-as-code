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

package trafficlogs

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
)

func TestFileBasedLogger_Log(t *testing.T) {
	// Create a temporary file system for testing
	fs := afero.NewMemMapFs()

	// Create a new trafficLogger with the temporary file system
	logger := &trafficLogger{
		fs:           fs,
		reqFilePath:  "request.log",
		respFilePath: "response.log",
	}

	// Create a sample request and response
	request := httptest.NewRequest("GET", "http://some-url.com/get", bytes.NewBufferString("request body"))
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("response body")),
	}

	// Log the request and response
	err := logger.Log(request, "request body", response, "response body")
	if err != nil {
		t.Errorf("Log failed: %v", err)
	}

	// Verify that the request and response logs are created
	assert.True(t, fileExists(fs, "request.log"))
	assert.True(t, fileExists(fs, "response.log"))

	// Close the logger
	logger.Close()

	// Verify that the log files are closed
	assert.True(t, fileClosed(logger.reqLogFile))
	assert.True(t, fileClosed(logger.respLogFile))
}

func fileExists(fs afero.Fs, path string) bool {
	exists, _ := afero.Exists(fs, path)
	return exists
}

func fileClosed(file afero.File) bool {
	err := file.Close()
	if err != nil {
		return false
	}
	return true
}
