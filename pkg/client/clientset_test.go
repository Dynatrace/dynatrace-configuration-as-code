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

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateClassicClientSet(t *testing.T) {
	t.Run("URL with leading space - should return an error", func(t *testing.T) {
		_, err := CreateClassicClientSet(" https://my-environment.live.dynatrace.com/", "", ClientOptions{})
		assert.Error(t, err)
	})

	t.Run("URL is without scheme - should throw an error", func(t *testing.T) {
		_, err := CreateClassicClientSet("some-url.com", "", ClientOptions{})
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL is without valid local path - should return an error", func(t *testing.T) {
		_, err := CreateClassicClientSet("/my-environment/live/dynatrace.com/", "", ClientOptions{})
		assert.ErrorContains(t, err, "no host specified")
	})

	t.Run("without valid protocol - should return an error", func(t *testing.T) {
		var err error

		_, err = CreateClassicClientSet("https//my-environment.live.dynatrace.com/", "", ClientOptions{})
		assert.ErrorContains(t, err, "not valid")
	})
}
