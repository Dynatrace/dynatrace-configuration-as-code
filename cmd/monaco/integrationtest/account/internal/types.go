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

type (
	FullFile struct {
		Policies []Policy `mapstructure:"policies"`
		Groups   []Group  `mapstructure:"groups"`
		Users    []User   `mapstructure:"users"`
	}

	Policy struct {
		ID             string      `mapstructure:"id"`
		Name           string      `mapstructure:"name"`
		Level          interface{} `mapstructure:"level"` // either PolicyLevelAccount or PolicyLevelEnvironment
		Description    string      `mapstructure:"description"`
		Policy         string      `mapstructure:"policy"`
		OriginObjectID string      `mapstructure:"originObjectId" yaml:"originObjectId,omitempty"`
	}
	PolicyLevelAccount struct {
		Type string `mapstructure:"type"`
	}
	PolicyLevelEnvironment struct {
		Type        string `mapstructure:"type"`
		Environment string `mapstructure:"environment"`
	}
	Group struct {
		ID             string           `mapstructure:"id"`
		Name           string           `mapstructure:"name"`
		Description    string           `mapstructure:"description"`
		Account        *Account         `mapstructure:"account"`
		Environment    []Environment    `mapstructure:"environment"`
		ManagementZone []ManagementZone `mapstructure:"managementZone" yaml:"managementZone"`
		OriginObjectID string           `mapstructure:"originObjectId" yaml:"originObjectId,omitempty"`
	}
	Account struct {
		Permissions []any `mapstructure:"permissions"`
		Policies    []any `mapstructure:"policies"`
	}
	Environment struct {
		Name        string `mapstructure:"name"`
		Permissions []any  `mapstructure:"permissions"`
		Policies    []any  `mapstructure:"policies"`
	}
	ManagementZone struct {
		Environment    string `mapstructure:"environment"`
		ManagementZone string `mapstructure:"managementZone" yaml:"managementZone"`
		Permissions    []any  `mapstructure:"permissions"`
	}
	User struct {
		Email  string `mapstructure:"email"`
		Groups []any  `mapstructure:"groups"`
	}
	Reference struct {
		Type string `mapstructure:"type"`
		Id   string `mapstructure:"id"`
	}
)
