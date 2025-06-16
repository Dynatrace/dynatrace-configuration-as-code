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
	"errors"
	"fmt"
	"path"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/internal/types"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

var ErrMixingConfigs = errors.New("mixing both configurations and account resources is not allowed")
var ErrMixingDelete = errors.New("mixing account resources with a delete file definition is not allowed")

// LoadResources loads and merges resources from the specified projects assumed to be located within the specified working directory.
func LoadResources(fs afero.Fs, workingDir string, projects manifest.ProjectDefinitionByProjectID) (*account.Resources, error) {
	allResources := account.NewAccountManagementResources()
	for _, p := range projects {
		projectResources, err := Load(fs, path.Join(workingDir, p.Path))
		if err != nil {
			return nil, fmt.Errorf("unable to load resources from project '%s': %w", p.Name, err)
		}

		if err := addProjectResources(allResources, projectResources); err != nil {
			return nil, fmt.Errorf("unable to add resources from project '%s': %w", p.Name, err)
		}
	}

	return allResources, nil
}

func addProjectResources(targetResources *account.Resources, sourceResources *account.Resources) error {
	for _, pol := range sourceResources.Policies {
		if _, exists := targetResources.Policies[pol.ID]; exists {
			return fmt.Errorf("policy with id '%s' already defined in another project", pol.ID)
		}
		targetResources.Policies[pol.ID] = pol
	}

	for _, gr := range sourceResources.Groups {
		if _, exists := targetResources.Groups[gr.ID]; exists {
			return fmt.Errorf("group with id '%s' already defined in another project", gr.ID)
		}
		targetResources.Groups[gr.ID] = gr
	}

	for _, us := range sourceResources.Users {
		if _, exists := targetResources.Users[us.Email.Value()]; exists {
			return fmt.Errorf("user with email '%s' already defined in another project", us.Email)
		}
		targetResources.Users[us.Email.Value()] = us
	}

	if featureflags.ServiceUsers.Enabled() {
		for _, su := range sourceResources.ServiceUsers {
			for _, existingServiceUser := range targetResources.ServiceUsers {
				if err := verifyServiceUsersAreNotAmbiguous(su, existingServiceUser); err != nil {
					return err
				}
			}
			targetResources.ServiceUsers = append(targetResources.ServiceUsers, su)
		}
	}
	return nil
}

// Load loads account management resources from YAML configuration files
// located within the specified root directory path.
// It:
//  1. parses YAML files found under rootPath, extracts policies, groups, users and service users
//  2. validates the loaded data for correct syntax
//  3. returns the data in the in-memory account.Resources representation
func Load(fs afero.Fs, rootPath string) (*account.Resources, error) {
	resources, err := findAndLoadResources(fs, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load account management resources from %q: %w", rootPath, err)
	}

	if err := validateReferences(resources); err != nil {
		return nil, fmt.Errorf("account management resources from %q are invalid: %w", rootPath, err)
	}

	return resources, nil
}

// HasAnyAccountKeyDefined checks whether the map has any AM key defined.
// The current keys are `users`, `serviceUsers`, `groups`, and `policies`.
func HasAnyAccountKeyDefined(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}

	return m[persistence.KeyUsers] != nil || m[persistence.KeyServiceUsers] != nil || m[persistence.KeyGroups] != nil || m[persistence.KeyPolicies] != nil
}

func findAndLoadResources(fs afero.Fs, rootPath string) (*account.Resources, error) {
	resources := account.Resources{
		Policies:     make(map[string]account.Policy),
		Groups:       make(map[string]account.Group),
		Users:        make(map[string]account.User),
		ServiceUsers: make([]account.ServiceUser, 0),
	}

	yamlFilePaths, err := files.FindYamlFiles(fs, rootPath)
	if err != nil {
		return nil, err
	}

	for _, yamlFilePath := range yamlFilePaths {
		file, err := loadFile(fs, yamlFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load file %q: %w", yamlFilePath, err)
		}

		err = validateFile(*file)
		if err != nil {
			return nil, fmt.Errorf("invalid file %q: %w", yamlFilePath, err)
		}

		err = addResourcesFromFile(&resources, *file)
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

func addResourcesFromFile(res *account.Resources, file persistence.File) error {
	for _, p := range file.Policies {
		if _, exists := res.Policies[p.ID]; exists {
			return fmt.Errorf("found duplicate policy with id %q", p.ID)
		}
		res.Policies[p.ID] = transformPolicy(p)
	}

	for _, g := range file.Groups {
		if _, exists := res.Groups[g.ID]; exists {
			return fmt.Errorf("found duplicate group with id %q", g.ID)
		}
		res.Groups[g.ID] = transformGroup(g)
	}

	for _, u := range file.Users {
		if _, exists := res.Users[u.Email.Value()]; exists {
			return fmt.Errorf("found duplicate user with email %q", u.Email)
		}
		res.Users[u.Email.Value()] = transformUser(u)
	}

	if featureflags.ServiceUsers.Enabled() {
		for _, su := range file.ServiceUsers {
			serviceUser := transformServiceUser(su)
			for _, existingServiceUser := range res.ServiceUsers {
				if err := verifyServiceUsersAreNotAmbiguous(existingServiceUser, serviceUser); err != nil {
					return err
				}
			}
			res.ServiceUsers = append(res.ServiceUsers, serviceUser)
		}
	}

	return nil
}

// verifyServiceUsersAreNotAmbiguous returns an error iff the two objects could refer to the same underlying service users.
func verifyServiceUsersAreNotAmbiguous(su1 account.ServiceUser, su2 account.ServiceUser) error {
	// if they both have origin object ids that are the same they are ambiguous
	if (su1.OriginObjectID != "") && (su2.OriginObjectID != "") && (su1.OriginObjectID == su2.OriginObjectID) {
		return fmt.Errorf("multiple service users with the same originObjectId '%s'", su1.OriginObjectID)
	}

	// if they have different names they are not ambiguous
	if su1.Name != su2.Name {
		return nil
	}

	// if they have the same name but one or both are missing originObjectIds they are ambiguous
	if su1.OriginObjectID == "" || su2.OriginObjectID == "" {
		return fmt.Errorf("multiple service users with name '%s' but at least one is without originObjectId", su1.Name)
	}

	// other combinations are OK
	return nil
}
