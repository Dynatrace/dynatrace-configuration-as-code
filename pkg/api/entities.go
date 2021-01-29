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

package api

type ValuesResponse struct {
	Values []Value `json:"values"`
}

type SyntheticLocationResponse struct {
	Locations []SyntheticValue `json:"locations"`
}

type SyntheticMonitorsResponse struct {
	Monitors []SyntheticValue `json:"monitors"`
}

type Value struct {
	Id    string  `json:"id"`
	Name  string  `json:"name"`
	Owner *string `json:"owner,omitempty"`
}

type SyntheticValue struct {
	Name          string    `json:"name"`
	EntityId      string    `json:"entityId"`
	Type          string    `json:"type"`
	CloudPlatform *string   `json:"cloudPlatform"`
	Ips           *[]string `json:"ips"`
	Stage         *string   `json:"stage"`
	Enabled       *bool     `json:"enabled"`
}

type SyntheticEntity struct {
	EntityId string `json:"entityId"`
}

type DynatraceEntity struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
