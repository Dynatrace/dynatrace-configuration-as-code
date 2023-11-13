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

package delete

type (
	FileDefinition struct {
		DeleteEntries []any `yaml:"delete"`
	}

	// DeleteEntry defines the one shared property of account delete entries - their Type
	// Individual entries are to be loaded as UserDeleteEntry, GroupDeleteEntry or PolicyDeleteEntry nased on the content of Type
	DeleteEntry struct {
		Type string `yaml:"type" mapstructure:"type"`
	}
	UserDeleteEntry struct {
		Email string `mapstructure:"email"`
	}
	GroupDeleteEntry struct {
		Name string `mapstructure:"name"`
	}
	PolicyDeleteEntry struct {
		Name  string      `mapstructure:"name"`
		Level PolicyLevel `mapstructure:"level"` // either PolicyLevelAccount or PolicyLevelEnvironment
	}
	PolicyLevel struct {
		Type        string `mapstructure:"type"`
		Environment string `mapstructure:"environment"`
	}
)
