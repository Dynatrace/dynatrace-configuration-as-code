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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"strings"
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
		go func(component graph.SortedComponent, subGraphID int) {
			ctx = context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(subGraphID))
			componentDeployErrs := deployComponent(ctx, component, clientSet, apis, opts)
			errChan <- componentDeployErrs
		}(components[i], i)
	}

	// Collect errors from goroutines and append to the 'errs' slice.
	for range components {
		componentDeployErrs := <-errChan
		if len(componentDeployErrs) > 0 && !opts.ContinueOnErr && !opts.DryRun {
			return componentDeployErrs
		}
		errs = append(errs, componentDeployErrs...)
	}

	// Close the error channel.
	close(errChan)

	return errs
}

func deployComponent(ctx context.Context, component graph.SortedComponent, clientSet ClientSet, apis api.APIs, opts DeployConfigsOptions) []error {
	var errs []error

	entityMap := newEntityMap(apis) //entityMap is only used to when resolving parameter references, and configs that reference each other are in the same component; no global entity map is needed

	g := component.Graph
	sortedNodes := component.SortedNodes

	if log.Level() == loggers.LevelDebug {
		log.WithCtxFields(ctx).Debug("Deploying configurations in current component: %v", sortedNodes)
	}

	dontDeploy := map[int64]struct{}{} //look-up map marking Nodes (by ID) that should not be deployed, because something they depend on was skipped or failed

	for _, gNode := range sortedNodes {

		node := gNode.(graph.ConfigNode)
		id := node.ID()

		if _, dont := dontDeploy[id]; dont {
			continue
		}

		ctx := context.WithValue(ctx, log.CtxKeyCoord{}, node.Config.Coordinate)
		entity, err := deployFunc(ctx, node.Config, clientSet, apis, entityMap)

		if err != nil {
			deploymentFailed := false

			if !errors.Is(err, skipError) {
				deploymentFailed = true
				errs = append(errs, err)
				if !opts.ContinueOnErr && !opts.DryRun {
					return errs
				}
			}

			markChildrenAsNotToDeploy(ctx, g, &node, nil, dontDeploy, deploymentFailed)

		} else {
			entityMap.put(*entity)
			log.Info("Deployed %v successfully (graph-node-id %d)", node.Config.Coordinate, id)
		}
	}
	return errs
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

func markChildrenAsNotToDeploy(ctx context.Context, g graph.ConfigGraph, node, root *graph.ConfigNode, dontDeploy map[int64]struct{}, deploymentFailed bool) {

	reason := "was skipped"
	if deploymentFailed {
		reason = "failed to deploy"
	}

	children := g.From(node.ID())
	for children.Next() {
		child := children.Node().(graph.ConfigNode)
		if root != nil {
			log.WithCtxFields(ctx).WithFields(field.F("child", child.Config.Coordinate), field.F("root", root.Config.Coordinate), field.F("deploymentFailed", deploymentFailed)).Warn("Skipping deployment of %v, as it depends on %v which was not deployed after root dependency configuration %v %s", child.Config.Coordinate, node.Config.Coordinate, root.Config.Coordinate, reason)
		} else {
			log.WithCtxFields(ctx).WithFields(field.F("child", child.Config.Coordinate), field.F("deploymentFailed", deploymentFailed)).Warn("Skipping deployment of %v, as it depends on %v which %s", child.Config.Coordinate, node.Config.Coordinate, reason)
		}

		dontDeploy[children.Node().ID()] = struct{}{}
		markChildrenAsNotToDeploy(ctx, g, &child, node, dontDeploy, deploymentFailed)
	}
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
	switch c.Type.(type) {

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
			log.WithCtxFields(ctx).WithFields(field.Error(responseErr)).Error("Failed to deploy config %s: %s", c.Coordinate, responseErr.Reason)
		} else {
			log.WithCtxFields(ctx).WithFields(field.Error(deployErr)).Error("Failed to deploy config %s: %s", c.Coordinate, deployErr.Error())
		}
		return nil, []error{deployErr}
	}
	return res, nil

}
