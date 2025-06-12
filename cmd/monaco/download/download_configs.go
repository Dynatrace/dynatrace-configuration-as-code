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

package download

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/id_extraction"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
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

type downloadCmdOptions struct {
	projectName    string
	outputFolder   string
	forceOverwrite bool
	environmentURL string
	auth
	manifestFile            string
	specificEnvironmentName string
	specificAPIs            []string
	specificSchemas         []string
	onlyOptions             OnlyOptions
}

func (d downloadCmdOptions) toDownloadConfigsOptions(url manifest.URLDefinition, auth manifest.Auth) downloadConfigsOptions {
	return downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL:         url,
			auth:                   auth,
			outputFolder:           d.outputFolder,
			projectName:            d.projectName,
			forceOverwriteManifest: d.forceOverwrite,
		},
		specificAPIs:    d.specificAPIs,
		specificSchemas: d.specificSchemas,
		onlyOptions:     d.onlyOptions,
	}
}

func (d DefaultCommand) DownloadConfigsBasedOnManifest(ctx context.Context, fs afero.Fs, cmdOptions downloadCmdOptions) error {

	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
		Environments: []string{cmdOptions.specificEnvironmentName},
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	if len(errs) > 0 {
		err := printAndFormatErrors(errs, "failed to load manifest '%v'", cmdOptions.manifestFile)
		return err
	}

	env, found := m.Environments.SelectedEnvironments[cmdOptions.specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment '%s' is not defined in manifest '%s'", cmdOptions.specificEnvironmentName, cmdOptions.manifestFile)
	}

	if featureflags.VerifyEnvironmentType.Enabled() {
		if err := dynatrace.VerifyEnvironmentAuthentication(ctx, env); err != nil {
			return err
		}
	}

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	options := cmdOptions.toDownloadConfigsOptions(env.URL, env.Auth)
	if errs := options.valid(); len(errs) != 0 {
		err := printAndFormatErrors(errs, "command options are not valid")
		return err
	}

	clientSet, err := client.CreateClientSet(ctx, options.environmentURL.Value, options.auth)
	if err != nil {
		return err
	}

	return doDownloadConfigs(ctx, fs, clientSet, prepareAPIs(api.NewAPIs(), options), options)
}

func (d DefaultCommand) DownloadConfigs(ctx context.Context, fs afero.Fs, cmdOptions downloadCmdOptions) error {
	a, errs := cmdOptions.mapToAuth()
	errs = append(errs, validateParameters(cmdOptions.environmentURL, cmdOptions.projectName)...)

	if len(errs) > 0 {
		return printAndFormatErrors(errs, "not all necessary information is present to start downloading configurations")
	}

	options := cmdOptions.toDownloadConfigsOptions(
		manifest.URLDefinition{Type: manifest.ValueURLType, Value: cmdOptions.environmentURL}, *a)

	if errs := options.valid(); len(errs) != 0 {
		err := printAndFormatErrors(errs, "command options are not valid")
		return err
	}

	clientSet, err := client.CreateClientSet(ctx, options.environmentURL.Value, options.auth)
	if err != nil {
		return err
	}

	return doDownloadConfigs(ctx, fs, clientSet, prepareAPIs(api.NewAPIs(), options), options)
}

func doDownloadConfigs(ctx context.Context, fs afero.Fs, clientSet *client.ClientSet, apisToDownload api.APIs, opts downloadConfigsOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentURL.Value, opts.projectName)
	downloadedConfigs, err := downloadConfigs(ctx, clientSet, apisToDownload, opts)
	if err != nil {
		return err
	}

	if len(downloadedConfigs) == 0 {
		log.Info("No configurations downloaded. No project will be created.")
		return nil
	}

	for c := range downloadedConfigs.AllConfigs {
		// We would need quite a huge refactoring to support Classic- and Automation-APIS here.
		// Automation and Buckets already also does what we do here, but does set custom {{.variables}} that we can't easily escape here.
		// To fix this, it might be better do extract the variables at a later place instead of doing it before.
		if c.Type.ID() == config.ClassicApiTypeID || c.Type.ID() == config.AutomationTypeID || c.Type.ID() == config.BucketTypeID {
			continue
		}

		err := escapeGoTemplating(&c)
		if err != nil {
			log.WithFields(field.Coordinate(c.Coordinate), field.Error(err)).Warn("Failed to escape Go templating expressions. Template needs manual adaptation: %s", err)
		}
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs, err = dependency_resolution.ResolveDependencies(downloadedConfigs)
	if err != nil {
		return err
	}

	log.Info("Extracting additional identifiers into YAML parameters")
	// must happen after dep-resolution, as it removes IDs from the JSONs in which the dep-resolution searches as well
	downloadedConfigs, err = id_extraction.ExtractIDsIntoYAML(downloadedConfigs)
	if err != nil {
		return err
	}

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, fs)
}

func escapeGoTemplating(c *config.Config) error {
	content, err := c.Template.Content()
	if err != nil {
		return err
	}

	content = string(template.UseGoTemplatesForDoubleCurlyBraces([]byte(content)))

	err = c.Template.UpdateContent(content)
	if err != nil {
		return err
	}

	return nil
}

func downloadConfigs(ctx context.Context, clientSet *client.ClientSet, apisToDownload api.APIs, opts downloadConfigsOptions) (project.ConfigsPerType, error) {
	downloadables, err := prepareDownloadables(apisToDownload, opts, clientSet)
	if err != nil {
		return nil, err
	}

	configs := make(project.ConfigsPerType)
	for _, downloadable := range downloadables {
		currentConfigs, err := downloadable.Download(ctx, opts.projectName)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, currentConfigs)
	}

	return configs, nil
}

const oAuthSkipMsg = "Skipped downloading %s due to missing OAuth credentials"
const authSkipMsg = "Skipped downloading %s due to missing API token"

func prepareDownloadables(apisToDownload api.APIs, opts downloadConfigsOptions, clientSet *client.ClientSet) ([]resource.Downloadable, error) {
	downloadables := make([]resource.Downloadable, 0)

	if opts.onlyOptions.ShouldDownload(OnlyApisFlag) {
		if opts.auth.ApiToken != nil {
			downloadables = append(downloadables, classic.NewAPI(clientSet.ConfigClient, prepareAPIs(apisToDownload, opts), classic.ApiContentFilters))
		} else if opts.onlyOptions.IsSingleOption(OnlyApisFlag) {
			return nil, errors.New("classic client config requires API token")
		} else {
			log.Warn(authSkipMsg, "configuration objects")
		}
	}

	if opts.onlyOptions.ShouldDownload(OnlySettingsFlag) {
		// auth is already validated during load that either an API token or OAuth is set
		downloadables = append(downloadables, settings.NewAPI(clientSet.SettingsClient, settings.DefaultSettingsFilters, opts.specificSchemas))
	}

	if opts.onlyOptions.ShouldDownload(OnlyAutomationFlag) {
		if opts.auth.OAuth != nil {
			downloadables = append(downloadables, automation.NewAPI(clientSet.AutClient))
		} else if opts.onlyOptions.IsSingleOption(OnlyAutomationFlag) {
			return nil, errors.New("can't download automation resources: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "automation resources")
		}
	}

	if opts.onlyOptions.ShouldDownload(OnlyBucketsFlag) {
		if opts.auth.OAuth != nil {
			downloadables = append(downloadables, bucket.NewAPI(clientSet.BucketClient))
		} else if opts.onlyOptions.IsSingleOption(OnlyBucketsFlag) {
			return nil, errors.New("can't download buckets: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "Grail buckets")
		}
	}

	if opts.onlyOptions.ShouldDownload(OnlyDocumentsFlag) {
		if opts.auth.OAuth != nil {
			downloadables = append(downloadables, document.NewAPI(clientSet.DocumentClient))
		} else if opts.onlyOptions.IsSingleOption(OnlyDocumentsFlag) {
			return nil, errors.New("can't download documents: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "documents")
		}
	}

	if featureflags.OpenPipeline.Enabled() && opts.onlyOptions.ShouldDownload(OnlyOpenPipelineFlag) {
		if opts.auth.OAuth != nil {
			downloadables = append(downloadables, openpipeline.NewAPI(clientSet.OpenPipelineClient))
		} else if opts.onlyOptions.IsSingleOption(OnlyOpenPipelineFlag) {
			return nil, errors.New("can't download openpipeline resources: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "openpipelines")
		}
	}

	if featureflags.Segments.Enabled() && opts.onlyOptions.ShouldDownload(OnlySegmentsFlag) {
		if opts.auth.OAuth != nil {
			downloadables = append(downloadables, segment.NewAPI(clientSet.SegmentClient))
		} else if opts.onlyOptions.IsSingleOption(OnlySegmentsFlag) {
			return nil, errors.New("can't download segment resources: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "segments")
		}
	}

	if featureflags.ServiceLevelObjective.Enabled() && opts.onlyOptions.ShouldDownload(OnlySloV2Flag) {
		if opts.auth.OAuth != nil {
			downloadables = append(downloadables, slo.NewAPI(clientSet.ServiceLevelObjectiveClient))
		} else if opts.onlyOptions.IsSingleOption(OnlySloV2Flag) {
			return nil, fmt.Errorf("can't download %s resources: no OAuth credentials configured", config.ServiceLevelObjectiveID)
		} else {
			log.Warn(oAuthSkipMsg, "SLO-V2")
		}
	}

	return downloadables, nil
}

func copyConfigs(dest, src project.ConfigsPerType) {
	for k, v := range src {
		dest[k] = append(dest[k], v...)
	}
}
