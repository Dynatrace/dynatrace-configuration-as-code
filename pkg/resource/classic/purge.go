/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package classic

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

type PurgeSource interface {
	Delete(ctx context.Context, api api.API, id string) error
	List(ctx context.Context, api api.API) ([]dtclient.Value, error)
}

type Purger struct {
	configSource PurgeSource
	apisToPurge  api.APIs
}

func NewPurger(configSource PurgeSource, apisToPurge api.APIs) *Purger {
	return &Purger{configSource: configSource, apisToPurge: apisToPurge}
}

// DeleteAll collects and deletes all classic API configuration objects using the provided ConfigClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - client (dtclient.ConfigClient): An implementation of the ConfigClient interface for managing configuration objects.
//   - apis (api.APIs): A list of APIs for which configuration values need to be collected and deleted.
//
// Returns:
//   - error: After all deletions where attempted an error is returned if any attempt failed.
func (d Purger) DeleteAll(ctx context.Context) error {
	// This is a problem for unification, because the interface does not care for api.APIs passed. Most other clients don't care about that.

	errs := 0

	for _, a := range d.apisToPurge {
		logger := log.With(log.TypeAttr(a.ID))
		if a.HasParent() {
			logger.DebugContext(ctx, "Skipping %q, will be deleted by the parent api %q", a.ID, a.Parent)
		}
		logger.InfoContext(ctx, "Collecting configs of type %q...", a.ID)
		values, err := d.configSource.List(ctx, a)
		if err != nil {
			errs++
			continue
		}

		logger.InfoContext(ctx, "Deleting %d configs of type %q...", len(values), a.ID)

		for _, v := range values {
			logger := logger.With(slog.Any("value", v))
			logger.DebugContext(ctx, "Deleting config %s:%s...", a.ID, v.Id)
			err := d.configSource.Delete(ctx, a, v.Id)

			if err != nil {
				logger.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to delete %s with ID %s: %v", a.ID, v.Id, err)
				errs++
			}
		}
	}

	if errs > 0 {
		return fmt.Errorf("failed to delete %d config(s)", errs)
	}

	return nil
}
