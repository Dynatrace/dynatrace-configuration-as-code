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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persitence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/internal/types"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"io"
)

// Load loads account management resources from YAML configuration files
// located within the specified root directory path.
// It:
//  1. parses YAML files found under rootPath, extracts policies, groups, and users data
//  2. validates the loaded data for correct syntax
//  3. returns the data in the in-memory account.Resources representation
func Load(fs afero.Fs, rootPath string) (*account.Resources, error) {
	persisted, err := load(fs, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load account managment resources from %s: %w", rootPath, err)
	}

	if err := validate(persisted); err != nil {
		return nil, fmt.Errorf("account managment resources from %s are invalid: %w", rootPath, err)
	}

	return transform(persisted), nil
}

func load(fs afero.Fs, rootPath string) (*persitence.Resources, error) {
	resources := &persitence.Resources{
		Policies: make(map[string]persitence.Policy),
		Groups:   make(map[string]persitence.Group),
		Users:    make(map[string]persitence.User),
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

		var policies persitence.Policies
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
				pol.Level = persitence.PolicyLevelAccount{Type: "account"}
			}
			if level.Type == "environment" {
				pol.Level = persitence.PolicyLevelEnvironment{
					Type:        "environment",
					Environment: level.Environment,
				}
			}
			resources.Policies[pol.ID] = pol
		}

		var groups persitence.Groups
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
					var reference persitence.Reference
					if err = mapstructure.Decode(pol, &reference); err != nil {
						return nil, err
					}
					typedPolicies = append(typedPolicies, reference)
				}
				gr.Account.Policies = typedPolicies
			}

			if gr.Environment != nil {
				var typedEnvs []persitence.Environment
				for _, env := range gr.Environment {
					var typedPolicies []any
					for _, pol := range env.Policies {
						if polStr, ok := pol.(string); ok {
							typedPolicies = append(typedPolicies, polStr)
							continue
						}
						var reference persitence.Reference
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

		var users persitence.Users
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
				var reference persitence.Reference
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

func transform(resources *persitence.Resources) *account.Resources {
	inMemResources := account.Resources{
		Policies: make(map[account.PolicyId]account.Policy),
		Groups:   make(map[account.GroupId]account.Group),
		Users:    make(map[account.UserId]account.User),
	}
	for id, v := range resources.Policies {
		inMemResources.Policies[id] = account.Policy{
			ID:          v.ID,
			Name:        v.Name,
			Level:       v.Level,
			Description: v.Description,
			Policy:      v.Policy,
		}
	}
	for id, v := range resources.Groups {
		var acc *account.Account
		if v.Account != nil {
			acc = &account.Account{
				Permissions: v.Account.Permissions,
				Policies:    v.Account.Policies,
			}
		}
		env := make([]account.Environment, len(v.Environment))
		for i, e := range v.Environment {
			env[i] = account.Environment{
				Name:        e.Name,
				Permissions: e.Permissions,
				Policies:    e.Policies,
			}
		}
		mz := make([]account.ManagementZone, len(v.ManagementZone))
		for i, m := range v.ManagementZone {
			mz[i] = account.ManagementZone{
				Environment:    m.Environment,
				ManagementZone: m.ManagementZone,
				Permissions:    m.Permissions,
			}
		}
		inMemResources.Groups[id] = account.Group{
			ID:             v.ID,
			Name:           v.Name,
			Description:    v.Description,
			Account:        acc,
			Environment:    env,
			ManagementZone: mz,
		}
	}
	for id, v := range resources.Users {
		inMemResources.Users[id] = account.User{
			Email:  v.Email,
			Groups: v.Groups,
		}
	}
	return &inMemResources
}
