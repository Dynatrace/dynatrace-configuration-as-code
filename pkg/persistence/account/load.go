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

package account

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"io"
)

// Load loads account management resources from YAML configuration files
// located within the specified root directory path. It parses the YAML files, extracts policies,
// groups, and users data, and organizes them into a AMResources struct, which is then returned.
func Load(fs afero.Fs, rootPath string) (*AMResources, error) {
	resources := &AMResources{
		Policies: make([]Policy, 0),
		Groups:   make([]Group, 0),
		Users:    make([]User, 0),
	}

	yamlFilePaths, err := files.FindYamlFiles(fs, rootPath)
	if err != nil {
		return nil, err
	}

	for _, yamlFilePath := range yamlFilePaths {
		yamlFile, err := fs.Open(yamlFilePath)
		if err != nil {
			return nil, err
		}
		content, err := decode(yamlFile)
		if err != nil {
			return nil, err
		}

		var policies Policies
		err = mapstructure.Decode(content, &policies)
		if err != nil {
			return nil, err
		}
		resources.Policies = append(resources.Policies, policies.Policies...)

		var groups Groups
		err = mapstructure.Decode(content, &groups)
		if err != nil {
			return nil, err
		}
		resources.Groups = append(resources.Groups, groups.Groups...)

		var users Users
		err = mapstructure.Decode(content, &users)
		if err != nil {
			return nil, err
		}
		resources.Users = append(resources.Users, users.Users...)
	}

	return resources, nil
}

func decode(in io.ReadCloser) (map[string]any, error) {
	defer in.Close()
	var content map[string]interface{}
	if err := yaml.NewDecoder(in).Decode(&content); err != nil {
		return content, err
	}
	return content, nil
}
