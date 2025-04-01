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
	"os"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/id_extraction"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/segment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/slo"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	projectv2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
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
	onlyAPIs                bool
	onlySettings            bool
	onlyAutomation          bool
	onlyDocuments           bool
	onlyOpenPipeline        bool
	onlySegments            bool
	onlySLOsV2              bool
}

type auth struct {
	token, clientID, clientSecret string
}

func (a auth) mapToAuth() (*manifest.Auth, []error) {
	errs := make([]error, 0)
	mAuth := manifest.Auth{}

	if token, err := readEnvVariable(a.token); err != nil {
		errs = append(errs, err)
	} else {
		mAuth.Token = &token
	}

	if a.clientID != "" && a.clientSecret != "" {
		mAuth.OAuth = &manifest.OAuth{}
		if clientId, err := readEnvVariable(a.clientID); err != nil {
			errs = append(errs, err)
		} else {
			mAuth.OAuth.ClientID = clientId
		}
		if clientSecret, err := readEnvVariable(a.clientSecret); err != nil {
			errs = append(errs, err)
		} else {
			mAuth.OAuth.ClientSecret = clientSecret
		}
	}
	return &mAuth, errs
}

func readEnvVariable(envVar string) (manifest.AuthSecret, error) {
	var content string
	if envVar == "" {
		return manifest.AuthSecret{}, fmt.Errorf("unknown environment variable name")
	} else if content = os.Getenv(envVar); content == "" {
		return manifest.AuthSecret{}, fmt.Errorf("the content of the environment variable %q is not set", envVar)
	}
	return manifest.AuthSecret{Name: envVar, Value: secret.MaskedString(content)}, nil
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

	env, found := m.Environments[cmdOptions.specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment %q was not available in manifest %q", cmdOptions.specificEnvironmentName, cmdOptions.manifestFile)
	}

	ok := dynatrace.VerifyEnvironmentGeneration(ctx, manifest.Environments{env.Name: env})
	if !ok {
		return fmt.Errorf("unable to verify Dynatrace environment generation")
	}

	checkIfAbleToUploadToSameEnvironment(ctx, env)

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL:         env.URL,
			auth:                   env.Auth,
			outputFolder:           cmdOptions.outputFolder,
			projectName:            cmdOptions.projectName,
			forceOverwriteManifest: cmdOptions.forceOverwrite,
		},
		specificAPIs:     cmdOptions.specificAPIs,
		specificSchemas:  cmdOptions.specificSchemas,
		onlyAPIs:         cmdOptions.onlyAPIs,
		onlySettings:     cmdOptions.onlySettings,
		onlyAutomation:   cmdOptions.onlyAutomation,
		onlyDocuments:    cmdOptions.onlyDocuments,
		onlyOpenPipeline: cmdOptions.onlyOpenPipeline,
		onlySegment:      cmdOptions.onlySegments,
		onlySLOV2:        cmdOptions.onlySLOsV2,
	}

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
	a, errs := cmdOptions.auth.mapToAuth()
	errs = append(errs, validateParameters(cmdOptions.environmentURL, cmdOptions.projectName)...)

	if len(errs) > 0 {
		return printAndFormatErrors(errs, "not all necessary information is present to start downloading configurations")
	}

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: cmdOptions.environmentURL,
			},
			auth:                   *a,
			outputFolder:           cmdOptions.outputFolder,
			projectName:            cmdOptions.projectName,
			forceOverwriteManifest: cmdOptions.forceOverwrite,
		},
		specificAPIs:     cmdOptions.specificAPIs,
		specificSchemas:  cmdOptions.specificSchemas,
		onlyAPIs:         cmdOptions.onlyAPIs,
		onlySettings:     cmdOptions.onlySettings,
		onlyAutomation:   cmdOptions.onlyAutomation,
		onlyDocuments:    cmdOptions.onlyDocuments,
		onlyOpenPipeline: cmdOptions.onlyOpenPipeline,
	}

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
	downloadedConfigs, err := downloadConfigs(ctx, clientSet, apisToDownload, opts, defaultDownloadFn)
	if err != nil {
		return err
	}

	if len(downloadedConfigs) == 0 {
		log.Info("No configurations downloaded. No project will be created.")
		return nil
	}

	for c := range downloadedConfigs.AllConfigs {
		// We would need quite a huge refactoring to support Classic- and Automation-APIS here.
		// Automation already also does what we do here, but does set custom {{.variables}} that we can't easily escape here.
		// To fix this, it might be better do extract the variables at a later place instead of doing it before.
		if c.Type.ID() == config.ClassicApiTypeID || c.Type.ID() == config.AutomationTypeID {
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

type downloadFn struct {
	classicDownload      func(context.Context, client.ConfigClient, string, api.APIs, classic.ContentFilters) (projectv2.ConfigsPerType, error)
	settingsDownload     func(context.Context, client.SettingsClient, string, settings.Filters, ...config.SettingsType) (projectv2.ConfigsPerType, error)
	automationDownload   func(context.Context, client.AutomationClient, string, ...config.AutomationType) (projectv2.ConfigsPerType, error)
	bucketDownload       func(context.Context, client.BucketClient, string) (projectv2.ConfigsPerType, error)
	documentDownload     func(context.Context, client.DocumentClient, string) (projectv2.ConfigsPerType, error)
	openPipelineDownload func(context.Context, client.OpenPipelineClient, string) (projectv2.ConfigsPerType, error)
	segmentDownload      func(context.Context, segment.DownloadSegmentClient, string) (projectv2.ConfigsPerType, error)
	sloDownload          func(context.Context, slo.DownloadSloClient, string) (projectv2.ConfigsPerType, error)
}

var defaultDownloadFn = downloadFn{
	classicDownload:      classic.Download,
	settingsDownload:     settings.Download,
	automationDownload:   automation.Download,
	bucketDownload:       bucket.Download,
	documentDownload:     document.Download,
	openPipelineDownload: openpipeline.Download,
	segmentDownload:      segment.Download,
	sloDownload:          slo.Download,
}

func downloadConfigs(ctx context.Context, clientSet *client.ClientSet, apisToDownload api.APIs, opts downloadConfigsOptions, fn downloadFn) (project.ConfigsPerType, error) {
	configs := make(project.ConfigsPerType)
	if shouldDownloadConfigs(opts) {
		if opts.auth.Token == nil {
			return nil, errors.New("classic client config requires token")
		}
		classicCfgs, err := fn.classicDownload(ctx, clientSet.ConfigClient, opts.projectName, prepareAPIs(apisToDownload, opts), classic.ApiContentFilters)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, classicCfgs)
	}

	if shouldDownloadSettings(opts) {
		log.Info("Downloading settings objects")
		settingCfgs, err := fn.settingsDownload(ctx, clientSet.SettingsClient, opts.projectName, settings.DefaultSettingsFilters, makeSettingTypes(opts.specificSchemas)...)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, settingCfgs)
	}

	if shouldDownloadAutomationResources(opts) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading automation resources")
			automationCfgs, err := fn.automationDownload(ctx, clientSet.AutClient, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, automationCfgs)
		} else if opts.onlyAutomation {
			return nil, errors.New("can't download automation resources: no OAuth credentials configured")
		}
	}

	if shouldDownloadBuckets(opts) && opts.auth.OAuth != nil {
		log.Info("Downloading Grail buckets")
		bucketCfgs, err := fn.bucketDownload(ctx, clientSet.BucketClient, opts.projectName)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, bucketCfgs)
	}

	if shouldDownloadDocuments(opts) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading documents")
			documentCfgs, err := fn.documentDownload(ctx, clientSet.DocumentClient, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, documentCfgs)
		} else if opts.onlyDocuments {
			return nil, errors.New("can't download documents: no OAuth credentials configured")
		}
	}

	if featureflags.OpenPipeline.Enabled() {
		if shouldDownloadOpenPipeline(opts) {
			if opts.auth.OAuth != nil {
				openPipelineCfgs, err := fn.openPipelineDownload(ctx, clientSet.OpenPipelineClient, opts.projectName)
				if err != nil {
					return nil, err
				}
				copyConfigs(configs, openPipelineCfgs)
			} else if opts.onlyOpenPipeline {
				return nil, errors.New("can't download openpipeline resources: no OAuth credentials configured")
			}
		}
	}

	if featureflags.Segments.Enabled() {
		if shouldDownloadSegments(opts) {
			if opts.auth.OAuth != nil {
				segmentCgfs, err := fn.segmentDownload(ctx, clientSet.SegmentClient, opts.projectName)
				if err != nil {
					return nil, err
				}
				copyConfigs(configs, segmentCgfs)
			} else if opts.onlySegment {
				return nil, errors.New("can't download segment resources: no OAuth credentials configured")
			}
		}
	}

	if featureflags.ServiceLevelObjective.Enabled() {
		if shouldDownloadSLOsV2(opts) {
			if opts.auth.OAuth != nil {
				sloCgfs, err := fn.sloDownload(ctx, clientSet.ServiceLevelObjectiveClient, opts.projectName)
				if err != nil {
					return nil, err
				}
				copyConfigs(configs, sloCgfs)
			} else if opts.onlySLOV2 {
				return nil, fmt.Errorf("can't download %s resources: no OAuth credentials configured", config.ServiceLevelObjectiveID)
			}
		}
	}

	return configs, nil
}

func makeSettingTypes(specificSchemas []string) []config.SettingsType {
	var settingTypes []config.SettingsType
	for _, schema := range specificSchemas {
		settingTypes = append(settingTypes, config.SettingsType{SchemaId: schema})
	}
	return settingTypes
}

func copyConfigs(dest, src project.ConfigsPerType) {
	for k, v := range src {
		dest[k] = append(dest[k], v...)
	}
}

// shouldDownloadConfigs returns true unless onlySettings or specificSchemas but no specificAPIs are defined
func shouldDownloadConfigs(opts downloadConfigsOptions) bool {
	return (len(opts.specificSchemas) == 0 || len(opts.specificAPIs) > 0) &&
		!opts.onlyAutomation &&
		!opts.onlySettings &&
		!opts.onlyDocuments &&
		!opts.onlyOpenPipeline &&
		!opts.onlySegment &&
		!opts.onlySLOV2
}

// shouldDownloadSettings returns true unless onlyAPIs or specificAPIs but no specificSchemas are defined
func shouldDownloadSettings(opts downloadConfigsOptions) bool {
	return (len(opts.specificAPIs) == 0 || len(opts.specificSchemas) > 0) &&
		!opts.onlyAPIs &&
		!opts.onlyAutomation &&
		!opts.onlyDocuments &&
		!opts.onlyOpenPipeline &&
		!opts.onlySegment &&
		!opts.onlySLOV2
}

// shouldDownloadAutomationResources returns true unless download is limited to settings or config API types
func shouldDownloadAutomationResources(opts downloadConfigsOptions) bool {
	return !opts.onlyAPIs && len(opts.specificSchemas) == 0 &&
		!opts.onlySettings && len(opts.specificAPIs) == 0 &&
		!opts.onlyDocuments &&
		!opts.onlyOpenPipeline &&
		!opts.onlySegment &&
		!opts.onlySLOV2
}

// shouldDownloadBuckets returns true if download is not limited to another specific type
func shouldDownloadBuckets(opts downloadConfigsOptions) bool {
	return !opts.onlyAPIs && len(opts.specificAPIs) == 0 &&
		!opts.onlySettings && len(opts.specificSchemas) == 0 &&
		!opts.onlyAutomation &&
		!opts.onlyDocuments &&
		!opts.onlyOpenPipeline &&
		!opts.onlySegment &&
		!opts.onlySLOV2
}

func shouldDownloadDocuments(opts downloadConfigsOptions) bool {
	return !opts.onlyAPIs && len(opts.specificAPIs) == 0 && // only Config APIs requested
		!opts.onlySettings && len(opts.specificSchemas) == 0 && // only settings requested
		!opts.onlyAutomation &&
		!opts.onlyOpenPipeline &&
		!opts.onlySegment &&
		!opts.onlySLOV2
}

func shouldDownloadOpenPipeline(opts downloadConfigsOptions) bool {
	return !opts.onlyAPIs && len(opts.specificAPIs) == 0 && // only Config APIs requested
		!opts.onlySettings && len(opts.specificSchemas) == 0 && // only settings requested
		!opts.onlyAutomation &&
		!opts.onlyDocuments &&
		!opts.onlySegment &&
		!opts.onlySLOV2
}

func shouldDownloadSegments(opts downloadConfigsOptions) bool {
	return !opts.onlySettings && len(opts.specificSchemas) == 0 && // only settings requested
		!opts.onlyAPIs && len(opts.specificAPIs) == 0 && // only Config APIs requested
		!opts.onlyAutomation &&
		!opts.onlyDocuments &&
		!opts.onlyOpenPipeline &&
		!opts.onlySLOV2
}

func shouldDownloadSLOsV2(opts downloadConfigsOptions) bool {
	return !opts.onlySettings && len(opts.specificSchemas) == 0 && // only settings requested
		!opts.onlyAPIs && len(opts.specificAPIs) == 0 && // only Config APIs requested
		!opts.onlyAutomation &&
		!opts.onlyDocuments &&
		!opts.onlyOpenPipeline &&
		!opts.onlySegment
}
