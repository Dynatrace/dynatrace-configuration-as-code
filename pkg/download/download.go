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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
)

var baseProjectName = "" //for folder naming
var cont = 0

//GetConfigs Entry point that retrieves the specified configurations from a Dynatrace environment
func GetConfigs(environments map[string]environment.Environment, path string, downloadSpecificAPI string) (statusCode int) {

	list, apiErrors := getAPIList(downloadSpecificAPI)
	if len(apiErrors) != 0 {
		util.Log.Error("The API list contains invalid values. Run monaco command to see the available options\n")
		for _, el := range apiErrors {
			util.Log.Error("\t " + el.Error())
		}
		statusCode = -1
		return statusCode
	}
	downloadErrors := make(map[string]error)

	for _, environment := range environments {

		//download configs for each environment
		err := downloadConfigFromEnvironment(environment, path, list)
		if err != nil {
			downloadErrors[environment.GetId()] = err
		}
	}
	statusCode = publishErrors(downloadErrors)
	return statusCode
}

func publishErrors(errors map[string]error) (statusCode int) {
	for environment, err := range errors {
		util.Log.Error("Download to %s failed with error %s\n", environment, err)
		statusCode = -1
	}
	if statusCode == -1 {
		return statusCode
	}
	return 0
}

//returns the list of API filter if the download specific flag is used, otherwise returns all the API's
func getAPIList(downloadSpecificAPI string) (filterAPIList map[string]api.Api, errorList []error) {
	availableApis := api.NewApis()
	blank := strings.TrimSpace(downloadSpecificAPI) == ""
	filterAPIList = make(map[string]api.Api)
	if blank {
		for _, entity := range availableApis {
			path := transFormSpecialCasesAPIPath(entity.GetId(), entity.GetApiPath())
			filterAPIList[entity.GetId()] = api.NewApi(entity.GetId(), path)
		}
		return filterAPIList, nil
	}
	requestedApis := strings.Split(downloadSpecificAPI, ",")
	result := true

	for _, id := range requestedApis {
		cleanAPI := strings.TrimSpace(id)
		isApi := api.IsApi(cleanAPI)
		if isApi == false {
			result = false
			err := fmt.Errorf("Value %s is not a valid API name ", cleanAPI)
			errorList = append(errorList, err)
		} else {
			filterAPI := availableApis[cleanAPI]
			path := transFormSpecialCasesAPIPath(filterAPI.GetId(), filterAPI.GetApiPath())
			filterAPIList[cleanAPI] = api.NewApi(filterAPI.GetId(), path)
		}
	}
	if result == false {
		return nil, errorList
	}
	return filterAPIList, nil
}

//function that deals with modifying the api path register in the api class to apply filters to skip read only entities from being downloaded
func transFormSpecialCasesAPIPath(apiID string, apiURL string) string {

	switch apiID {
	case "synthetic-location":
		return apiURL + "?type=PRIVATE"
	default:
		return apiURL
	}
}

//creates the project and dowload the configs
func downloadConfigFromEnvironment(environment environment.Environment, basepath string, listApis map[string]api.Api) (err error) {

	projectName := baseProjectName + environment.GetId()
	path := filepath.Join(basepath, projectName)
	creator := files.NewDiskFileCreator()

	util.Log.Info("Creating base project name %s", projectName)
	fullpath, err := creator.CreateFolder(path)
	if err != nil {
		return err
	}
	token, err := environment.GetToken()
	if err != nil {
		util.Log.Error("error retrieving token: %s %v", err)
		return err
	}
	client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), token)
	if err != nil {
		util.Log.Error("error creating dynatrace client: %s %v", err)
		return err
	}
	for _, api := range listApis {
		util.Log.Info(" --- GETTING CONFIGS for %s", api.GetId())
		jcreator := jsoncreator.NewJSONCreator()
		ycreator := yamlcreator.NewYamlConfig()
		err = createConfigsFromAPI(api, token, creator, fullpath, client, jcreator, ycreator)
		if err != nil {
			util.Log.Error("error configs for api: %s %v", api.GetId(), err)
		}
	}
	util.Log.Info("END downloading info %s", projectName)
	return err
}

func createConfigsFromAPI(api api.Api, token string, creator files.FileCreator, fullpath string, client rest.DynatraceClient,
	jcreator jsoncreator.JSONCreator, ycreator yamlcreator.YamlCreator) (err error) {
	//retrieves all objects for the specific api
	values, err := client.List(api)

	if len(values) == 0 {
		util.Log.Info("No elements for API %s", api.GetId())
		return nil
	}
	subPath := filepath.Join(fullpath, api.GetId())
	creator.CreateFolder(subPath)

	for _, val := range values {
		util.Log.Debug("getting detail %s", val)
		cont++
		util.Log.Debug("REQUEST counter %v", cont)
		name, filter, err := jcreator.CreateJSONConfig(client, api, val, creator, subPath)
		if err != nil {
			util.Log.Error("error creating config api json file: %v", err)
			continue
		}
		if filter == true {
			continue
		}
		ycreator.AddConfig(name, val.Name)
	}

	err = ycreator.CreateYamlFile(creator, subPath, api.GetId())
	if err != nil {
		util.Log.Error("error creating config api yaml file: %v", err)
		return err
	}
	return nil
}
