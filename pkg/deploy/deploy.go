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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	graph2 "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"strings"
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
	Automation automationClient
	Bucket     bucketClient
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
		} else {
			entityMap.put(*entity)
		}
	}

	return errs
}

type EnvironmentDeploymentErrors map[string][]error

func (e EnvironmentDeploymentErrors) Error() string {
	b := strings.Builder{}
	for env, errs := range e {
		b.WriteString(fmt.Sprintf("%s deployment errors: %v", env, errs))
	}
	return b.String()
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

func DeployConfigGraph(projects []project.Project, environmentClients EnvironmentClients, opts DeployConfigsOptions) EnvironmentDeploymentErrors {

	apis := api.NewAPIs()
	g := graph.New(projects, environmentClients.Names())

	errs := make(EnvironmentDeploymentErrors)

	for env, clients := range environmentClients {
		envErrs := deployComponentsToEnvironment(g, env, clients, apis, opts)
		if len(envErrs) > 0 {
			errs[env.Name] = envErrs

			if !opts.ContinueOnErr && !opts.DryRun {
				return errs
			}
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func deployComponentsToEnvironment(g graph.ConfigGraphPerEnvironment, env EnvironmentInfo, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) []error {

	ctx := context.WithValue(context.TODO(), log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})

	log.WithCtxFields(ctx).Info("Deploying configurations to environment %q...", env.Name)

	sortedConfigs, err := g.GetIndependentlySortedConfigs(env.Name)
	if err != nil {
		return []error{fmt.Errorf("failed to get independently sorted configs for environment %q: %w", env.Name, err)}
	}

	var deployErrs []error
	if featureflags.DependencyGraphBasedDeployParallel().Enabled() {
		deployErrs = deployComponentsParallel(ctx, sortedConfigs, clientSet, apis, opts)
	} else {
		deployErrs = deployComponents(ctx, sortedConfigs, clientSet, apis, opts)
	}

	if len(deployErrs) > 0 {
		return deployErrs
	}

	return nil
}

var skipError = errors.New("skip error")

func deployComponents(ctx context.Context, components []graph.SortedComponent, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) []error {

	var errs []error

	log.WithCtxFields(ctx).Info("Deploying %d independent configuration sets...", len(components))

	for i := range components {
		ctx = context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(i))
		componentDeployErrs := deployComponent(ctx, components[i], clientSet, apis, opts)

		if len(componentDeployErrs) > 0 && !opts.ContinueOnErr && !opts.DryRun {
			return componentDeployErrs
		}

		errs = append(errs, componentDeployErrs...)
	}

	return errs
}

func deployComponentsParallel(ctx context.Context, components []graph.SortedComponent, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) []error {
	var errs []error
	log.WithCtxFields(ctx).Info("Deploying %d independent configuration sets in parallel...", len(components))

	errChan := make(chan []error, len(components))

	// Iterate over components and launch a goroutine for each component deployment.
	for i := range components {
		c := context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(i))
		go func(ctx context.Context, component graph.SortedComponent) {
			componentDeployErrs := deployComponent(ctx, component, clientSet, apis, opts)
			errChan <- componentDeployErrs
		}(c, components[i])
	}

	// Collect errors from goroutines and append to the 'errs' slice.
	for range components {
		componentDeployErrs := <-errChan
		errs = append(errs, componentDeployErrs...)
	}

	// Close the error channel.
	close(errChan)

	return errs
}

type componentDeployer struct {
	lock             sync.Mutex
	graph            graph.ConfigGraph
	clients          ClientSet
	resolvedEntities entityMap
	apis             api.APIs
}

func (c *componentDeployer) deploy(ctx context.Context) []error {
	var errs []error

	errChan := make(chan error)
	for c.graph.Nodes().Len() != 0 {
		roots := graph.Roots(c.graph)

		for _, root := range roots {
			node := root.(graph.ConfigNode)
			ctx := context.WithValue(ctx, log.CtxKeyCoord{}, node.Config.Coordinate)

			go func(ctx context.Context, node graph.ConfigNode) {
				errChan <- c.deployNode(ctx, node)
			}(ctx, node)
		}

		for range roots {
			err := <-errChan
			if err != nil {
				errs = append(errs, err)
			}
		}

		// since all subroutines are done, we need not to lock here
		for _, root := range roots {
			c.graph.RemoveNode(root.ID())
		}
	}

	close(errChan)

	return errs
}

func (c *componentDeployer) deployNode(ctx context.Context, n graph.ConfigNode) error {
	entity, err := deployFunc(ctx, n.Config, c.clients, c.apis, &c.resolvedEntities)

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

	c.resolvedEntities.put(*entity)
	log.WithCtxFields(ctx).Info("Deployment successful")
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
		childCnf := child.Config

		// after the first iteration
		if parent != root {
			log.WithCtxFields(ctx).WithFields(field.F("parent", parent.Config.Coordinate), field.F("deploymentFailed", failed), field.F("root", root.Config.Coordinate)).
				Warn("Skipping deployment of %v, as it depends on %v which was not deployed after root dependency configuration %v %s", childCnf.Coordinate, parent.Config.Coordinate, root.Config.Coordinate, reason)
		} else {
			log.WithCtxFields(ctx).WithFields(field.F("parent", parent.Config.Coordinate), field.F("deploymentFailed", failed)).
				Warn("Skipping deployment of %v, as it depends on %v which %s", childCnf.Coordinate, parent.Config.Coordinate, reason)
		}

		c.removeChildren(ctx, child, root, failed)

		c.graph.RemoveNode(child.ID())
	}
}
func deployComponent(ctx context.Context, component graph.SortedComponent, clientSet ClientSet, apis api.APIs, _ DeployConfigsOptions) []error {
	g := simple.NewDirectedGraph()
	graph2.Copy(g, component.Graph)

	deployer := componentDeployer{
		lock:             sync.Mutex{},
		graph:            g,
		clients:          clientSet,
		resolvedEntities: *newEntityMap(apis),
		apis:             apis,
	}
	return deployer.deploy(ctx)
}

// deployFunc kinda just is a smarter deploy... TODO refactor!
func deployFunc(ctx context.Context, c *config.Config, clientSet ClientSet, apis api.APIs, entityMap *entityMap) (*parameter.ResolvedEntity, error) {
	if c.Skip {
		log.WithCtxFields(ctx).Info("Skipping deployment of config %s", c.Coordinate)
		return nil, skipError //fake resolved entity that "old" deploy creates is never needed, as we don't even try to deploy dependencies of skipped configs (so no reference will ever be attempted to resolve)
	}

	entity, deploymentErrors := deploy(ctx, clientSet, apis, entityMap, c)

	if len(deploymentErrors) > 0 {
		return nil, fmt.Errorf("failed to deploy config %s: %w", c.Coordinate, errors.Join(deploymentErrors...))
	}

	return entity, nil
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

	log.WithCtxFields(ctx).Info("Deploying config")
	var res *parameter.ResolvedEntity
	var deployErr error
	switch c.Type.(type) {
	case config.SettingsType:
		res, deployErr = deploySetting(ctx, clientSet.Settings, properties, renderedConfig, c)

	case config.ClassicApiType:
		res, deployErr = deployClassicConfig(ctx, clientSet.Classic, apis, em, properties, renderedConfig, c)

	case config.AutomationType:
		res, deployErr = deployAutomation(ctx, clientSet.Automation, properties, renderedConfig, c)

	case config.BucketType:
		res, deployErr = deployBucket(ctx, clientSet.Bucket, properties, renderedConfig, c)

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
		return nil, []error{deployErr}
	}
	return res, nil

}
