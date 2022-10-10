//go:build unit

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

package downloader

import (
	"gotest.tools/assert"
	"os"
	"testing"
)

func TestGetDownloadLimit(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			"no env supplied",
			"",
			defaultConcurrentDownloads,
		},
		{
			"env invalid",
			"invalid",
			defaultConcurrentDownloads,
		},
		{
			"negative",
			"-1",
			defaultConcurrentDownloads,
		},
		{
			"valid env",
			"1000",
			1000,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := os.Setenv(concurrentRequestsEnvKey, test.envValue)
			assert.NilError(t, err)

			limit := getConcurrentDownloadLimit()
			assert.Equal(t, limit, test.expected)
		})
	}
}
