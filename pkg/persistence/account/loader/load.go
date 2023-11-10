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

package loader

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"io"
)

// Load loads account management resources from YAML configuration files
// located within the specified root directory path. It parses the YAML files, extracts policies,
// groups, and users data, and organizes them into a AMResources struct, which is then returned.
func Load(fs afero.Fs, rootPath string) (*account.AMResources, error) {
	resources := &account.AMResources{
		Policies: make(map[string]account.Policy),
		Groups:   make(map[string]account.Group),
		Users:    make(map[string]account.User),
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

		var policies account.Policies
		err = mapstructure.Decode(content, &policies)
		if err != nil {
			return nil, err
		}
		for _, pol := range policies.Policies {
			if _, exists := resources.Policies[pol.ID]; exists {
				return nil, fmt.Errorf("found duplicate policy with id %q", pol.ID)
			}
			type PolicyLevel struct {
				Type        string `mapstructure:"type"`
				Environment string `mapstructure:"environment"`
			}
			var level PolicyLevel
			if err := mapstructure.Decode(pol.Level, &level); err != nil {
				return nil, err
			}

			if level.Type == "account" {
				pol.Level = account.PolicyLevelAccount{Type: "account"}
			}
			if level.Type == "environment" {
				pol.Level = account.PolicyLevelEnvironment{
					Type:        "environment",
					Environment: level.Environment,
				}
			}
			resources.Policies[pol.ID] = pol
		}

		var groups account.Groups
		err = mapstructure.Decode(content, &groups)
		if err != nil {
			return nil, err
		}
		for _, gr := range groups.Groups {
			if _, exists := resources.Groups[gr.ID]; exists {
				return nil, fmt.Errorf("found duplicate group with id %q", gr.ID)
			}

			//---  parsing refs
			if gr.Account != nil {
				var typedPolicies []any
				for _, pol := range gr.Account.Policies {
					if polStr, ok := pol.(string); ok {
						typedPolicies = append(typedPolicies, polStr)
						continue
					}
					var reference account.Reference
					if err = mapstructure.Decode(pol, &reference); err != nil {
						return nil, err
					}
					typedPolicies = append(typedPolicies, reference)
				}
				gr.Account.Policies = typedPolicies
			}

			if gr.Environment != nil {
				var typedEnvs []account.Environment
				for _, env := range gr.Environment {
					var typedPolicies []any
					for _, pol := range env.Policies {
						if polStr, ok := pol.(string); ok {
							typedPolicies = append(typedPolicies, polStr)
							continue
						}
						var reference account.Reference
						if err = mapstructure.Decode(pol, &reference); err != nil {
							return nil, err
						}
						typedPolicies = append(typedPolicies, reference)
					}
					env.Policies = typedPolicies
					typedEnvs = append(typedEnvs, env)
				}
				gr.Environment = typedEnvs
			}
			resources.Groups[gr.ID] = gr
		}

		var users account.Users
		err = mapstructure.Decode(content, &users)
		if err != nil {
			return nil, err
		}
		for _, us := range users.Users {
			if _, exists := resources.Users[us.Email]; exists {
				return nil, fmt.Errorf("found duplicate user with id %q", us.Email)
			}

			typedGroups := make([]any, 0)
			for _, gr := range us.Groups {
				if grStr, ok := gr.(string); ok {
					typedGroups = append(typedGroups, grStr)
					continue
				}
				var reference account.Reference
				if err = mapstructure.Decode(gr, &reference); err != nil {
					return nil, err
				}
				typedGroups = append(typedGroups, reference)
			}
			us.Groups = typedGroups
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
