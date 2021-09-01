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
	"testing"
)

func TestNewClientNoUrl(t *testing.T) {
	client, err := NewDynatraceClient("", "abc")
	assert.ErrorContains(t, err, "no environment url")
	assert.Check(t, client == nil)
}

func TestNewClientNoToken(t *testing.T) {
	client, err := NewDynatraceClient("http://my-environment.live.dynatrace.com/", "")
	assert.ErrorContains(t, err, "no token")
	assert.Check(t, client == nil)
}

func TestNewClientNoValidUrlLocalPath(t *testing.T) {
	client, err := NewDynatraceClient("/my-environment/live/dynatrace.com/", "abc")
	assert.ErrorContains(t, err, "not valid")
	assert.Check(t, client == nil)
}

func TestNewClientNoValidUrlTypo(t *testing.T) {
	client, err := NewDynatraceClient("https//my-environment.live.dynatrace.com/", "abc")
	assert.ErrorContains(t, err, "not valid")
	assert.Check(t, client == nil)
}

func TestNewClientNoValidUrlNoHttps(t *testing.T) {
	client, err := NewDynatraceClient("http//my-environment.live.dynatrace.com/", "abc")
	assert.ErrorContains(t, err, "not valid")
	assert.Check(t, client == nil)
}

func TestNewClient(t *testing.T) {
	client, err := NewDynatraceClient("https://my-environment.live.dynatrace.com/", "abc")
	assert.NilError(t, err, "not valid")
	assert.Check(t, client != nil)
}
