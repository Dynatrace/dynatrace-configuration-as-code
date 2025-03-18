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
	"path"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	accountLoader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/loader"
)

func loadResources(fs afero.Fs, workingDir string, projects manifest.ProjectDefinitionByProjectID) (*account.Resources, error) {
	resources := account.NewAccountManagementResources()
	for _, p := range projects {
		res, err := accountLoader.Load(fs, path.Join(workingDir, p.Path))
		if err != nil {
			return nil, err
		}
		for _, pol := range res.Policies {
			if _, exists := resources.Policies[pol.ID]; exists {
				return nil, fmt.Errorf("policy with id %q already defined in another project", pol.ID)
			}
			resources.Policies[pol.ID] = pol
		}

		for _, gr := range res.Groups {
			if _, exists := resources.Groups[gr.ID]; exists {
				return nil, fmt.Errorf("group with id %q already defined in another project", gr.ID)
			}
			resources.Groups[gr.ID] = gr
		}

		for _, us := range res.Users {
			if _, exists := resources.Users[us.Email.Value()]; exists {
				return nil, fmt.Errorf("user with id %q already defined in another project", us.Email)
			}
			resources.Users[us.Email.Value()] = us
		}

		if featureflags.ServiceUsers.Enabled() {
			for _, su := range res.ServiceUsers {
				for _, existingServiceUser := range resources.ServiceUsers {
					if areServiceUsersAmbiguous(su, existingServiceUser) {
						return nil, fmt.Errorf("found ambiguous service user with name %q", su.Name)
					}
				}
				resources.ServiceUsers = append(resources.ServiceUsers, su)
			}
		}
	}

	return resources, nil
}

// areServiceUsersAmbiguous returns true iff the two objects could refer to the same underlying service users.
func areServiceUsersAmbiguous(su1 account.ServiceUser, su2 account.ServiceUser) bool {
	// if they both have origin object ids that are the same they are ambiguous
	if (su1.OriginObjectID != "") && (su2.OriginObjectID != "") && (su1.OriginObjectID == su2.OriginObjectID) {
		return true
	}

	// if they have different names they are not ambiguous
	if su1.Name != su2.Name {
		return false
	}

	// if they have the same name but one or both are missing originObjectIds they are ambiguous
	if su1.OriginObjectID == "" || su2.OriginObjectID == "" {
		return true
	}

	// other combinations are OK
	return false
}
