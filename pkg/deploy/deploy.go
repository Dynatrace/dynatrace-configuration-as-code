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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/mutlierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	errors2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/entitymap"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	graph2 "gonum.org/v1/gonum/graph"
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

func DeployConfigGraph(projects []project.Project, environmentClients EnvironmentClients, opts DeployConfigsOptions) error {

	apis := api.NewAPIs()
	g := graph.New(projects, environmentClients.Names())

	errs := make(errors2.EnvironmentDeploymentErrors)

	validationErrs := classic.ValidateUniqueConfigNames(projects)
	if validationErrs != nil {
		if !opts.ContinueOnErr && !opts.DryRun {
			return validationErrs
		}

		errors.As(validationErrs, &errs) // use validation errors as base errors if possible
	}

	for env, clients := range environmentClients {
		err := deployComponentsToEnvironment(g, env, clients, apis, opts)
		if err != nil {
			log.WithFields(field.Environment(env.Name, env.Group), field.Error(err)).Error("Deployment failed for environment %q: %v", env.Name, err)
			errs = errs.Append(env.Name, err)

			if !opts.ContinueOnErr && !opts.DryRun {
				return errs
			}
		} else {
			log.WithFields(field.Environment(env.Name, env.Group)).Info("Deployment successful for environment %q", env.Name)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func deployComponentsToEnvironment(g graph.ConfigGraphPerEnvironment, env EnvironmentInfo, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) error {

	ctx := context.WithValue(context.TODO(), log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})

	log.WithCtxFields(ctx).Info("Deploying configurations to environment %q...", env.Name)

	sortedConfigs, err := g.GetIndependentlySortedConfigs(env.Name)
	if err != nil {
		return fmt.Errorf("failed to get independently sorted configs for environment %q: %w", env.Name, err)
	}

	if featureflags.DependencyGraphBasedDeployParallel().Enabled() {
		return deployComponentsParallel(ctx, sortedConfigs, clientSet, apis, opts)
	}

	return deployComponents(ctx, sortedConfigs, clientSet, apis, opts)
}

var skipError = errors.New("skip error")

func deployComponents(ctx context.Context, components []graph.SortedComponent, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) error {

	errCount := 0

	log.WithCtxFields(ctx).Info("Deploying %d independent configuration sets...", len(components))

	for i := range components {
		ctx = context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(i))
		err := deployComponent(ctx, components[i], clientSet, apis, opts)

		if err != nil {
			if !opts.ContinueOnErr && !opts.DryRun {
				return err
			}

			var deploymentErrs errors2.DeploymentErrors
			if errors.As(err, &deploymentErrs) {
				errCount += deploymentErrs.ErrorCount
			} else {
				errCount += 1
			}
		}
	}

	if errCount > 0 {
		return errors2.DeploymentErrors{ErrorCount: errCount}
	}

	return nil
}

func deployComponentsParallel(ctx context.Context, components []graph.SortedComponent, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) error {

	errCount := 0

	log.WithCtxFields(ctx).Info("Deploying %d independent configuration sets in parallel...", len(components))

	errChan := make(chan error, len(components))

	// Iterate over components and launch a goroutine for each component deployment.
	for i := range components {
		c := context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(i))
		go func(ctx context.Context, component graph.SortedComponent) {
			componentDeployErr := deployComponent(ctx, component, clientSet, apis, opts)
			errChan <- componentDeployErr
		}(c, components[i])
	}

	// Collect errors from goroutines and append to the 'errs' slice.
	for range components {
		err := <-errChan

		var deploymentErrs errors2.DeploymentErrors
		if errors.As(err, &deploymentErrs) {
			errCount += deploymentErrs.ErrorCount
		} else if err != nil {
			errCount += 1
		}
	}

	// Close the error channel.
	close(errChan)

	if errCount > 0 {
		return errors2.DeploymentErrors{ErrorCount: errCount}
	}

	return nil
}

type componentDeployer struct {
	lock             sync.Mutex
	graph            graph.ConfigGraph
	clients          ClientSet
	resolvedEntities entitymap.EntityMap
	apis             api.APIs
}

func (c *componentDeployer) deploy(ctx context.Context) error {
	errCount := 0

	errChan := make(chan error)
	for c.graph.Nodes().Len() != 0 {
		roots := graph.Roots(c.graph)

		for _, root := range roots {
			node := root.(graph.ConfigNode)
			go func(ctx context.Context, node graph.ConfigNode) {
				errChan <- c.deployNode(ctx, node)
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
			c.graph.RemoveNode(root.ID())
		}
	}

	close(errChan)

	if errCount > 0 {
		return errors2.DeploymentErrors{ErrorCount: errCount}
	}

	return nil
}

func (c *componentDeployer) deployNode(ctx context.Context, n graph.ConfigNode) error {
	entity, err := deploy(ctx, n.Config, c.clients, c.apis, &c.resolvedEntities)

	// lock changes we will make to shared variables. Writing them is trivial compared to any http request
	c.lock.Lock()
	defer c.lock.Unlock()

	if err != nil {
		failed := !errors.Is(err, skipError)
		c.removeChildren(ctx, n, n, failed)

		if failed {
			return err
		}
		return nil
	}

	c.resolvedEntities.Put(entity)
	log.WithCtxFields(ctx).WithFields(field.StatusDeployed()).Info("Deployment successful")
	return nil
}

func (c *componentDeployer) removeChildren(ctx context.Context, parent, root graph.ConfigNode, failed bool) {

	children := c.graph.From(parent.ID())
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

		c.removeChildren(ctx, child, root, failed)

		c.graph.RemoveNode(child.ID())
	}
}
func deployComponent(ctx context.Context, component graph.SortedComponent, clientSet ClientSet, apis api.APIs, _ DeployConfigsOptions) error {
	g := simple.NewDirectedGraph()
	graph2.Copy(g, component.Graph)

	deployer := componentDeployer{
		lock:             sync.Mutex{},
		graph:            g,
		clients:          clientSet,
		resolvedEntities: *entitymap.New(),
		apis:             apis,
	}
	return deployer.deploy(ctx)
}

func deploy(ctx context.Context, c *config.Config, clientSet ClientSet, apis api.APIs, entityMap *entitymap.EntityMap) (config.ResolvedEntity, error) {
	if c.Skip {
		log.WithCtxFields(ctx).WithFields(field.StatusDeploymentSkipped()).Info("Skipping deployment of config")
		return config.ResolvedEntity{}, skipError //fake resolved entity that "old" deploy creates is never needed, as we don't even try to deploy dependencies of skipped configs (so no reference will ever be attempted to resolve)
	}

	properties, errs := c.ResolveParameterValues(entityMap)
	if len(errs) > 0 {
		err := mutlierror.New(errs...)
		log.WithCtxFields(ctx).WithFields(field.Error(err), field.StatusDeploymentFailed()).Error("Invalid configuration - failed to resolve parameter values: %v", err)
		return config.ResolvedEntity{}, err
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		log.WithCtxFields(ctx).WithFields(field.Error(err), field.StatusDeploymentFailed()).Error("Invalid configuration - failed to render JSON template: %v", err)
		return config.ResolvedEntity{}, err
	}

	log.WithCtxFields(ctx).WithFields(field.StatusDeploying()).Info("Deploying config")
	var entity config.ResolvedEntity
	var deployErr error
	switch c.Type.(type) {
	case config.SettingsType:
		entity, deployErr = setting.Deploy(ctx, clientSet.Settings, properties, renderedConfig, c)

	case config.ClassicApiType:
		entity, deployErr = classic.Deploy(ctx, clientSet.Classic, apis, properties, renderedConfig, c)

	case config.AutomationType:
		entity, deployErr = automation.Deploy(ctx, clientSet.Automation, properties, renderedConfig, c)

	case config.BucketType:
		entity, deployErr = bucket.Deploy(ctx, clientSet.Bucket, properties, renderedConfig, c)

	default:
		deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
	}

	if deployErr != nil {
		var responseErr clientErrors.RespError
		if errors.As(deployErr, &responseErr) {
			logResponseError(ctx, responseErr)
			return config.ResolvedEntity{}, responseErr
		}

		log.WithCtxFields(ctx).WithFields(field.Error(deployErr)).Error("Deployment failed - Monaco Error: %v", deployErr)
		return config.ResolvedEntity{}, deployErr
	}
	return entity, nil
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
