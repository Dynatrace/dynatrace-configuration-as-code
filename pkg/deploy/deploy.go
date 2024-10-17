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
	"sync"
	"time"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/multierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/setting"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/validate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
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
	Classic      client.ConfigClient
	Settings     client.SettingsClient
	Automation   automation.Client
	Bucket       bucket.Client
	Document     document.Client
	OpenPipeline openpipeline.Client
}

var DummyClientSet = ClientSet{
	Classic:      &dtclient.DummyClient{},
	Settings:     &dtclient.DummyClient{},
	Automation:   &automation.DummyClient{},
	Bucket:       &bucket.DummyClient{},
	Document:     &document.DummyClient{},
	OpenPipeline: &openpipeline.DummyClient{},
}

var (
	lock sync.Mutex

	skipError = errors.New("skip error")
)

func Deploy(ctx context.Context, projects []project.Project, environmentClients dynatrace.EnvironmentClients, opts DeployConfigsOptions) error {
	preloadCaches(ctx, projects, environmentClients)
	g := graph.New(projects, environmentClients.Names())
	deploymentErrors := make(deployErrors.EnvironmentDeploymentErrors)

	if validationErrs := validate.Validate(projects); validationErrs != nil {
		if !opts.ContinueOnErr && !opts.DryRun {
			return validationErrs
		}
		errors.As(validationErrs, &deploymentErrors)
	}

	for env, clients := range environmentClients {
		ctx := newContextWithEnvironment(ctx, env)
		log.WithCtxFields(ctx).Info("Deploying configurations to environment %q...", env.Name)

		sortedConfigs, err := g.GetIndependentlySortedConfigs(env.Name)
		if err != nil {
			return fmt.Errorf("failed to get independently sorted configs for environment %q: %w", env.Name, err)
		}

		var clientSet ClientSet
		if opts.DryRun {
			clientSet = DummyClientSet
		} else {
			clientSet = ClientSet{
				Classic:      clients.DTClient,
				Settings:     clients.DTClient,
				Automation:   clients.AutClient,
				Bucket:       clients.BucketClient,
				Document:     clients.DocumentClient,
				OpenPipeline: clients.OpenPipelineClient,
			}
		}

		if err = deployComponents(ctx, sortedConfigs, clientSet); err != nil {
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
			time.Sleep(api.NewAPIs()[node.Config.Coordinate.Type].DeployWaitDuration)

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
	ctx = report.NewContextWithDetailer(ctx, report.NewDefaultDetailer())
	resolvedEntity, err := deployConfig(ctx, n.Config, clients, resolvedEntities)
	details := report.GetDetailerFromContextOrDiscard(ctx).GetDetails()

	// Need to tidy this up, just keep it all in once place at the moment
	if err != nil {
		if errors.Is(err, skipError) {
			report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(n.Config.Coordinate, report.State_DEPL_EXCLUDED, details, nil)
		} else {
			report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(n.Config.Coordinate, report.State_DEPL_ERR, details, err)
		}
	} else {
		report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(n.Config.Coordinate, report.State_DEPL_SUCCESS, details, nil)
	}

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

		report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(childCfg.Coordinate, report.State_DEPL_SKIPPED, nil, nil)

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
		err := multierror.New(errs...)
		log.WithCtxFields(ctx).WithFields(field.Error(err), field.StatusDeploymentFailed()).Error("Invalid configuration - failed to resolve parameter values: %v", err)
		report.GetDetailerFromContextOrDiscard(ctx).AddDetail(report.Detail{Type: report.TypeError, Message: fmt.Sprintf("Failed to resolve parameter values: %v", err)})
		return entities.ResolvedEntity{}, err
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		log.WithCtxFields(ctx).WithFields(field.Error(err), field.StatusDeploymentFailed()).Error("Invalid configuration - failed to render JSON template: %v", err)
		report.GetDetailerFromContextOrDiscard(ctx).AddDetail(report.Detail{Type: report.TypeError, Message: fmt.Sprintf("Failed to render JSON template: %v", err)})
		return entities.ResolvedEntity{}, err
	}

	log.WithCtxFields(ctx).WithFields(field.StatusDeploying()).Info("Deploying config")
	var resolvedEntity entities.ResolvedEntity
	var deployErr error
	switch c.Type.(type) {
	case config.SettingsType:
		var insertAfter string
		if ia, ok := properties[config.InsertAfterParameter]; ok {
			insertAfter = ia.(string)
		}
		resolvedEntity, deployErr = setting.Deploy(ctx, clients.Settings, properties, renderedConfig, c, insertAfter)

	case config.ClassicApiType:
		resolvedEntity, deployErr = classic.Deploy(ctx, clients.Classic, api.NewAPIs(), properties, renderedConfig, c)

	case config.AutomationType:
		resolvedEntity, deployErr = automation.Deploy(ctx, clients.Automation, properties, renderedConfig, c)

	case config.BucketType:
		resolvedEntity, deployErr = bucket.Deploy(ctx, clients.Bucket, properties, renderedConfig, c)

	case config.DocumentType:
		if featureflags.Temporary[featureflags.Documents].Enabled() {
			resolvedEntity, deployErr = document.Deploy(ctx, clients.Document, properties, renderedConfig, c)
		} else {
			deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
		}

	case config.OpenPipelineType:
		if featureflags.Temporary[featureflags.OpenPipeline].Enabled() {
			resolvedEntity, deployErr = openpipeline.Deploy(ctx, clients.OpenPipeline, properties, renderedConfig, c)
		} else {
			deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
		}

	default:
		deployErr = fmt.Errorf("unknown config-type (ID: %q)", c.Type.ID())
	}

	if deployErr != nil {
		var responseErr coreapi.APIError
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
func logResponseError(ctx context.Context, responseErr coreapi.APIError) {
	if responseErr.StatusCode >= 400 && responseErr.StatusCode <= 499 {
		log.WithCtxFields(ctx).WithFields(field.Error(responseErr), field.StatusDeploymentFailed()).Error("Deployment failed - Dynatrace API rejected HTTP request / JSON data: %v", responseErr)
		report.GetDetailerFromContextOrDiscard(ctx).AddDetail(report.Detail{Type: "ERROR", Message: fmt.Sprintf("Dynatrace API rejected request: : %v", responseErr)})
		return
	}

	if responseErr.StatusCode >= 500 && responseErr.StatusCode <= 599 {
		log.WithCtxFields(ctx).WithFields(field.Error(responseErr), field.StatusDeploymentFailed()).Error("Deployment failed - Dynatrace Server Error: %v", responseErr)
		return
	}

	log.WithCtxFields(ctx).WithFields(field.Error(responseErr), field.StatusDeploymentFailed()).Error("Deployment failed - Dynatrace API call unsuccessful: %v", responseErr)
}

func newContextWithEnvironment(ctx context.Context, env dynatrace.EnvironmentInfo) context.Context {
	return context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})
}
