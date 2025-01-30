//go:build integration || download_restore || unit || nightly || cleanup

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

package testutils

import (
	"testing"
)

func FailTestOnAnyError(t *testing.T, errors []error, errorMessage string) {
	if len(errors) == 0 {
		return
	}

	for _, err := range errors {
		t.Logf("%s: %v", errorMessage, err)
	}
	t.FailNow()
}
