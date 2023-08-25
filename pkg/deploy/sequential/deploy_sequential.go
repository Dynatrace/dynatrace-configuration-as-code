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

package sequential

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/entitymap"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/resolve"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/setting"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"golang.org/x/net/context"
)

// deprecation notice: This complete file can be dropped once graph-based parallel deployment becomes non-optional

// DeployConfigs sequentially deploys the given configs with the given apis to a single environment via the given client
// NOTE: the given configs need to be sorted, otherwise deployment will probably fail, as references cannot be resolved.
func DeployConfigs(clientSet deploy.ClientSet, apis api.APIs, sortedConfigs []config.Config, opts deploy.DeployConfigsOptions) []error {
	entityMap := entitymap.New()
	var errs []error

	for i := range sortedConfigs {
		c := &sortedConfigs[i] // avoid implicit memory aliasing (gosec G601)

		ctx := context.WithValue(context.TODO(), log.CtxKeyCoord{}, c.Coordinate)
		ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: c.Environment, Group: c.Group})

		entity, deploymentErrors := deployConfig(ctx, clientSet, apis, entityMap, c)

		if len(deploymentErrors) > 0 {
			for _, err := range deploymentErrors {
				errs = append(errs, fmt.Errorf("failed to deploy config %s: %w", c.Coordinate, err))
			}

			if !opts.ContinueOnErr && !opts.DryRun {
				return errs
			}
		} else {
			entityMap.Put(entity)
		}
	}

	return errs
}

func deployConfig(ctx context.Context, clientSet deploy.ClientSet, apis api.APIs, em *entitymap.EntityMap, c *config.Config) (config.ResolvedEntity, []error) {
	if c.Skip {
		log.WithCtxFields(ctx).Info("Skipping deployment of config %s", c.Coordinate)
		return config.ResolvedEntity{EntityName: c.Coordinate.ConfigId, Coordinate: c.Coordinate, Properties: parameter.Properties{}, Skip: true}, nil
	}

	properties, errs := resolve.Properties(c, em)
	if len(errs) > 0 {
		return config.ResolvedEntity{}, errs
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		return config.ResolvedEntity{}, []error{err}
	}

	log.WithCtxFields(ctx).Info("Deploying config")
	var res config.ResolvedEntity
	var deployErr error
	switch c.Type.(type) {
	case config.SettingsType:
		res, deployErr = setting.Deploy(ctx, clientSet.Settings, properties, renderedConfig, c)

	case config.ClassicApiType:
		res, deployErr = classic.Deploy(ctx, clientSet.Classic, apis, properties, renderedConfig, c)

	case config.AutomationType:
		res, deployErr = automation.Deploy(ctx, clientSet.Automation, properties, renderedConfig, c)

	case config.BucketType:
		res, deployErr = bucket.Deploy(ctx, clientSet.Bucket, properties, renderedConfig, c)

	default:
		deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
	}

	if deployErr != nil {
		var responseErr clientErrors.RespError
		if errors.As(deployErr, &responseErr) {
			log.WithCtxFields(ctx).WithFields(field.Error(responseErr)).Error("Failed to deploy config %s: %s", c.Coordinate, responseErr.Reason)
		} else {
			log.WithCtxFields(ctx).WithFields(field.Error(deployErr)).Error("Failed to deploy config %s: %s", c.Coordinate, deployErr.Error())
		}
		return config.ResolvedEntity{}, []error{deployErr}
	}
	return res, nil

}
