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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/document"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/id_extraction"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/slo"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/segment"
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
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyOptions:     cmdOptions.onlyOptions,
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
	a, errs := cmdOptions.mapToAuth()
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
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyOptions:     cmdOptions.onlyOptions,
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

type Downloadable interface {

	// Download returns downloaded project.ConfigsPerType, and an error, if something went wrong during the download.
	// The string projectName is used to set the Project attribute of each downloaded config.
	Download(ctx context.Context, projectName string) (project.ConfigsPerType, error)
}

type downloadFn struct {
	classicDownload      func(context.Context, client.ConfigClient, string, api.APIs, classic.ContentFilters) (project.ConfigsPerType, error)
	settingsDownload     func(context.Context, client.SettingsClient, string, settings.Filters, ...config.SettingsType) (project.ConfigsPerType, error)
	automationDownload   func(context.Context, client.AutomationClient, string, ...config.AutomationType) (project.ConfigsPerType, error)
	bucketDownload       func(client.BucketClient) Downloadable
	documentDownload     func(context.Context, client.DocumentClient, string) (project.ConfigsPerType, error)
	openPipelineDownload func(context.Context, client.OpenPipelineClient, string) (project.ConfigsPerType, error)
	segmentDownload      func(segment.Source) Downloadable
	sloDownload          func(context.Context, slo.DownloadSloClient, string) (project.ConfigsPerType, error)
}

var defaultDownloadFn = downloadFn{
	classicDownload:    classic.Download,
	settingsDownload:   settings.Download,
	automationDownload: automation.Download,
	bucketDownload: func(bucketClient client.BucketClient) Downloadable {
		return bucket.NewBucketAPI(bucketClient)
	},
	documentDownload:     document.Download,
	openPipelineDownload: openpipeline.Download,
	segmentDownload: func(source segment.Source) Downloadable {
		return segment.NewSegmentAPI(source)
	},
	sloDownload: slo.Download,
}

const oAuthSkipMsg = "Skipped downloading %s due to missing OAuth credentials"
const authSkipMsg = "Skipped downloading %s due to missing token"

func downloadConfigs(ctx context.Context, clientSet *client.ClientSet, apisToDownload api.APIs, opts downloadConfigsOptions, fn downloadFn) (project.ConfigsPerType, error) {
	configs := make(project.ConfigsPerType)
	if opts.onlyOptions.ShouldDownload(OnlyApis) {
		if opts.auth.Token != nil {
			log.Info("Downloading configuration objects")
			classicCfgs, err := fn.classicDownload(ctx, clientSet.ConfigClient, opts.projectName, prepareAPIs(apisToDownload, opts), classic.ApiContentFilters)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, classicCfgs)
		} else if opts.onlyOptions.IsSingleOption(OnlyApis) {
			return nil, errors.New("classic client config requires token")
		} else {
			log.Warn(authSkipMsg, "configuration objects")
		}
	}

	if opts.onlyOptions.ShouldDownload(OnlySettings) {
		// auth is already validated during load that either token or OAuth is set
		log.Info("Downloading settings objects")
		settingCfgs, err := fn.settingsDownload(ctx, clientSet.SettingsClient, opts.projectName, settings.DefaultSettingsFilters, makeSettingTypes(opts.specificSchemas)...)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, settingCfgs)
	}

	if opts.onlyOptions.ShouldDownload(OnlyAutomation) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading automation resources")
			automationCfgs, err := fn.automationDownload(ctx, clientSet.AutClient, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, automationCfgs)
		} else if opts.onlyOptions.IsSingleOption(OnlyAutomation) {
			return nil, errors.New("can't download automation resources: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "automation resources")
		}
	}

	if opts.onlyOptions.ShouldDownload(OnlyBuckets) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading Grail buckets")
			bucketCfgs, err := fn.bucketDownload(clientSet.BucketClient).Download(ctx, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, bucketCfgs)
		} else if opts.onlyOptions.IsSingleOption(OnlyBuckets) {
			return nil, errors.New("can't download buckets: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "Grail buckets")
		}
	}

	if opts.onlyOptions.ShouldDownload(OnlyDocuments) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading documents")
			documentCfgs, err := fn.documentDownload(ctx, clientSet.DocumentClient, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, documentCfgs)
		} else if opts.onlyOptions.IsSingleOption(OnlyDocuments) {
			return nil, errors.New("can't download documents: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "documents")
		}
	}

	if featureflags.OpenPipeline.Enabled() && opts.onlyOptions.ShouldDownload(OnlyOpenPipeline) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading openpipelines")
			openPipelineCfgs, err := fn.openPipelineDownload(ctx, clientSet.OpenPipelineClient, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, openPipelineCfgs)
		} else if opts.onlyOptions.IsSingleOption(OnlyOpenPipeline) {
			return nil, errors.New("can't download openpipeline resources: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "openpipelines")
		}
	}

	if featureflags.Segments.Enabled() && opts.onlyOptions.ShouldDownload(OnlySegments) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading segments")
			segmentCgfs, err := fn.segmentDownload(clientSet.SegmentClient).Download(ctx, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, segmentCgfs)
		} else if opts.onlyOptions.IsSingleOption(OnlySegments) {
			return nil, errors.New("can't download segment resources: no OAuth credentials configured")
		} else {
			log.Warn(oAuthSkipMsg, "segments")
		}
	}

	if featureflags.ServiceLevelObjective.Enabled() && opts.onlyOptions.ShouldDownload(OnlySloV2) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading SLO-V2")
			sloCgfs, err := fn.sloDownload(ctx, clientSet.ServiceLevelObjectiveClient, opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, sloCgfs)
		} else if opts.onlyOptions.IsSingleOption(OnlySloV2) {
			return nil, fmt.Errorf("can't download %s resources: no OAuth credentials configured", config.ServiceLevelObjectiveID)
		} else {
			log.Warn(oAuthSkipMsg, "SLO-V2")
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
