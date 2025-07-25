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
	"log/slog"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
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
func Delete(ctx context.Context, environments manifest.EnvironmentDefinitionsByName, entriesToDelete delete.DeleteEntries) error {
	var envsWithDeleteErrs []string
	for _, env := range environments {
		ctx := context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})
		if containsPlatformTypes(entriesToDelete) && !env.HasPlatformCredentials() {
			slog.WarnContext(ctx, "Delete file contains entries referencing platform configurations, but environment is missing platform credentials. These configurations won't be deleted.")
		}

		clientSet, err := client.CreateClientSet(ctx, env.URL.Value, env.Auth)
		if err != nil {
			return fmt.Errorf("failed to create API client for environment %q due to the following error: %w", env.Name, err)
		}

		slog.InfoContext(ctx, "Deleting configs...")

		if err := delete.Configs(ctx, *clientSet, entriesToDelete); err != nil {
			slog.ErrorContext(ctx, "Failed to delete all configurations from environment - check log for details")
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
