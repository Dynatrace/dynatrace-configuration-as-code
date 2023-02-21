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

package log

import (
	"gotest.tools/assert"
	"net/http"
	"testing"
)

var shouldDumpCases = []string{
	"application/json",
	"application/json; charset=utf-8",
	"application/json;charset=utf-8",
	"text/plain",
	"text/plain; charset=utf-16",
	"application/xml",
	"text/xml",
}

var shouldNotDumpCases = []string{
	"application/binary",
	"application/java-archive",
	"application/yaml",
	"video/quicktime",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"multipart/mixed",
}

func TestShouldDumpBodyPositiveCases(t *testing.T) {

	for _, dumpCase := range shouldDumpCases {
		assert.Equal(t, shouldDumpBodyForContentType(dumpCase), true, "Should dump content-type '%v'", dumpCase)
	}
}

func TestShouldDumpBodyNegativeCases(t *testing.T) {

	for _, notDumpCase := range shouldNotDumpCases {
		assert.Equal(t, shouldDumpBodyForContentType(notDumpCase), false, "Should not dump content-type '%v'", notDumpCase)
	}
}

func TestShouldDumpContentTypeInHeader(t *testing.T) {

	for _, dumpCase := range shouldDumpCases {

		var headers http.Header = map[string][]string{
			"Content-Type": {dumpCase},
		}

		assert.Equal(t, shouldDumpBody(headers), true, "Should dump content-type '%v'", dumpCase)
	}
}

func TestShouldNotDumpContentTypeInHeader(t *testing.T) {

	for _, notDumpCase := range shouldNotDumpCases {

		var headers http.Header = map[string][]string{
			"Content-Type": {notDumpCase},
		}

		assert.Equal(t, shouldDumpBody(headers), false, "Should not dump content-type '%v'", notDumpCase)
	}
}
