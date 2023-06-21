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

package deploy

import (
	"context"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/parameter"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
)

// DeployConfigsOptions defines additional options used by DeployConfigs
type DeployConfigsOptions struct {
	// ContinueOnErr states that the deployment continues even when there happens to be an
	// error while deploying a certain configuration
	ContinueOnErr bool
	// DryRun states that the deployment shall just run in dry-run mode, meaning
	// that actual deployment of the configuration to a tenant will be skipped
	DryRun bool
}

type ClientSet struct {
	Classic    dtclient.Client
	Settings   dtclient.Client
	Automation automationClient
}

var DummyClientSet = ClientSet{
	Classic:    &dtclient.DummyClient{},
	Settings:   &dtclient.DummyClient{},
	Automation: &dummyAutomationClient{},
}

// DeployConfigs deploys the given configs with the given apis via the given client
// NOTE: the given configs need to be sorted, otherwise deployment will
// probably fail, as references cannot be resolved
func DeployConfigs(clientSet ClientSet, apis api.APIs, sortedConfigs []config.Config, opts DeployConfigsOptions) []error {
	entityMap := newEntityMap(apis)
	var errs []error

	for i := range sortedConfigs {
		c := &sortedConfigs[i] // avoid implicit memory aliasing (gosec G601)

		ctx := context.WithValue(context.TODO(), log.CtxKeyCoord{}, c.Coordinate)
		ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: c.Environment, Group: c.Group})

		entity, deploymentErrors := deploy(ctx, clientSet, apis, entityMap, c)

		if len(deploymentErrors) > 0 {
			for _, err := range deploymentErrors {
				errs = append(errs, fmt.Errorf("failed to deploy config %s: %w", c.Coordinate, err))
			}

			if !opts.ContinueOnErr && !opts.DryRun {
				return errs
			}
		} else if entity != nil {
			entityMap.put(*entity)
		}
	}

	return errs
}

func deploy(ctx context.Context, clientSet ClientSet, apis api.APIs, em *entityMap, c *config.Config) (*parameter.ResolvedEntity, []error) {
	if c.Skip {
		log.WithCtxFields(ctx).Info("Skipping deployment of config %s", c.Coordinate)
		return &parameter.ResolvedEntity{EntityName: c.Coordinate.ConfigId, Coordinate: c.Coordinate, Properties: parameter.Properties{}, Skip: true}, nil
	}

	properties, errs := resolveProperties(c, em.get())
	if len(errs) > 0 {
		return &parameter.ResolvedEntity{}, errs
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		return &parameter.ResolvedEntity{}, []error{err}
	}

	var res *parameter.ResolvedEntity
	var deployErr error
	switch t := c.Type.(type) {
	case config.EntityType:
		log.WithCtxFields(ctx).Debug("Entities are not deployable, skipping entity type: %s", t.EntitiesType)
		return nil, nil

	case config.SettingsType:
		log.WithCtxFields(ctx).Info("Deploying config %s", c.Coordinate)
		res, deployErr = deploySetting(ctx, clientSet.Settings, properties, renderedConfig, c)

	case config.ClassicApiType:
		log.WithCtxFields(ctx).Info("Deploying config %s", c.Coordinate)
		res, deployErr = deployClassicConfig(ctx, clientSet.Classic, apis, em, properties, renderedConfig, c)

	case config.AutomationType:
		log.WithCtxFields(ctx).Info("Deploying config %s", c.Coordinate)
		res, deployErr = deployAutomation(ctx, clientSet.Automation, properties, renderedConfig, c)

	default:
		deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
	}

	if deployErr != nil {
		var responseErr clientErrors.RespError
		if errors.As(deployErr, &responseErr) {
			log.WithCtxFields(ctx).WithFields(field.Error(responseErr)).Error("Failed to deploy config %s: %s", c.Coordinate, responseErr.Message)
		} else {
			log.WithCtxFields(ctx).WithFields(field.Error(deployErr)).Error("Failed to deploy config %s: %s", c.Coordinate, deployErr.Error())
		}
		return nil, []error{deployErr}
	}
	return res, nil

}
