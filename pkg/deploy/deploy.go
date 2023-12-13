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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/mutlierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/validate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"sync"
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
	Automation automation.Client
	Bucket     bucket.Client
}

var DummyClientSet = ClientSet{
	Classic:    &dtclient.DummyClient{},
	Settings:   &dtclient.DummyClient{},
	Automation: &automation.DummyClient{},
	Bucket:     &bucket.DummyClient{},
}

type EnvironmentInfo struct {
	Name  string
	Group string
}
type EnvironmentClients map[EnvironmentInfo]ClientSet

func (e EnvironmentClients) Names() []string {
	n := make([]string, 0, len(e))
	for k := range e {
		n = append(n, k.Name)
	}
	return n
}

var (
	lock sync.Mutex

	skipError = errors.New("skip error")
)

func Deploy(projects []project.Project, environmentClients EnvironmentClients, opts DeployConfigsOptions) error {
	g := graph.New(projects, environmentClients.Names())
	deploymentErrors := make(deployErrors.EnvironmentDeploymentErrors)

	if validationErrs := validate.Validate(projects, []validate.Validator{&classic.Validator{}, &setting.Validator{}}); validationErrs != nil {
		if !opts.ContinueOnErr && !opts.DryRun {
			return validationErrs
		}
		errors.As(validationErrs, &deploymentErrors)
	}

	for env, clients := range environmentClients {
		ctx := createContextWithEnvironment(env)
		log.WithCtxFields(ctx).Info("Deploying configurations to environment %q...", env.Name)

		sortedConfigs, err := g.GetIndependentlySortedConfigs(env.Name)
		if err != nil {
			return fmt.Errorf("failed to get independently sorted configs for environment %q: %w", env.Name, err)
		}

		if err = deployComponents(ctx, sortedConfigs, clients); err != nil {
			log.WithFields(field.Environment(env.Name, env.Group), field.Error(err)).Error("Deployment failed for environment %q: %v", env.Name, err)
			deploymentErrors = deploymentErrors.Append(env.Name, err)
			if !opts.ContinueOnErr && !opts.DryRun {
				return deploymentErrors
			}
		} else {
			log.WithFields(field.Environment(env.Name, env.Group)).Info("Deployment successful for environment %q", env.Name)
		}
	}

	if len(deploymentErrors) != 0 {
		return deploymentErrors
	}

	return nil
}

func deployComponents(ctx context.Context, components []graph.SortedComponent, clients ClientSet) error {
	log.WithCtxFields(ctx).Info("Deploying %d independent configuration sets in parallel...", len(components))
	errCount := 0
	errChan := make(chan error, len(components))

	resolvedEntities := entities.New()
	// Iterate over components and launch a goroutine for each component deployment.
	for i := range components {
		go func(ctx context.Context, component graph.SortedComponent) {
			errChan <- deployGraph(ctx, component.Graph, clients, resolvedEntities)
		}(context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(i)), components[i])
	}

	for range components {
		err := <-errChan
		var deploymentErrs deployErrors.DeploymentErrors
		if errors.As(err, &deploymentErrs) {
			errCount += deploymentErrs.ErrorCount
		} else if err != nil {
			errCount += 1
		}
	}

	close(errChan)

	if errCount > 0 {
		return deployErrors.DeploymentErrors{ErrorCount: errCount}
	}

	return nil
}

func deployGraph(ctx context.Context, configGraph *simple.DirectedGraph, clients ClientSet, resolvedEntities *entities.EntityMap) error {
	g := simple.NewDirectedGraph()
	gonum.Copy(g, configGraph)

	errCount := 0

	errChan := make(chan error)
	for configGraph.Nodes().Len() != 0 {
		roots := graph.Roots(configGraph)

		for _, root := range roots {
			node := root.(graph.ConfigNode)
			go func(ctx context.Context, node graph.ConfigNode) {
				errChan <- deployNode(ctx, node, configGraph, clients, resolvedEntities)
			}(context.WithValue(ctx, log.CtxKeyCoord{}, node.Config.Coordinate), node)
		}

		for range roots {
			err := <-errChan
			if err != nil {
				errCount += 1
			}
		}

		// since all subroutines are done, we need not to lock here
		for _, root := range roots {
			configGraph.RemoveNode(root.ID())
		}
	}

	close(errChan)

	if errCount > 0 {
		return deployErrors.DeploymentErrors{ErrorCount: errCount}
	}

	return nil
}

func deployNode(ctx context.Context, n graph.ConfigNode, configGraph graph.ConfigGraph, clients ClientSet, resolvedEntities *entities.EntityMap) error {
	resolvedEntity, err := deployConfig(ctx, n.Config, clients, resolvedEntities)

	if err != nil {
		failed := !errors.Is(err, skipError)

		lock.Lock()
		removeChildren(ctx, n, n, configGraph, failed)
		lock.Unlock()

		if failed {
			return err
		}
		return nil
	}

	resolvedEntities.Put(resolvedEntity)
	log.WithCtxFields(ctx).WithFields(field.StatusDeployed()).Info("Deployment successful")
	return nil
}

func removeChildren(ctx context.Context, parent, root graph.ConfigNode, configGraph graph.ConfigGraph, failed bool) {

	children := configGraph.From(parent.ID())
	for children.Next() {
		child := children.Node().(graph.ConfigNode)

		reason := "was skipped"
		if failed {
			reason = "failed to deploy"
		}
		childCfg := child.Config

		l := log.WithCtxFields(ctx).WithFields(
			field.F("parent", parent.Config.Coordinate),
			field.F("deploymentFailed", failed),
			field.F("child", child.Config.Coordinate),
			field.StatusDeploymentSkipped())

		// after the first iteration
		if parent != root {
			l.WithFields(field.F("root", root.Config.Coordinate)).
				Warn("Skipping deployment of %v, as it depends on %v which was not deployed after root dependency configuration %v %s", childCfg.Coordinate, parent.Config.Coordinate, root.Config.Coordinate, reason)
		} else {
			l.Warn("Skipping deployment of %v, as it depends on %v which %s", childCfg.Coordinate, parent.Config.Coordinate, reason)
		}

		removeChildren(ctx, child, root, configGraph, failed)

		configGraph.RemoveNode(child.ID())
	}
}

func deployConfig(ctx context.Context, c *config.Config, clients ClientSet, resolvedEntities config.EntityLookup) (entities.ResolvedEntity, error) {
	if c.Skip {
		log.WithCtxFields(ctx).WithFields(field.StatusDeploymentSkipped()).Info("Skipping deployment of config")
		return entities.ResolvedEntity{}, skipError //fake resolved entity that "old" deploy creates is never needed, as we don't even try to deploy dependencies of skipped configs (so no reference will ever be attempted to resolve)
	}

	properties, errs := c.ResolveParameterValues(resolvedEntities)
	if len(errs) > 0 {
		err := mutlierror.New(errs...)
		log.WithCtxFields(ctx).WithFields(field.Error(err), field.StatusDeploymentFailed()).Error("Invalid configuration - failed to resolve parameter values: %v", err)
		return entities.ResolvedEntity{}, err
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		log.WithCtxFields(ctx).WithFields(field.Error(err), field.StatusDeploymentFailed()).Error("Invalid configuration - failed to render JSON template: %v", err)
		return entities.ResolvedEntity{}, err
	}

	log.WithCtxFields(ctx).WithFields(field.StatusDeploying()).Info("Deploying config")
	var resolvedEntity entities.ResolvedEntity
	var deployErr error
	switch c.Type.(type) {
	case config.SettingsType:
		resolvedEntity, deployErr = setting.Deploy(ctx, clients.Settings, properties, renderedConfig, c)

	case config.ClassicApiType:
		resolvedEntity, deployErr = classic.Deploy(ctx, clients.Classic, api.NewAPIs(), properties, renderedConfig, c)

	case config.AutomationType:
		resolvedEntity, deployErr = automation.Deploy(ctx, clients.Automation, properties, renderedConfig, c)

	case config.BucketType:
		resolvedEntity, deployErr = bucket.Deploy(ctx, clients.Bucket, properties, renderedConfig, c)

	default:
		deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
	}

	if deployErr != nil {
		var responseErr clientErrors.RespError
		if errors.As(deployErr, &responseErr) {
			logResponseError(ctx, responseErr)
			return entities.ResolvedEntity{}, responseErr
		}

		log.WithCtxFields(ctx).WithFields(field.Error(deployErr)).Error("Deployment failed - Monaco Error: %v", deployErr)
		return entities.ResolvedEntity{}, deployErr
	}
	return resolvedEntity, nil
}

// logResponseError prints user-friendly messages based on the response errors status
func logResponseError(ctx context.Context, responseErr clientErrors.RespError) {
	if responseErr.StatusCode >= 400 && responseErr.StatusCode <= 499 {
		log.WithCtxFields(ctx).WithFields(field.Error(responseErr), field.StatusDeploymentFailed()).Error("Deployment failed - Dynatrace API rejected HTTP request / JSON data: %v", responseErr)
		return
	}

	if responseErr.StatusCode >= 500 && responseErr.StatusCode <= 599 {
		log.WithCtxFields(ctx).WithFields(field.Error(responseErr), field.StatusDeploymentFailed()).Error("Deployment failed - Dynatrace Server Error: %v", responseErr)
		return
	}

	log.WithCtxFields(ctx).WithFields(field.Error(responseErr), field.StatusDeploymentFailed()).Error("Deployment failed - Dynatrace API call unsuccessful: %v", responseErr)
}

func createContextWithEnvironment(env EnvironmentInfo) context.Context {
	return context.WithValue(context.TODO(), log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})
}
