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
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/client"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/classic"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/settings"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/spf13/afero"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	defaultConcurrentDownloads = 50
	concurrentRequestsEnvKey   = "CONCURRENT_REQUESTS"
)

//go:generate mockgen -source=download.go -destination=download_mock.go -package=download -write_package_comment=false Command

// Command is used to test the CLi commands properly without executing the actual monaco download.
//
// The actual implementations are in the [DefaultCommand] struct.
type Command interface {
	DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions manifestDownloadOptions) error
	DownloadConfigs(fs afero.Fs, cmdOptions directDownloadOptions) error
}

// DefaultCommand is used to implement the [Command] interface.
type DefaultCommand struct{}

// make sure DefaultCommand implements the Command interface
var (
	_ Command = (*DefaultCommand)(nil)
)

type downloadCommandOptions struct {
	projectName     string
	outputFolder    string
	forceOverwrite  bool
	specificAPIs    []string
	specificSchemas []string
	skipSettings    bool
}

type manifestDownloadOptions struct {
	manifestFile            string
	specificEnvironmentName string
	downloadCommandOptions
}

type directDownloadOptions struct {
	environmentUrl, envVarName string
	downloadCommandOptions
}

func (d DefaultCommand) DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions manifestDownloadOptions) error {

	man, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
	})

	if errs != nil {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to load manifest '%v'", cmdOptions.manifestFile)
	}

	env, found := man.Environments[cmdOptions.specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment '%v' was not available in manifest '%v'", cmdOptions.specificEnvironmentName, cmdOptions.manifestFile)
	}

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to load apis")
	}

	u, err := env.GetUrl()
	if err != nil {
		errs = append(errs, err)
	}

	token, err := env.GetToken()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to load manifest data")
	}

	tokenEnvVar := fmt.Sprintf("TOKEN_%s", strings.ToUpper(cmdOptions.projectName))
	if envVarToken, ok := env.Token.(*manifest.EnvironmentVariableToken); ok {
		tokenEnvVar = envVarToken.EnvironmentVariableName
	}

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	concurrentDownloadLimit := concurrentRequestLimitFromEnv()

	options := downloadOptions{
		environmentUrl:          u,
		token:                   token,
		tokenEnvVarName:         tokenEnvVar,
		outputFolder:            cmdOptions.outputFolder,
		projectName:             cmdOptions.projectName,
		forceOverwriteManifest:  cmdOptions.forceOverwrite,
		specificAPIs:            cmdOptions.specificAPIs,
		specificSchemas:         cmdOptions.specificSchemas,
		clientProvider:          client.NewDynatraceClient,
		concurrentDownloadLimit: concurrentDownloadLimit,
		skipSettings:            cmdOptions.skipSettings,
	}
	return doDownload(fs, api.NewApis(), options)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, cmdOptions directDownloadOptions) error {
	token := os.Getenv(cmdOptions.envVarName)
	concurrentDownloadLimit := concurrentRequestLimitFromEnv()
	errors := validateParameters(cmdOptions.envVarName, cmdOptions.environmentUrl, cmdOptions.projectName, token)

	if len(errors) > 0 {
		util.PrintErrors(errors)

		return fmt.Errorf("not all necessary information is present to start downloading configurations")
	}

	options := downloadOptions{
		environmentUrl:          cmdOptions.environmentUrl,
		token:                   token,
		tokenEnvVarName:         cmdOptions.envVarName,
		outputFolder:            cmdOptions.outputFolder,
		projectName:             cmdOptions.projectName,
		forceOverwriteManifest:  cmdOptions.forceOverwrite,
		specificAPIs:            cmdOptions.specificAPIs,
		specificSchemas:         cmdOptions.specificSchemas,
		clientProvider:          client.NewDynatraceClient,
		concurrentDownloadLimit: concurrentDownloadLimit,
		skipSettings:            cmdOptions.skipSettings,
	}
	return doDownload(fs, api.NewApis(), options)
}

type DynatraceClientProvider func(string, string, ...func(*client.DynatraceClient)) (*client.DynatraceClient, error)

type downloadOptions struct {
	environmentUrl          string
	token                   string
	tokenEnvVarName         string
	outputFolder            string
	projectName             string
	specificAPIs            []string
	specificSchemas         []string
	forceOverwriteManifest  bool
	clientProvider          DynatraceClientProvider
	concurrentDownloadLimit int
	skipSettings            bool
}

func (c downloadOptions) getDynatraceClient() (client.Client, error) {
	return c.clientProvider(c.environmentUrl, c.token)
}

func doDownload(fs afero.Fs, apis api.ApiMap, opts downloadOptions) error {
	errors := validateOutputFolder(fs, opts.outputFolder, opts.projectName)
	if len(errors) > 0 {
		util.PrintErrors(errors)
		return fmt.Errorf("output folder is invalid")
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentUrl, opts.projectName)
	downloadedConfigs, err := downloadConfigs(apis, opts)
	if err != nil {
		return err
	}

	if numConfigs := sumConfigs(downloadedConfigs); numConfigs > 0 {
		log.Info("Downloaded %d configurations.", numConfigs)
	} else {
		log.Info("No configurations were found. No files will be created.")
		return nil
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs = download.ResolveDependencies(downloadedConfigs)

	proj := download.CreateProjectData(downloadedConfigs, opts.projectName)

	downloadWriterContext := download.WriterContext{
		ProjectToWrite:         proj,
		TokenEnvVarName:        opts.tokenEnvVarName,
		EnvironmentUrl:         opts.environmentUrl,
		OutputFolder:           opts.outputFolder,
		ForceOverwriteManifest: opts.forceOverwriteManifest,
	}
	err = download.WriteToDisk(fs, downloadWriterContext)
	if err != nil {
		return err
	}

	if depErr := reportForCircularDependencies(proj); depErr != nil {
		log.Warn("Download finished with problems: %s", depErr)
	} else {
		log.Info("Finished download")
	}

	return nil
}

func reportForCircularDependencies(p project.Project) error {
	_, errs := topologysort.GetSortedConfigsForEnvironments([]project.Project{p}, []string{p.Id})
	if len(errs) != 0 {
		util.PrintWarnings(errs)
		return fmt.Errorf("there are circular dependencies between %d configurations that need to be resolved manually", len(errs))
	}
	return nil
}

func downloadConfigs(apis api.ApiMap, opts downloadOptions) (project.ConfigsPerType, error) {
	c, err := opts.getDynatraceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	c = client.LimitClientParallelRequests(c, opts.concurrentDownloadLimit)

	apisToDownload, errors := getApisToDownload(apis, opts.specificAPIs)
	if len(errors) > 0 {
		util.PrintErrors(errors)
		return nil, fmt.Errorf("failed to load apis")
	}

	configObjects := make(project.ConfigsPerType)

	// download specific APIs only
	if len(opts.specificAPIs) > 0 {
		log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
		cfgs := classic.DownloadAllConfigs(apisToDownload, c, opts.projectName)
		maps.Copy(configObjects, cfgs)
	}

	// download specific settings only
	if len(opts.specificSchemas) > 0 {
		log.Debug("Settings to download: \n - %v", strings.Join(opts.specificSchemas, "\n - "))
		s := settings.Download(c, opts.specificSchemas, opts.projectName)
		maps.Copy(configObjects, s)
	}

	// return specific download objects
	if len(opts.specificSchemas) > 0 || len(opts.specificAPIs) > 0 {
		return configObjects, nil
	}

	// if nothing was specified specifically, lets download all configs and settings
	log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
	configObjects = classic.DownloadAllConfigs(apisToDownload, c, opts.projectName)
	if !opts.skipSettings {
		settingsObjects := settings.DownloadAll(c, opts.projectName)
		maps.Copy(configObjects, settingsObjects)
	}
	return configObjects, nil
}

func sumConfigs(configs project.ConfigsPerType) int {
	sum := 0

	for _, v := range configs {
		sum += len(v)
	}

	return sum
}

// validateParameters checks that all necessary variables have been set.
func validateParameters(envVarName, environmentUrl, projectName, token string) []error {
	errors := make([]error, 0)

	if envVarName == "" {
		errors = append(errors, fmt.Errorf("token not specified"))
	} else if token == "" {
		errors = append(errors, fmt.Errorf("the content of token '%v' is not set", envVarName))
	}

	if environmentUrl == "" {
		errors = append(errors, fmt.Errorf("url not specified"))
	}

	if _, err := url.Parse(environmentUrl); err != nil {
		errors = append(errors, fmt.Errorf("url is invalid: %w", err))
	}

	if projectName == "" {
		errors = append(errors, fmt.Errorf("project not specified"))
	}

	return errors
}

func validateOutputFolder(fs afero.Fs, outputFolder, project string) []error {
	errors := make([]error, 0)

	errors = append(errors, validateFolder(fs, outputFolder)...)
	if len(errors) > 0 {
		return errors
	}
	errors = append(errors, validateFolder(fs, path.Join(outputFolder, project))...)
	return errors
}

func validateFolder(fs afero.Fs, path string) []error {
	errors := make([]error, 0)
	exists, err := afero.Exists(fs, path)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to check if output folder '%s' exists: %w", path, err))
	}
	if exists {
		isDir, err := afero.IsDir(fs, path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to check if output folder '%s' is a folder: %w", path, err))
		}
		if !isDir {
			errors = append(errors, fmt.Errorf("unable to write to '%s': file exists and is not a directory", path))
		}
	}

	return errors
}

// Get all v2 apis and filter for the selected ones
func getApisToDownload(apis api.ApiMap, specificAPIs []string) (api.ApiMap, []error) {
	var errors []error

	apisToDownload, unknownApis := apis.FilterApisByName(specificAPIs)
	if len(unknownApis) > 0 {
		errors = append(errors, fmt.Errorf("APIs '%v' are not known. Please consult our documentation for known API-names", strings.Join(unknownApis, ",")))
	}

	if len(specificAPIs) == 0 {
		var deprecated api.ApiMap
		apisToDownload, deprecated = apisToDownload.Filter(deprecatedEndpointFilter)
		for _, d := range deprecated {
			log.Warn("API '%s' is deprecated by '%s' and will be skip", d.GetId(), d.DeprecatedBy())
		}
	}

	apisToDownload, filtered := apisToDownload.Filter(func(api api.Api) bool {
		return api.ShouldSkipDownload()
	})

	if len(filtered) > 0 {
		keys := strings.Join(maps.Keys(filtered), ", ")
		log.Info("APIs that won't be downloaded and need manual creation: '%v'.", keys)
	}

	if len(apisToDownload) == 0 {
		errors = append(errors, fmt.Errorf("no APIs to download"))
	}

	return apisToDownload, errors
}

func deprecatedEndpointFilter(api api.Api) bool {
	return api.DeprecatedBy() != ""
}

func concurrentRequestLimitFromEnv() int {
	limit, err := strconv.Atoi(os.Getenv(concurrentRequestsEnvKey))
	if err != nil || limit < 0 {
		limit = defaultConcurrentDownloads
	}
	return limit
}
