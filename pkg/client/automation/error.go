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

package automation

import "fmt"

// ResponseErr is used to return HTTP related information as an error
type ResponseErr struct {
	StatusCode int
	Message    string
	Data       []byte
}

func (e ResponseErr) Error() string {
	if e.Message == "" {
		e.Message = "Could not perform HTTP request"
	}
	return fmt.Sprintf("%s (HTTP %d)\n\tResponse was:%s", e.Message, e.StatusCode, e.Data)
}
