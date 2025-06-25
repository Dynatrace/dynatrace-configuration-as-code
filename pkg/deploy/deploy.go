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
	"log/slog"
	"sync"
	"time"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/attribute"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/multierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/validate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/slo"
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

var (
	lock    sync.Mutex
	errSkip = errors.New("skip error")
)

type ctxDeploymentLimiterKey struct{}

func newContextWithDeploymentLimiter(ctx context.Context, limiter *rest.ConcurrentRequestLimiter) context.Context {
	return context.WithValue(ctx, ctxDeploymentLimiterKey{}, limiter)
}

func getDeploymentLimiterFromContext(ctx context.Context) *rest.ConcurrentRequestLimiter {
	if limiter, ok := ctx.Value(ctxDeploymentLimiterKey{}).(*rest.ConcurrentRequestLimiter); ok {
		return limiter
	}
	return nil
}

func DeployForAllEnvironments(ctx context.Context, projects []project.Project, environmentClients dynatrace.EnvironmentClients, opts DeployConfigsOptions) error {
	maxConcurrentDeployments := environment.GetEnvValueIntLog(environment.ConcurrentDeploymentsEnvKey)
	if maxConcurrentDeployments > 0 {
		log.InfoContext(ctx, "%s set, limiting concurrent deployments to %d", environment.ConcurrentDeploymentsEnvKey, maxConcurrentDeployments)
		limiter := rest.NewConcurrentRequestLimiter(maxConcurrentDeployments)
		ctx = newContextWithDeploymentLimiter(ctx, limiter)
	}
	deploymentErrs := make(deployErrors.EnvironmentDeploymentErrors)

	// note: Currently the validation works 'environment-independent', but that might be something we should reconsider to improve error messages
	if validationErrs := validate.Validate(projects); validationErrs != nil {
		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateError, validationErrs, "", nil)
		if !opts.ContinueOnErr && !opts.DryRun {
			return validationErrs
		}
		errors.As(validationErrs, &deploymentErrs)
	}

	reporter := report.GetReporterFromContextOrDiscard(ctx)

	envNames := environmentClients.Names()
	g := graph.New(projects, envNames)
	envConfigs, err := getSortedEnvConfigs(g, envNames)
	if err != nil {
		reporter.ReportLoading(report.StateError, err, "", nil)
		return err
	}

	projectString := "project"
	if len(projects) > 1 {
		projectString = "projects"
	}
	reporter.ReportInfo(fmt.Sprintf("%d %v validated", len(projects), projectString))
	defer reporter.ReportInfo("Deployment finished")

	for env, clientSet := range environmentClients {
		sortedConfigs, ok := envConfigs[env.Name]
		if !ok {
			return fmt.Errorf("failed to get independently sorted configs for environment %q", env.Name)
		}
		ctx = newContextWithEnvironment(ctx, env)

		if depErr := Deploy(ctx, clientSet, projects, sortedConfigs, env.Name); depErr != nil {
			log.With(attribute.EnvironmentAttr(env.Name, env.Group), attribute.ErrorAttr(depErr)).ErrorContext(ctx, "Deployment failed for environment '%s': %v", env.Name, depErr)
			deploymentErrs = deploymentErrs.Append(env.Name, depErr)

			if !opts.ContinueOnErr && !opts.DryRun {
				return deploymentErrs
			}
		} else {
			log.With(attribute.EnvironmentAttr(env.Name, env.Group)).InfoContext(ctx, "Deployment successful for environment '%s'", env.Name)
		}
	}

	if len(deploymentErrs) != 0 {
		return deploymentErrs
	}

	return nil
}

func createDeployables(clientSet *client.ClientSet) resource.Deployables {
	return resource.Deployables{
		config.ServiceLevelObjectiveID: slo.NewDeployAPI(clientSet.ServiceLevelObjectiveClient),
		config.SegmentID:               segment.NewDeployAPI(clientSet.SegmentClient),
		config.ClassicApiTypeID:        classic.NewDeployAPI(clientSet.ConfigClient, api.NewAPIs()),
		config.SettingsTypeID:          settings.NewDeployAPI(clientSet.SettingsClient),
		config.OpenPipelineTypeID:      openpipeline.NewDeployAPI(clientSet.OpenPipelineClient),
		config.AutomationTypeID:        automation.NewDeployAPI(clientSet.AutClient),
		config.DocumentTypeID:          document.NewDeployAPI(clientSet.DocumentClient),
		config.BucketTypeID:            bucket.NewDeployAPI(clientSet.BucketClient),
	}
}

func Deploy(ctx context.Context, clientSet *client.ClientSet, projects []project.Project, sortedConfigs []graph.SortedComponent, environment string) error {
	preloadCaches(ctx, projects, clientSet, environment)
	defer clearCaches(clientSet)
	deployables := createDeployables(clientSet)
	log.InfoContext(ctx, "Deploying configurations to environment %q...", environment)

	return deployComponents(ctx, sortedConfigs, deployables)
}

// getSortedEnvConfigs sorts the config graphs and checks for certain errors like cyclic dependencies
func getSortedEnvConfigs(g graph.ConfigGraphPerEnvironment, envNames []string) (map[string][]graph.SortedComponent, error) {
	envConfigs := make(map[string][]graph.SortedComponent)
	for _, env := range envNames {
		sortedConfigs, err := g.GetIndependentlySortedConfigs(env)
		if err != nil {
			return nil, fmt.Errorf("failed to get independently sorted configs for environment %q: %w", env, err)
		}
		envConfigs[env] = sortedConfigs
	}
	return envConfigs, nil
}

func deployComponents(ctx context.Context, components []graph.SortedComponent, deployables resource.Deployables) error {
	log.InfoContext(ctx, "Deploying %d independent configuration sets in parallel...", len(components))
	errCount := 0
	errChan := make(chan error, len(components))

	// Iterate over components and launch a goroutine for each component deployment.
	for i := range components {
		go func(ctx context.Context, component graph.SortedComponent) {
			errChan <- deployGraph(ctx, component.Graph, deployables)
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

func deployGraph(ctx context.Context, configGraph *simple.DirectedGraph, deployables resource.Deployables) error {
	g := simple.NewDirectedGraph()
	gonum.Copy(g, configGraph)
	resolvedEntities := entities.New()
	errCount := 0

	errChan := make(chan error)
	for configGraph.Nodes().Len() != 0 {
		roots := graph.Roots(configGraph)

		for _, root := range roots {
			node := root.(graph.ConfigNode)
			time.Sleep(api.NewAPIs()[node.Config.Coordinate.Type].DeployWaitDuration)

			go func(ctx context.Context, node graph.ConfigNode) {
				errChan <- deployNode(ctx, node, configGraph, deployables, resolvedEntities)
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

func deployNode(ctx context.Context, n graph.ConfigNode, configGraph graph.ConfigGraph, deployables resource.Deployables, resolvedEntities *entities.EntityMap) error {
	ctx = report.NewContextWithDetailer(ctx, report.NewDefaultDetailer())
	resolvedEntity, err := deployConfig(ctx, n.Config, deployables, resolvedEntities)
	details := report.GetDetailerFromContextOrDiscard(ctx).GetAll()

	if err != nil {
		failed := !errors.Is(err, errSkip)

		lock.Lock()
		removeChildren(ctx, n, n, configGraph, failed)
		lock.Unlock()

		if failed {
			report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(n.Config.Coordinate, report.StateError, details, err)
			return err
		}
		report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(n.Config.Coordinate, report.StateExcluded, details, nil)
		return nil
	}

	resolvedEntities.Put(resolvedEntity)
	report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(n.Config.Coordinate, report.StateSuccess, details, nil)
	log.With(attribute.StatusDeployedAttr()).InfoContext(ctx, "Deployment successful")
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

		l := log.With(
			slog.Any("parent", parent.Config.Coordinate),
			slog.Any("deploymentFailed", failed),
			slog.Any("child", childCfg.Coordinate),
			attribute.StatusDeploymentSkippedAttr())

		// after the first iteration
		var skipDeploymentWarning string
		if parent != root {
			l = l.With(slog.Any("root", root.Config.Coordinate))
			skipDeploymentWarning = fmt.Sprintf("Skipping deployment of %v, as it depends on %v which was not deployed after root dependency configuration %v %s", childCfg.Coordinate, parent.Config.Coordinate, root.Config.Coordinate, reason)
		} else {
			skipDeploymentWarning = fmt.Sprintf("Skipping deployment of %v, as it depends on %v which %s", childCfg.Coordinate, parent.Config.Coordinate, reason)
		}

		l.WarnContext(ctx, "%s", skipDeploymentWarning)
		report.GetReporterFromContextOrDiscard(ctx).ReportDeployment(childCfg.Coordinate, report.StateSkipped, []report.Detail{{Type: report.DetailTypeWarn, Message: skipDeploymentWarning}}, nil)

		removeChildren(ctx, child, root, configGraph, failed)

		configGraph.RemoveNode(child.ID())
	}
}

type ErrUnknownConfigType struct {
	configType config.TypeID
}

func (e ErrUnknownConfigType) Error() string {
	return fmt.Sprintf("unknown config type (ID: %q)", e.configType)
}

func deployConfig(ctx context.Context, c *config.Config, deployables resource.Deployables, resolvedEntities config.EntityLookup) (entities.ResolvedEntity, error) {
	if limiter := getDeploymentLimiterFromContext(ctx); limiter != nil {
		limiter.Acquire()
		defer limiter.Release()
	}

	if c.Skip {
		log.With(attribute.StatusDeploymentSkippedAttr()).InfoContext(ctx, "Skipping deployment of config")
		return entities.ResolvedEntity{}, errSkip // fake resolved entity that "old" deploy creates is never needed, as we don't even try to deploy dependencies of skipped configs (so no reference will ever be attempted to resolve)
	}

	properties, errs := c.ResolveParameterValues(resolvedEntities)
	if len(errs) > 0 {
		err := multierror.New(errs...)
		log.With(attribute.ErrorAttr(err), attribute.StatusDeploymentFailedAttr()).ErrorContext(ctx, "Invalid configuration - failed to resolve parameter values: %v", err)
		report.GetDetailerFromContextOrDiscard(ctx).Add(report.Detail{Type: report.DetailTypeError, Message: fmt.Sprintf("Failed to resolve parameter values: %v", err)})
		return entities.ResolvedEntity{}, err
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		log.With(attribute.ErrorAttr(err), attribute.StatusDeploymentFailedAttr()).ErrorContext(ctx, "Invalid configuration - failed to render JSON template: %v", err)
		report.GetDetailerFromContextOrDiscard(ctx).Add(report.Detail{Type: report.DetailTypeError, Message: fmt.Sprintf("Failed to render JSON template: %v", err)})
		return entities.ResolvedEntity{}, err
	}

	log.With(attribute.StatusDeployingAttr()).InfoContext(ctx, "Deploying config")
	var resolvedEntity entities.ResolvedEntity
	var deployErr error
	if deployable, ok := deployables[c.Type.ID()]; ok {
		resolvedEntity, deployErr = deployable.Deploy(ctx, properties, renderedConfig, c)
	} else {
		deployErr = ErrUnknownConfigType{configType: c.Type.ID()}
	}

	if deployErr != nil {
		var responseErr coreapi.APIError
		if errors.As(deployErr, &responseErr) {
			logResponseError(ctx, responseErr)
			return entities.ResolvedEntity{}, responseErr
		}

		log.With(attribute.ErrorAttr(deployErr)).ErrorContext(ctx, "Deployment failed - Monaco Error: %v", deployErr)
		return entities.ResolvedEntity{}, deployErr
	}
	return resolvedEntity, nil
}

// logResponseError prints user-friendly messages based on the response errors status
func logResponseError(ctx context.Context, responseErr coreapi.APIError) {
	if responseErr.StatusCode >= 400 && responseErr.StatusCode <= 499 {
		log.With(attribute.ErrorAttr(responseErr), attribute.StatusDeploymentFailedAttr()).ErrorContext(ctx, "Deployment failed - Dynatrace API rejected HTTP request / JSON data: %v", responseErr)
		report.GetDetailerFromContextOrDiscard(ctx).Add(report.Detail{Type: report.DetailTypeError, Message: fmt.Sprintf("Dynatrace API rejected request: : %v", responseErr)})
		return
	}

	if responseErr.StatusCode >= 500 && responseErr.StatusCode <= 599 {
		log.With(attribute.ErrorAttr(responseErr), attribute.StatusDeploymentFailedAttr()).ErrorContext(ctx, "Deployment failed - Dynatrace Server Error: %v", responseErr)
		return
	}

	log.With(attribute.ErrorAttr(responseErr), attribute.StatusDeploymentFailedAttr()).ErrorContext(ctx, "Deployment failed - Dynatrace API call unsuccessful: %v", responseErr)
}

func newContextWithEnvironment(ctx context.Context, env dynatrace.EnvironmentInfo) context.Context {
	return context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})
}
