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
	"fmt"
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
		Policies: make(map[string]Policy, 0),
		Groups:   make(map[string]Group, 0),
		Users:    make(map[string]User, 0),
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
		for _, pol := range policies.Policies {
			if _, exists := resources.Policies[pol.ID]; exists {
				return nil, fmt.Errorf("found duplicate policy with id %q", pol.ID)
			}
			resources.Policies[pol.ID] = pol
		}

		var groups Groups
		err = mapstructure.Decode(content, &groups)
		if err != nil {
			return nil, err
		}
		for _, gr := range groups.Groups {
			if _, exists := resources.Groups[gr.ID]; exists {
				return nil, fmt.Errorf("found duplicate group with id %q", gr.ID)
			}
			resources.Groups[gr.ID] = gr
		}

		var users Users
		err = mapstructure.Decode(content, &users)
		if err != nil {
			return nil, err
		}
		for _, us := range users.Users {
			if _, exists := resources.Users[us.Email]; exists {
				return nil, fmt.Errorf("found duplicate user with id %q", us.Email)
			}
			resources.Users[us.Email] = us
		}
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
