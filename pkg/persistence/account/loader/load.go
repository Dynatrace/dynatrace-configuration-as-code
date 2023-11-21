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
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/internal/types"
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

func load(fs afero.Fs, rootPath string) (*persistence.Resources, error) {
	resources := &persistence.Resources{
		Policies: make(map[string]persistence.Policy),
		Groups:   make(map[string]persistence.Group),
		Users:    make(map[string]persistence.User),
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

		if err := loadPolicies(content, resources); err != nil {
			return nil, fmt.Errorf("failed to load policies for file %q: %w", yamlFilePath, err)
		}

		if err := loadGroups(content, resources); err != nil {
			return nil, fmt.Errorf("failed to load groups for file %q: %w", yamlFilePath, err)
		}

		if err := loadUsers(content, resources); err != nil {
			return nil, fmt.Errorf("failed to load users for file %q: %w", yamlFilePath, err)
		}
	}
	return resources, nil
}

func loadPolicies(content map[string]any, resources *persistence.Resources) error {
	var policies persistence.Policies
	err := mapstructure.Decode(content, &policies)
	if err != nil {
		return err
	}
	for _, pol := range policies.Policies {
		if _, exists := resources.Policies[pol.ID]; exists {
			return fmt.Errorf("found duplicate policy with id %q", pol.ID)
		}
		type PolicyLevel struct {
			Type        string `mapstructure:"type"`
			Environment string `mapstructure:"environment"`
		}
		var level PolicyLevel
		if err := mapstructure.Decode(pol.Level, &level); err != nil {
			return err
		}

		if level.Type == "account" {
			pol.Level = persistence.PolicyLevelAccount{Type: "account"}
		}
		if level.Type == "environment" {
			pol.Level = persistence.PolicyLevelEnvironment{
				Type:        "environment",
				Environment: level.Environment,
			}
		}
		resources.Policies[pol.ID] = pol
	}
	return nil
}

func loadGroups(content map[string]any, resources *persistence.Resources) error {
	var groups persistence.Groups
	err := mapstructure.Decode(content, &groups)
	if err != nil {
		return err
	}
	for _, gr := range groups.Groups {
		if _, exists := resources.Groups[gr.ID]; exists {
			return fmt.Errorf("found duplicate group with id %q", gr.ID)
		}

		if gr.Account != nil {
			typedPolicies, err := parsePolicies(gr.Account.Policies)
			if err != nil {
				return err
			}
			gr.Account.Policies = typedPolicies
		}

		if gr.Environment != nil {
			var typedEnvs []persistence.Environment
			for _, env := range gr.Environment {
				typedPolicies, err := parsePolicies(env.Policies)
				if err != nil {
					return err
				}
				env.Policies = typedPolicies
				typedEnvs = append(typedEnvs, env)
			}
			gr.Environment = typedEnvs
		}
		resources.Groups[gr.ID] = gr
	}

	return nil
}

func parsePolicies(untypedPolicies []any) (typedPolicies []any, err error) {
	for _, pol := range untypedPolicies {
		if polStr, ok := pol.(string); ok {
			typedPolicies = append(typedPolicies, polStr)
			continue
		}
		var reference persistence.Reference
		if err = mapstructure.Decode(pol, &reference); err != nil {
			return nil, err
		}
		typedPolicies = append(typedPolicies, reference)
	}
	return typedPolicies, nil
}

func loadUsers(content map[string]any, resources *persistence.Resources) error {
	var users persistence.Users
	err := mapstructure.Decode(content, &users)
	if err != nil {
		return err
	}
	for _, us := range users.Users {
		if _, exists := resources.Users[us.Email]; exists {
			return fmt.Errorf("found duplicate user with id %q", us.Email)
		}

		typedGroups := make([]any, 0)
		for _, gr := range us.Groups {
			if grStr, ok := gr.(string); ok {
				typedGroups = append(typedGroups, grStr)
				continue
			}
			var reference persistence.Reference
			if err = mapstructure.Decode(gr, &reference); err != nil {
				return err
			}
			typedGroups = append(typedGroups, reference)
		}
		us.Groups = typedGroups
		resources.Users[us.Email] = us
	}
	return nil
}

func decode(in io.ReadCloser) (map[string]any, error) {
	defer func(r io.ReadCloser) { _ = r.Close() }(in)
	var content map[string]interface{}
	if err := yaml.NewDecoder(in).Decode(&content); err != nil {
		return content, err
	}
	return content, nil
}

func transform(resources *persistence.Resources) *account.Resources {
	transformLevel := func(levelType any) any {
		switch v := levelType.(type) {
		case persistence.PolicyLevelAccount:
			return account.PolicyLevelAccount(v)
		case persistence.PolicyLevelEnvironment:
			return account.PolicyLevelEnvironment(v)
		default:
			panic("unable to convert persistence model")
		}
	}

	transformRefs := func(in []any) []account.Ref {
		var res []account.Ref
		for _, el := range in {
			switch v := el.(type) {
			case persistence.Reference:
				res = append(res, account.Reference(v))
			case string:
				res = append(res, account.StrReference(v))
			default:
				panic("unable to convert persistence model")
			}
		}
		return res
	}

	inMemResources := account.Resources{
		Policies: make(map[account.PolicyId]account.Policy),
		Groups:   make(map[account.GroupId]account.Group),
		Users:    make(map[account.UserId]account.User),
	}
	for id, v := range resources.Policies {
		inMemResources.Policies[id] = account.Policy{
			ID:          v.ID,
			Name:        v.Name,
			Level:       transformLevel(v.Level),
			Description: v.Description,
			Policy:      v.Policy,
		}
	}
	for id, v := range resources.Groups {
		var acc *account.Account
		if v.Account != nil {
			acc = &account.Account{
				Permissions: v.Account.Permissions,
				Policies:    transformRefs(v.Account.Policies),
			}
		}
		env := make([]account.Environment, len(v.Environment))
		for i, e := range v.Environment {
			env[i] = account.Environment{
				Name:        e.Name,
				Permissions: e.Permissions,
				Policies:    transformRefs(e.Policies),
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
			Groups: transformRefs(v.Groups),
		}
	}
	return &inMemResources
}
