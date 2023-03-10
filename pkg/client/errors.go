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
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
)

type RespError struct {
	Err        error
	StatusCode int
}

func (e RespError) Error() string {
	return e.Err.Error()
}

func (e RespError) Unwrap() error {
	return e.Err
}

func (e RespError) ConcurrentError() string {
	if e.StatusCode == 403 {
		concurrentDownloadLimit := environment.GetEnvValueInt(environment.ConcurrentRequestsEnvKey)
		additionalMessage := fmt.Sprintf("\n\n    A 403 error code probably means too many requests.\n    Reduce the number of concurrent requests by setting the %q environment variable (current value: %d). \n    Then wait a few minutes and retry ", environment.ConcurrentRequestsEnvKey, concurrentDownloadLimit)
		return fmt.Sprintf("%s\n%s", e.Err.Error(), additionalMessage)
	}

	return e.Error()
}
