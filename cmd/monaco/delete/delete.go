// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/support"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

// Delete removes configurations from multiple Dynatrace environments based on the specified deletion entries.
//
// Parameters:
//   - environments: A list of Dynatrace environments to perform the deletion on.
//   - entriesToDelete: Deletion entries specifying what configurations to remove.
//
// Returns:
//   - error: If an error occurs during the deletion process, an error is returned, describing the issue.
//     If no errors occur, nil is returned.
func Delete(environments manifest.Environments, entriesToDelete delete.DeleteEntries) error {
	var envsWithDeleteErrs []string
	for _, env := range environments {
		ctx := context.WithValue(context.TODO(), log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})
		if containsPlatformTypes(entriesToDelete) && env.Auth.OAuth == nil {
			log.WithCtxFields(ctx).Warn("Delete file contains Dynatrace Platform specific types, but no oAuth credentials are defined for environment %q - Dynatrace Platform configurations won't be deleted.", env.Name)
		}

		clientSet, err := client.CreateClientSet(ctx, env.URL.Value, env.Auth, client.ClientOptions{SupportArchive: support.SupportArchive})
		if err != nil {
			return fmt.Errorf("failed to create API client for environment %q due to the following error: %w", env.Name, err)
		}

		log.WithCtxFields(ctx).Info("Deleting configs for environment %q...", env.Name)

		classicAPIs := api.NewAPIs()
		automationAPIs := map[string]config.AutomationResource{
			string(config.Workflow):         config.Workflow,
			string(config.BusinessCalendar): config.BusinessCalendar,
			string(config.SchedulingRule):   config.SchedulingRule,
		}

		if err := delete.Configs(ctx, *clientSet, classicAPIs, automationAPIs, entriesToDelete); err != nil {
			log.Error("Failed to delete all configurations from environment %q - check log for details", env.Name)
			envsWithDeleteErrs = append(envsWithDeleteErrs, env.Name)
		}
	}

	if len(envsWithDeleteErrs) > 0 {
		return fmt.Errorf("encountered deletion errors for the following environments: %v", strings.Join(envsWithDeleteErrs, ", "))
	}
	return nil
}

func containsPlatformTypes(entriesToDelete delete.DeleteEntries) bool {
	for _, t := range []string{string(config.Workflow), string(config.SchedulingRule), string(config.BusinessCalendar), "bucket"} {
		if _, contains := entriesToDelete[t]; contains {
			return true
		}
	}
	return false
}
