// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manifest

type project struct {
	Name string  `yaml:"name"`
	Type *string `yaml:"type,omitempty"`
	Path string  `yaml:"path"`
}

type tokenConfig struct {
	Type   *string                `yaml:"type,omitempty"`
	Config map[string]interface{} `yaml:",inline"`
}

type environment struct {
	Name  string      `yaml:"name"`
	Url   string      `yaml:"url"`
	Token tokenConfig `yaml:"token"`
}

type group struct {
	Group   string        `yaml:"group"`
	Entries []environment `yaml:"entries"`
}

type manifest struct {
	Projects     []project `yaml:"projects"`
	Environments []group   `yaml:"environments"`
}
