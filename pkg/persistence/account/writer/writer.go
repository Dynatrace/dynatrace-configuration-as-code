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
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/internal/types"
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
// individual "policies.yaml", "users.yaml", "service-users.yaml" & "groups.yaml" files containing YAML representations of the given account.Resources.
//
// Returns an error if any step of transforming or persisting resources fails, but will attempt to write as many files as
// possible. If policies fail to be written to a file, an error is logged, but groups and users are attempted to be written
// to files before the method returns with an error.
func Write(writerContext Context, resources account.Resources) error {

	if err := createFolderIfNoneExists(writerContext.Fs, writerContext.OutputFolder); err != nil {
		return err
	}
	projectFolder := filepath.Join(writerContext.OutputFolder, writerContext.ProjectFolder)
	if err := createFolderIfNoneExists(writerContext.Fs, projectFolder); err != nil {
		return err
	}

	var errOccurred bool
	if len(resources.Policies) > 0 {
		policies := toPersistencePolicies(resources.Policies)
		if err := persistToFile(persistence.File{Policies: policies}, writerContext.Fs, filepath.Join(projectFolder, "policies.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist policies: %v", err)
		}
	}

	if len(resources.Groups) > 0 {
		groups := toPersistenceGroups(resources.Groups)
		if err := persistToFile(persistence.File{Groups: groups}, writerContext.Fs, filepath.Join(projectFolder, "groups.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist groups: %v", err)
		}
	}

	if len(resources.Users) > 0 {
		users := toPersistenceUsers(resources.Users)
		if err := persistToFile(persistence.File{Users: users}, writerContext.Fs, filepath.Join(projectFolder, "users.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist users: %v", err)
		}
	}

	if featureflags.ServiceUsers.Enabled() && len(resources.ServiceUsers) > 0 {
		serviceUsers := toPersistenceServiceUsers(resources.ServiceUsers)
		if err := persistToFile(persistence.File{ServiceUsers: serviceUsers}, writerContext.Fs, filepath.Join(projectFolder, "service-users.yaml")); err != nil {
			errOccurred = true
			log.Error("Failed to persist service users: %v", err)
		}
	}

	if errOccurred {
		return fmt.Errorf("failed to persist some account resources to folder %q", projectFolder)
	}

	log.WithFields(field.F("outputFolder", writerContext.OutputFolder)).Info("Downloaded account management resources written to '%s'", writerContext.OutputFolder)

	return nil
}

func toPersistencePolicies(policies map[string]account.Policy) []persistence.Policy {
	out := make([]persistence.Policy, 0, len(policies))
	for _, v := range policies {
		var level persistence.PolicyLevel
		switch tV := v.Level.(type) {
		case account.PolicyLevelAccount:
			level.Type = tV.Type
		case account.PolicyLevelEnvironment:
			level.Type = tV.Type
			level.Environment = tV.Environment
		}

		out = append(out, persistence.Policy{
			ID:             v.ID,
			Name:           v.Name,
			Level:          level,
			Description:    v.Description,
			Policy:         v.Policy,
			OriginObjectID: v.OriginObjectID,
		})
	}
	// sort policies by ID so that they are stable within a persisted file
	slices.SortFunc(out, func(a, b persistence.Policy) int {
		return caseInsensitiveLexicographicSmaller(a.ID, b.ID)
	})
	return out
}

func transformRefs(in []account.Ref) []persistence.Reference {
	var res []persistence.Reference
	// sort refs by ID() so that they are stable for both full refs and strings within a persisted file
	slices.SortFunc(in, func(a, b account.Ref) int {
		return caseInsensitiveLexicographicSmaller(a.ID(), b.ID())
	})
	for _, el := range in {
		switch v := el.(type) {
		case account.Reference:
			res = append(res, persistence.Reference{Type: persistence.ReferenceType, Id: v.Id})
		case account.StrReference:
			res = append(res, persistence.Reference{Value: string(v)})
		default:
			panic("unable to convert persistence model")
		}
	}
	return res
}

func toPersistenceGroups(groups map[string]account.Group) []persistence.Group {

	out := make([]persistence.Group, 0, len(groups))
	for _, v := range groups {
		var a *persistence.Account
		if v.Account != nil {
			a = &persistence.Account{
				Permissions: v.Account.Permissions,
				Policies:    transformRefs(v.Account.Policies),
			}
		}
		envs := make([]persistence.Environment, len(v.Environment))
		for i, e := range v.Environment {
			envs[i] = persistence.Environment{
				Name:        e.Name,
				Permissions: e.Permissions,
				Policies:    transformRefs(e.Policies),
			}
			// sort permissions so that they are stable within a persisted file
			slices.SortFunc(envs[i].Permissions, caseInsensitiveLexicographicSmaller)
		}
		// sort envs by name so that they are stable within a persisted file
		slices.SortFunc(envs, func(a, b persistence.Environment) int {
			return caseInsensitiveLexicographicSmaller(a.Name, b.Name)
		})
		mzs := make([]persistence.ManagementZone, len(v.ManagementZone))
		for i, e := range v.ManagementZone {
			mzs[i] = persistence.ManagementZone{
				Environment:    e.Environment,
				ManagementZone: e.ManagementZone,
				Permissions:    e.Permissions,
			}
			// sort permissions so that they are stable within a persisted file
			slices.SortFunc(mzs[i].Permissions, caseInsensitiveLexicographicSmaller)
		}
		// sort mzs by env and name so that they are stable within a persisted file
		slices.SortFunc(mzs, func(a, b persistence.ManagementZone) int {
			return caseInsensitiveLexicographicSmaller(a.Environment, b.Environment) + caseInsensitiveLexicographicSmaller(a.ManagementZone, b.ManagementZone)
		})

		out = append(out, persistence.Group{
			ID:                       v.ID,
			Name:                     v.Name,
			Description:              v.Description,
			FederatedAttributeValues: v.FederatedAttributeValues,
			Account:                  a,
			Environment:              envs,
			ManagementZone:           mzs,
			OriginObjectID:           v.OriginObjectID,
		})
	}
	// sort groups by ID so that they are stable within a persisted file
	slices.SortFunc(out, func(a, b persistence.Group) int {
		return caseInsensitiveLexicographicSmaller(a.ID, b.ID)
	})
	return out
}

func caseInsensitiveLexicographicSmaller(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

func toPersistenceUsers(users map[string]account.User) []persistence.User {
	out := make([]persistence.User, 0, len(users))
	for _, v := range users {
		out = append(out, persistence.User{
			Email:  v.Email,
			Groups: transformRefs(v.Groups),
		})
	}
	// sort users by email so that they are stable within a persisted file
	slices.SortFunc(out, func(a, b persistence.User) int {
		return caseInsensitiveLexicographicSmaller(a.Email.Value(), b.Email.Value())
	})
	return out
}

func toPersistenceServiceUsers(serviceUsers map[string]account.ServiceUser) []persistence.ServiceUser {
	out := make([]persistence.ServiceUser, 0, len(serviceUsers))
	for _, v := range serviceUsers {
		out = append(out, persistence.ServiceUser{
			Name:           v.Name,
			Description:    v.Description,
			Groups:         transformRefs(v.Groups),
			OriginObjectID: v.OriginObjectID,
		})
	}
	// sort service users by name so that they are stable within a persisted file
	slices.SortFunc(out, func(a, b persistence.ServiceUser) int {
		return caseInsensitiveLexicographicSmaller(a.Name, b.Name)
	})
	return out
}

func createFolderIfNoneExists(fs afero.Fs, path string) error {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return fmt.Errorf("failed to create folder to persist account resources: %w", err)
	}
	if exists {
		return nil
	}
	if err := fs.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to folder to persist account resources: %w", err)
	}
	return nil
}

func persistToFile(data persistence.File, fs afero.Fs, filepath string) error {
	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, filepath, b, 0644)
}
