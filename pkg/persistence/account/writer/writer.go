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

package writer

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path/filepath"
)

// Context for this account resource writer, defining the filesystem and paths to create resources at
type Context struct {
	// Fs to use when writing files
	Fs afero.Fs
	// OutputFolder to write an account resources ProjectFolder into. If this is not an absolute path, Write will transform it into one using filepath.Abs.
	OutputFolder string
	// ProjectFolder to create and fill with account resources YAML files
	ProjectFolder string
}

// Write the given account.Resources to the target filesystem and paths defined by the Context.
// This will create a folder "filepath.Abs(<writerContext.OutputFolder>)/<writerContext.ProjectFolder>/", and create
// individual "policies.yaml", "users.yaml" & "groups.yaml" files containing YAML representations of the given account.Resources.
//
// Returns an error if any step of transforming or persisting resources fails, but will attempt to write as many files as
// possible. If policies fail to be written to a file, an error is logged, but groups and users are attempted to be written
// to files before the method returns with an error.
func Write(writerContext Context, resources account.Resources) error {

	outputFolder, err := filepath.Abs(writerContext.OutputFolder)
	if err != nil {
		return err
	}
	if err := createFolderIfNoneExists(writerContext.Fs, outputFolder); err != nil {
		return err
	}
	projectFolder := filepath.Join(outputFolder, writerContext.ProjectFolder)
	if err := createFolderIfNoneExists(writerContext.Fs, projectFolder); err != nil {
		return err
	}

	var errOccurred bool
	if len(resources.Policies) > 0 {
		policies := toPersistencePolicies(resources.Policies)
		if err := persistToFile(policies, writerContext.Fs, filepath.Join(projectFolder, "policies.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist policies: %w", err)
		}
	}

	if len(resources.Groups) > 0 {
		groups := toPersistenceGroups(resources.Groups)
		if err := persistToFile(groups, writerContext.Fs, filepath.Join(projectFolder, "groups.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist groups: %w", err)
		}
	}

	if len(resources.Users) > 0 {
		users := toPersistenceUsers(resources.Users)
		if err := persistToFile(users, writerContext.Fs, filepath.Join(projectFolder, "users.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist users: %w", err)
		}
	}

	if errOccurred {
		return fmt.Errorf("failed to persist some account resources to folder %q", projectFolder)
	}
	return nil
}

func toPersistencePolicies(policies map[string]account.Policy) persistence.Policies {
	out := make([]persistence.Policy, 0, len(policies))
	for _, v := range policies {
		out = append(out, persistence.Policy{
			ID:             v.ID,
			Name:           v.Name,
			Level:          v.Level,
			Description:    v.Description,
			Policy:         v.Policy,
			OriginObjectID: v.OriginObjectID,
		})
	}
	return persistence.Policies{
		Policies: out,
	}
}

func toPersistenceGroups(groups map[string]account.Group) persistence.Groups {
	out := make([]persistence.Group, 0, len(groups))
	for _, v := range groups {
		a := persistence.Account{
			Permissions: v.Account.Permissions,
			Policies:    v.Account.Policies,
		}
		envs := make([]persistence.Environment, len(v.Environment))
		for i, e := range v.Environment {
			envs[i] = persistence.Environment{
				Name:        e.Name,
				Permissions: e.Permissions,
				Policies:    e.Policies,
			}
		}
		mzs := make([]persistence.ManagementZone, len(v.ManagementZone))
		for i, e := range v.ManagementZone {
			mzs[i] = persistence.ManagementZone{
				Environment:    e.Environment,
				ManagementZone: e.ManagementZone,
				Permissions:    e.Permissions,
			}
		}

		out = append(out, persistence.Group{
			ID:             v.ID,
			Name:           v.Name,
			Description:    v.Description,
			Account:        &a,
			Environment:    envs,
			ManagementZone: mzs,
			OriginObjectID: v.OriginObjectID,
		})
	}
	return persistence.Groups{
		Groups: out,
	}
}

func toPersistenceUsers(users map[string]account.User) persistence.Users {
	out := make([]persistence.User, 0, len(users))
	for _, v := range users {
		out = append(out, persistence.User{
			Email:  v.Email,
			Groups: v.Groups,
		})
	}
	return persistence.Users{
		Users: out,
	}
}

func createFolderIfNoneExists(fs afero.Fs, path string) error {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return fmt.Errorf("failed to create folder to persist account resources: %w", err)
	}
	if exists {
		return nil
	}
	if err := fs.MkdirAll(path, 0644); err != nil {
		return fmt.Errorf("failed to folder to persist account resources: %w", err)
	}
	return nil
}

func persistToFile(data any, fs afero.Fs, filepath string) error {
	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, filepath, b, 0644)
}
