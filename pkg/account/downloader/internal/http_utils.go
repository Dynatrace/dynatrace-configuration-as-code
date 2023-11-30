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

package internal

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"io"
	"net/http"
)

func closeResponseBody(resp *http.Response) {
	_ = resp.Body.Close()
}

func handleClientResponseError(resp *http.Response, clientErr error) error {
	if clientErr != nil && resp == nil {
		return clientErr
	}

	if !rest.IsSuccess(resp) && resp.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body %w", err)
		}
		return fmt.Errorf("(HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
