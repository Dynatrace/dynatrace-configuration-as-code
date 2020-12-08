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

package util

import (
	"gotest.tools/assert"
	"testing"
)

func TestStringTimestampToHumanReadableFormatWithAValidTimestamp(t *testing.T) {

	_, parsedTimestamp, err := StringTimestampToHumanReadableFormat("0") // time travel to the 70s

	assert.NilError(t, err)
	assert.Equal(t, 0, int(parsedTimestamp))
}

func TestStringTimestampToHumanReadableFormatWithAnInvalidTimestampShouldProduceError(t *testing.T) {

	_, _, err := StringTimestampToHumanReadableFormat("abc")
	assert.ErrorContains(t, err, "is not a valid unix timestamp")
}
