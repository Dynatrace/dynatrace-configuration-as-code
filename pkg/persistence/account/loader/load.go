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
	persisted, err := findAndLoadResources(fs, rootPath)
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

func findAndLoadResources(fs afero.Fs, rootPath string) (*persistence.Resources, error) {
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

		err = validateFile(*file)
		if err != nil {
			return nil, fmt.Errorf("invalid file %q: %w", yamlFilePath, err)
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

	var file persistence.File
	err = yaml.Unmarshal(bytes, &file)
	if err != nil {
		return nil, err
	}

	return &file, err
}

func validateFile(file persistence.File) error {
	for _, p := range file.Policies {
		if err := validatePolicy(p); err != nil {
			return err
		}
	}

	for _, g := range file.Groups {
		if err := validateGroup(g); err != nil {
			return err
		}
	}

	for _, u := range file.Users {
		if err := validateUser(u); err != nil {
			return err
		}
	}

	return nil
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
	return &account.Resources{
		Policies: transformPolicies(resources.Policies),
		Groups:   transformGroups(resources.Groups),
		Users:    transformUsers(resources.Users),
	}
}

func transformPolicies(in map[string]persistence.Policy) map[account.PolicyId]account.Policy {
	policies := make(map[account.PolicyId]account.Policy, len(in))
	for id, v := range in {
		policies[id] = account.Policy{
			ID:             v.ID,
			Name:           v.Name,
			Level:          transformLevel(v.Level),
			Description:    v.Description,
			Policy:         v.Policy,
			OriginObjectID: v.OriginObjectID,
		}
	}
	return policies
}

func transformLevel(level persistence.PolicyLevel) any {
	switch level.Type {
	case persistence.PolicyLevelAccount:
		return account.PolicyLevelAccount{Type: level.Type}
	case persistence.PolicyLevelEnvironment:
		return account.PolicyLevelEnvironment{Type: level.Type, Environment: level.Environment}
	default:
		panic("unable to convert persistence model")
	}
}

func transformGroups(in map[string]persistence.Group) map[account.GroupId]account.Group {
	groups := make(map[account.GroupId]account.Group, len(in))
	for id, v := range in {
		groups[id] = account.Group{
			ID:                       v.ID,
			Name:                     v.Name,
			Description:              v.Description,
			FederatedAttributeValues: v.FederatedAttributeValues,
			Account:                  transformAccount(v.Account),
			Environment:              transformEnvironments(v.Environment),
			ManagementZone:           transformManagementZones(v.ManagementZone),
			OriginObjectID:           v.OriginObjectID,
		}
	}
	return groups
}

func transformAccount(in *persistence.Account) *account.Account {
	if in == nil {
		return nil
	}

	return &account.Account{
		Permissions: in.Permissions,
		Policies:    transformReferences(in.Policies),
	}
}

func transformEnvironments(in []persistence.Environment) []account.Environment {
	env := make([]account.Environment, len(in))
	for i, e := range in {
		env[i] = account.Environment{
			Name:        e.Name,
			Permissions: e.Permissions,
			Policies:    transformReferences(e.Policies),
		}
	}
	return env
}

func transformManagementZones(in []persistence.ManagementZone) []account.ManagementZone {
	managementZones := make([]account.ManagementZone, len(in))
	for i, m := range in {
		managementZones[i] = account.ManagementZone{
			Environment:    m.Environment,
			ManagementZone: m.ManagementZone,
			Permissions:    m.Permissions,
		}
	}
	return managementZones
}

func transformUsers(in map[string]persistence.User) map[account.UserId]account.User {
	users := make(map[account.UserId]account.User, len(in))
	for id, v := range in {
		users[id] = account.User{
			Email:  v.Email,
			Groups: transformReferences(v.Groups),
		}
	}
	return users
}

func transformReferences(in []persistence.Reference) []account.Ref {
	res := make([]account.Ref, len(in))
	for i, el := range in {
		switch el.Type {
		case persistence.ReferenceType:
			res[i] = account.Reference{Id: el.Id}
		case "":
			res[i] = account.StrReference(el.Value)
		default:
			panic("unable to convert persistence model")
		}
	}
	return res
}
