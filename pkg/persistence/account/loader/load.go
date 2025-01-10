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

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/internal/types"
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
		return nil, fmt.Errorf("failed to load account management resources from %q: %w", rootPath, err)
	}

	if err := validateReferences(persisted); err != nil {
		return nil, fmt.Errorf("account management resources from %q are invalid: %w", rootPath, err)
	}

	return transform(persisted), nil
}

// HasAnyAccountKeyDefined checks whether the map has any AM key defined.
// The current keys are `users`, `groups`, and `policies`.
func HasAnyAccountKeyDefined(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}

	return m[persistence.KeyUsers] != nil || m[persistence.KeyGroups] != nil || m[persistence.KeyPolicies] != nil
}

func load(fs afero.Fs, rootPath string) (*persistence.Resources, error) {
	resources := persistence.Resources{
		Policies: make(map[string]persistence.Policy),
		Groups:   make(map[string]persistence.Group),
		Users:    make(map[string]persistence.User),
	}

	yamlFilePaths, err := files.FindYamlFiles(fs, rootPath)
	if err != nil {
		return nil, err
	}

	for _, yamlFilePath := range yamlFilePaths {
		log.WithFields(field.F("file", yamlFilePaths)).Debug("Loading file %q", yamlFilePath)

		file, err := loadFile(fs, yamlFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load file %q: %w", yamlFilePath, err)
		}

		err = addResourcesFromFile(resources, *file)
		if err != nil {
			return nil, fmt.Errorf("failed to add resources from file %q: %w", yamlFilePath, err)
		}
	}
	return &resources, nil
}

func loadFile(fs afero.Fs, yamlFilePath string) (*persistence.File, error) {
	log.WithFields(field.F("file", yamlFilePath)).Debug("Loading file %q", yamlFilePath)

	bytes, err := afero.ReadFile(fs, yamlFilePath)
	if err != nil {
		return nil, err
	}

	var content map[string]any
	if err := yaml.Unmarshal(bytes, &content); err != nil {
		return nil, err
	}

	if _, f := content["configs"]; f {
		if HasAnyAccountKeyDefined(content) {
			return nil, ErrMixingConfigs
		}

		log.WithFields(field.F("file", yamlFilePath)).Warn("File %q appears to be an config file, skipping loading", yamlFilePath)
		return &persistence.File{}, nil
	}

	if _, f := content["delete"]; f {
		if HasAnyAccountKeyDefined(content) {
			return nil, ErrMixingDelete
		}

		log.WithFields(field.F("file", yamlFilePath)).Debug("File %q appears to be an delete file, skipping loading", yamlFilePath)
		return &persistence.File{}, nil
	}

	var res persistence.File
	err = yaml.Unmarshal(bytes, &res)
	if err != nil {
		return nil, err
	}

	for _, p := range res.Policies {
		if err := validatePolicy(p); err != nil {
			return nil, err
		}
	}

	for _, g := range res.Groups {
		if err := validateGroup(g); err != nil {
			return nil, err
		}
	}

	for _, u := range res.Users {
		if err := validateUser(u); err != nil {
			return nil, err
		}
	}

	return &res, nil
}

func addResourcesFromFile(res persistence.Resources, file persistence.File) error {
	for _, p := range file.Policies {
		if _, exists := res.Policies[p.ID]; exists {
			return fmt.Errorf("found duplicate policy with id %q", p.ID)
		}
		res.Policies[p.ID] = p
	}

	for _, g := range file.Groups {
		if _, exists := res.Groups[g.ID]; exists {
			return fmt.Errorf("found duplicate group with id %q", g.ID)
		}
		res.Groups[g.ID] = g
	}

	for _, u := range file.Users {
		if _, exists := res.Users[u.Email.Value()]; exists {
			return fmt.Errorf("found duplicate user with email %q", u.Email)
		}
		res.Users[u.Email.Value()] = u
	}

	return nil
}

func transform(resources *persistence.Resources) *account.Resources {
	transformLevel := func(level persistence.PolicyLevel) any {
		switch level.Type {
		case persistence.PolicyLevelAccount:
			return account.PolicyLevelAccount{Type: level.Type}
		case persistence.PolicyLevelEnvironment:
			return account.PolicyLevelEnvironment{Type: level.Type, Environment: level.Environment}
		default:
			panic("unable to convert persistence model")
		}
	}

	transformRefs := func(in []persistence.Reference) []account.Ref {
		var res []account.Ref
		for _, el := range in {
			switch el.Type {
			case persistence.ReferenceType:
				res = append(res, account.Reference{Id: el.Id})
			case "":
				res = append(res, account.StrReference(el.Value))
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
			ID:             v.ID,
			Name:           v.Name,
			Level:          transformLevel(v.Level),
			Description:    v.Description,
			Policy:         v.Policy,
			OriginObjectID: v.OriginObjectID,
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
			ID:                       v.ID,
			Name:                     v.Name,
			Description:              v.Description,
			FederatedAttributeValues: v.FederatedAttributeValues,
			Account:                  acc,
			Environment:              env,
			ManagementZone:           mz,
			OriginObjectID:           v.OriginObjectID,
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
