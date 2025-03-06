/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package api

import "encoding/json"

func IsSloV1Payload(payload []byte) (bool, error) {
	type SloV1Keys struct {
		EvaluationType string `json:"evaluationType"` // evaluation type is the only required property that is used in rate metrics and non rate metrics
	}
	parsedPayload := SloV1Keys{}
	err := json.Unmarshal(payload, &parsedPayload)
	if err != nil {
		return false, err
	}
	return parsedPayload.EvaluationType != "", nil
}
