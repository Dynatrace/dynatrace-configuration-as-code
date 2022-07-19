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
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/jsoncreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/yamlcreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
)

var cont = 0

//GetConfigsFilterByEnvironment filters the enviroments list based on specificEnvironment flag value
func GetConfigsFilterByEnvironment(workingDir string, fs afero.Fs, environmentsFile string,
	specificEnvironment string, downloadSpecificAPI string) error {
	environments, errors := environment.LoadEnvironmentList(specificEnvironment, environmentsFile, fs)
	if len(errors) > 0 {
		for _, err := range errors {
			util.Log.Error("Error while getting enviroments ", err)
		}
		return fmt.Errorf("There were some errors while getting environment files")
	}
	return getConfigs(fs, workingDir, environments, downloadSpecificAPI)

}

//getConfigs Entry point that retrieves the specified configurations from a Dynatrace tenant
func getConfigs(fs afero.Fs, workingDir string, environments map[string]environment.Environment, downloadSpecificAPI string) error {
	list, err := getAPIList(downloadSpecificAPI)
	if err != nil {
		return err
	}
	isError := false
	for _, environment := range environments {
		//download configs for each environment
		err := downloadConfigFromEnvironment(fs, environment, workingDir, list)
		if err != nil {
			util.Log.Error("error while downloading configs for environment %v %v", environment.GetId())
			isError = true
		}
	}
	if isError {
		return fmt.Errorf("There were some errors while downloading the environment configs, please check the logs")
	}
	return nil

}

//returns the list of API filter if the download specific flag is used, otherwise returns all the API's
func getAPIList(downloadSpecificAPI string) (filterAPIList map[string]api.Api, err error) {
	availableApis := api.NewApis()
	noFilterAPIListProvided := strings.TrimSpace(downloadSpecificAPI) == ""
	filterAPIList = make(map[string]api.Api)

	// If no filter provided, return only non deprecated APIs
	if noFilterAPIListProvided {
		for availableApiId, availableApi := range availableApis {
			isDeprecated := availableApi.IsDeprecatedApi()
			if !isDeprecated {
				filterAPIList[availableApiId] = availableApi
			}
		}

		return filterAPIList, nil
	}

	requestedApis := strings.Split(downloadSpecificAPI, ",")
	isErr := false
	for _, id := range requestedApis {
		cleanAPI := strings.TrimSpace(id)
		isAPI := api.IsApi(cleanAPI)

		if !isAPI {
			util.Log.Error("Value %s is not a valid API name", cleanAPI)
			isErr = true
		} else {
			filterAPI := availableApis[cleanAPI]
			isDeprecatedApi := filterAPI.IsDeprecatedApi()

			if isDeprecatedApi {
				isDeprecatedBy := filterAPI.IsDeprecatedBy()
				util.Log.Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", cleanAPI, isDeprecatedBy)
			}

			filterAPIList[cleanAPI] = filterAPI
		}
	}
	if isErr {
		return nil, fmt.Errorf("There were some errors in the API list provided")
	}

	return filterAPIList, nil
}

//creates the project and downloads the configs
func downloadConfigFromEnvironment(fs afero.Fs, environment environment.Environment, basepath string, listApis map[string]api.Api) (err error) {
	environmentName := environment.GetId()
	path := filepath.Join(basepath, environmentName)

	util.Log.Info("Creating base project name %s", environmentName)
	err = fs.MkdirAll(path, 0777)
	if err != nil {
		util.Log.Error("error creating folder for enviroment %v %v", environmentName, err)
		return err
	}
	token, err := environment.GetToken()
	if err != nil {
		util.Log.Error("error retrieving token for enviroment %v %v", environmentName, err)
		return err
	}
	client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
	if err != nil {
		util.Log.Error("error creating dynatrace client for enviroment %v %v", environmentName, err)
		return err
	}
	for _, api := range listApis {
		util.Log.Info(" --- GETTING CONFIGS for %s", api.GetId())
		jcreator := jsoncreator.NewJSONCreator()
		ycreator := yamlcreator.NewYamlConfig(environmentName)

		// Retrieves object from single configuration API
		isSingleConfigurationApi := api.IsSingleConfigurationApi()
		if isSingleConfigurationApi {
			errorAPI := createConfigsFromSingleConfigurationAPI(fs, api, path, client, jcreator, ycreator)
			if errorAPI != nil {
				util.Log.Error("error getting configs from API %v %v", api.GetId())
			}
		} else {
			errorAPI := createConfigsFromAPI(fs, api, path, client, jcreator, ycreator)
			if errorAPI != nil {
				util.Log.Error("error getting configs from API %v %v", api.GetId())
			}
		}
	}
	util.Log.Info("END downloading info %s", environmentName)
	return nil
}

func createConfigsFolder(
	fs afero.Fs,
	api api.Api,
	fullpath string,
) (subPath string, err error) {
	subPath = filepath.Join(fullpath, api.GetId())
	return subPath, fs.MkdirAll(subPath, 0777)
}

func createConfigsFromSingleConfigurationAPI(
	fs afero.Fs,
	api api.Api,
	fullpath string,
	client rest.DynatraceClient,
	jcreator jsoncreator.JSONCreator,
	ycreator yamlcreator.YamlCreator,
) (err error) {
	subPath, err := createConfigsFolder(fs, api, fullpath)
	if err != nil {
		util.Log.Error("error creating folder for api %v %v", api.GetId(), err)
		return err
	}

	apiId := api.GetId()

	err = ycreator.ReadYamlFile(fs, subPath, apiId)
	if err != nil {
		util.Log.Debug("No previously downloaded configs found.")
	}

	// Single configuration entities don't have name or id. Therefore, configuration type
	// used for the API's id.
	// In the context of downloading this configuration type is used as entityId and entityName.
	entityId := api.GetId()
	entityName := entityId

	configId, err := getConfigId(entityId, entityName, api)
	if err != nil {
		util.Log.Error("error creating config id: %v", err)
		return err
	}

	jsonConfigFileName := ycreator.GetConfigFileName(configId)

	isKnownConfigFileName := jsonConfigFileName != ""
	if !isKnownConfigFileName {
		jsonConfigFileName = configId + ".json"
	}

	jsonConfigFilePath := filepath.Join(subPath, jsonConfigFileName)

	// At this point all API specific substitutions are made (e.g. reports name = dashboard id)
	filter, err := jcreator.CreateJSONConfig(fs, client, api, entityId, jsonConfigFilePath)
	if err != nil {
		util.Log.Error("error creating config api json file: %v", err)
		return err
	}
	if filter {
		return nil
	}

	isNonUniqueNameApi := false
	ycreator.UpdateConfig(entityId, entityName, configId, isNonUniqueNameApi, jsonConfigFileName)

	err = ycreator.WriteYamlFile(fs, subPath, api.GetId())
	if err != nil {
		util.Log.Error("error creating config api yaml file: %v", err)
		return err
	}

	return nil
}

func createConfigsFromAPI(
	fs afero.Fs,
	api api.Api,
	fullpath string,
	client rest.DynatraceClient,
	jcreator jsoncreator.JSONCreator,
	ycreator yamlcreator.YamlCreator,
) (err error) {
	//retrieves all objects for the specific api
	values, err := client.List(api)
	if err != nil {
		util.Log.Error("error getting client list from api %v %v", api.GetId(), err)
		return err
	}
	if len(values) == 0 {
		util.Log.Info("No elements for API %s", api.GetId())
		return nil
	}

	subPath, err := createConfigsFolder(fs, api, fullpath)
	if err != nil {
		util.Log.Error("error creating folder for api %v %v", api.GetId(), err)
		return err
	}

	apiId := api.GetId()

	err = ycreator.ReadYamlFile(fs, subPath, apiId)
	if err != nil {
		util.Log.Debug("No previously downloaded configs found.")
	}

	for _, val := range values {
		util.Log.Debug("Getting details for id %s, name %s", val.Id, val.Name)
		cont++
		util.Log.Debug("REQUEST counter %v", cont)

		configId, err := getConfigId(val.Id, val.Name, api)
		if err != nil {
			util.Log.Error("error creating config id: %v", err)
			continue
		}

		jsonConfigFileName := ycreator.GetConfigFileName(configId)

		isKnownConfigFileName := jsonConfigFileName != ""
		if !isKnownConfigFileName {
			jsonConfigFileName = configId + ".json"
		}

		jsonConfigFilePath := filepath.Join(subPath, jsonConfigFileName)

		// At this point all API specific substitutions are made (e.g. reports name = dashboard id)
		filter, err := jcreator.CreateJSONConfig(fs, client, api, val.Id, jsonConfigFilePath)
		if err != nil {
			util.Log.Error("error creating config api json file: %v", err)
			continue
		}
		if filter {
			continue
		}

		isNonUniqueNameApi := api.IsNonUniqueNameApi()
		ycreator.UpdateConfig(val.Id, val.Name, configId, isNonUniqueNameApi, jsonConfigFileName)
	}

	err = ycreator.WriteYamlFile(fs, subPath, apiId)
	if err != nil {
		util.Log.Error("error creating config api yaml file: %v", err)
		return err
	}

	return nil
}

// Retrieves and sanitizes config id which is used as a unique identifier of
// config details and as a reference to the config's json file
func getConfigId(id string, name string, api api.Api) (configId string, err error) {
	isNonUniqueNameApi := api.IsNonUniqueNameApi()
	if isNonUniqueNameApi {
		configId = util.SanitizeName(id)
		return configId, nil
	}

	//for the apis that return a name for the config
	if name != "" {
		configId = util.SanitizeName(name)
		return configId, nil
	}

	return "", fmt.Errorf("error getting configId for api %q", api.GetId())
}
