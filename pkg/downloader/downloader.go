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

package downloader

import (
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	configv2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	refParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/downloader/dynatraceparser"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectv2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

//DownloadConfigs get configurations from Dynatrace API and transforms them into objects
func DownloadConfigs(env manifest.EnvironmentDefinition, listApis map[string]api.Api, projectId string) (projectv2.ConfigsPerApisPerEnvironments, []error) {
	var errors []error
	result := make(projectv2.ConfigsPerApisPerEnvironments)
	result[env.Name] = make(map[string][]configv2.Config)

	token, err := env.GetToken()
	if err != nil {
		log.Error("error retrieving token for environment %v %v", env.Name, err)
		errors = append(errors, err)
		return nil, errors
	}
	client, err := rest.NewDynatraceClient(env.Url, token)
	if err != nil {
		log.Error("error creating dynatrace client for environment %v %v", env.Name, err)
		errors = append(errors, err)
		return nil, errors
	}
	for _, api := range listApis {
		creator := dynatraceparser.NewDynatraceParser()
		configs, errs := createConfigforTargetApi(env, creator, client, api, projectId)
		//apiconfig, err := createConfigFromApiV2(env, api, projectId, token, client)
		if len(errs) > 0 {
			log.Error("error getting configs for env %v %v", env.Name, errs[0])
			errors = append(errors, errs[0])
			continue
		}
		result[env.Name][api.GetId()] = append(result[env.Name][api.GetId()], configs...)
	}
	result[env.Name] = solveDependencies(projectId, result[env.Name])
	if errors != nil {
		return nil, errors
	}
	return result, nil
}
func GetDependencies() map[string][]string {
	dependenciesPerEnvironment := make(map[string][]string)
	return dependenciesPerEnvironment
}
func solveDependencies(projectId string, configs map[string][]configv2.Config) map[string][]configv2.Config {
	//temp object to hold the ids and coordinates
	tempConfig := make(map[string]coordinate.Coordinate)
	//builds temp object
	for _, configType := range configs {
		for _, config := range configType {
			tempConfig[config.Coordinate.DynatraceId] = config.Coordinate
		}
	}
	//Brute force for now, TODO find an optimize way to explore files
	for _, configType := range configs {
		for _, config := range configType {
			content := config.Template.Content()
			for key, val := range tempConfig {
				exists := strings.Contains(content, key)
				if exists {
					log.Debug(val.Api)
					//WARNING: validate multiple dependencies in 1 config!!
					//should update template and coordinate
					config = newDependencyConfig(projectId, config, val)
				}
			}
		}
	}
	return configs
}
func newDependencyConfig(projectId string, config configv2.Config, dependencyCoor coordinate.Coordinate) configv2.Config {
	newContent := strings.ReplaceAll(config.Template.Content(), dependencyCoor.DynatraceId, dependencyCoor.Config)
	idProperty := "id" //since all depends on id to relative configs
	ref := refParam.New(dependencyCoor.Project, dependencyCoor.Api, dependencyCoor.Config, idProperty)
	config.Template.UpdateContent(newContent)
	parameterName := dependencyCoor.Api + "-" + dependencyCoor.Config + "-id" //this makes dependency on multiple same config types possible
	config.Parameters[parameterName] = ref
	return config
}
func createConfigforTargetApi(env manifest.EnvironmentDefinition, creator dynatraceparser.DynatraceParser,
	client rest.DynatraceClient, api api.Api, projectId string) ([]configv2.Config, []error) {
	var errors []error
	var configs []configv2.Config

	values, err := client.List(api)
	if err != nil {
		log.Error("error getting client list from api %v %v", api.GetId(), err)
		errors = append(errors, err)
		return nil, errors
	}
	if len(values) == 0 {
		log.Info("No elements for API %s", api.GetId())
		errors = append(errors, err)
		return nil, errors
	}
	for _, val := range values {
		log.Debug("getting detail %s", val)
		subPath := filepath.Join(projectId, api.GetId())
		config, filter, err := createSingleConfig(env, creator, client, api, val, projectId, subPath)
		if err != nil {
			log.Error("error creating config api json file: %v", err)
			continue
		}
		if filter {
			continue
		}
		configs = append(configs, config)
	}
	return configs, nil
}
func createSingleConfig(env manifest.EnvironmentDefinition, creator dynatraceparser.DynatraceParser,
	client rest.DynatraceClient, api api.Api, val api.Value, projectId string, path string) (configv2.Config, bool, error) {

	templ, params, dynatraceId, filter, err := getTemplate(client, creator, api, val, path)
	coord := coordinate.Coordinate{
		Project:     projectId,
		Api:         api.GetId(),
		Config:      val.Name,
		DynatraceId: dynatraceId,
	}
	if err != nil {
		return configv2.Config{}, false, err
	}
	if filter {
		return configv2.Config{}, true, nil
	}
	var references []coordinate.Coordinate
	return configv2.Config{
		Template:    templ,
		Coordinate:  coord,
		Group:       env.Group,
		Environment: env.Name,
		Parameters:  params,
		References:  references,
		Skip:        false,
	}, false, nil
}
func getTemplate(client rest.DynatraceClient, creator dynatraceparser.DynatraceParser, api api.Api,
	val api.Value, path string) (template.Template, map[string]parameter.Parameter, string, bool, error) {

	file, dynatraceId, fullpath, name, filter, err := creator.GetConfig(client, api, val, path)
	if err != nil {
		log.Error("error getting config %s: %v", name, err)
		return nil, nil, "", false, err
	}
	if filter {
		return nil, nil, "", true, nil
	}

	templ, err := template.CreateFileBasedTemplateFromString(fullpath, file)

	if err != nil {
		log.Error("error creating config %s: %v", name, err)
		return nil, nil, "", false, err
	}
	params := map[string]parameter.Parameter{}
	params["name"] = &valueParam.ValueParameter{
		Value: name,
	}

	// if len(errors) > 0 {
	// 	return nil, nil,false, errors
	// }

	return templ, params, dynatraceId, false, nil
}
