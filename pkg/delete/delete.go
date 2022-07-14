/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package delete

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

const deleteDelimiter = "/"
const deleteFileName = "delete.yaml"

type deleteYaml struct {
	Delete interface{}
}

// LoadConfigsToDelete loads the delete.yaml file (if available) and converts its entries into configs which need
// to be deleted
func LoadConfigsToDelete(fs afero.Fs, apis map[string]api.Api, path string) (configs []config.Config, err error) {

	result := make([]config.Config, 0)

	deleteFilePath := filepath.Join(path, deleteFileName)
	data, err := afero.ReadFile(fs, deleteFilePath)
	if err != nil {
		// Don't raise an error. The delete.yaml might not be there, that's a valid case
		util.Log.Info("There is no delete file %s found in %s. Skipping delete config.", deleteFileName, deleteFilePath)
		return result, nil
	}

	list, err := unmarshalDeleteYaml(string(data), deleteFileName)
	if util.CheckError(err, deleteFileName+" file content was invalid") {
		return configs, err
	}

	for _, element := range list {

		configType, name, err := splitConfigToDelete(element)
		if util.CheckError(err, "deletion failed") {
			return configs, err
		}

		apiIface, validConfig := apis[configType]
		if !validConfig {
			return configs, errors.New("config type " + configType + " was not valid")
		}

		isNonUniqueNameApi := apiIface.IsNonUniqueNameApi()
		if isNonUniqueNameApi {
			util.Log.Warn("Detected non-unique naming API - can not safely delete %s. Please delete the correct configuration from your environment manually and remove from delete.yaml", element)
			continue
		}

		properties := make(map[string]map[string]string)
		properties[name] = make(map[string]string)
		properties[name]["name"] = name

		configForDeletion := config.NewConfigForDelete(name, "delete.yaml", properties, apiIface)
		result = append(result, configForDeletion)
	}

	return result, nil
}

// splitConfigToDelete gets one line of the delete.yaml as input and splits it into config type and name
// E.g.: dashboards/my-dashboard -> config type: dashboard, name: my-dashboard
func splitConfigToDelete(config string) (configType string, name string, err error) {

	if !strings.Contains(config, "/") {
		err = errors.New("config " + config + " does not contain '" + deleteDelimiter + "' delimiter")
		return
	}

	split := strings.Split(config, "/")
	if len(split) != 2 {
		err = errors.New("config " + config + " contains more than one '" + deleteDelimiter + "' delimiter")
		return
	}

	return split[0], split[1], nil
}

// unmarshalDeleteYaml takes the contents of a yaml file and converts it to a string array
// The yaml file should have the following format:
//
// delete:
//  - "list-entry-1"
//  - "list-entry-2"
//
func unmarshalDeleteYaml(text string, fileName string) (typed []string, err error) {

	d := deleteYaml{}

	err = yaml.Unmarshal([]byte(text), &d)
	if util.CheckError(err, "Failed to unmarshal yaml\n"+text+"for file name"+fileName+"\nerror:") {
		return typed, err
	}

	typed, err = convertList(d)
	if util.CheckError(err, "Failed to unmarshal yaml\n"+text+"for file name"+fileName+"\nerror:") {
		return typed, err
	}

	return typed, nil
}

// convertList converts the unmarshalled yaml into a string array
func convertList(original deleteYaml) ([]string, error) {

	if original.Delete == nil {
		return nil, errors.New("invalid YAML structure")
	}

	result := make([]string, 0)
	list, _ := original.Delete.([]interface{})

	for _, x := range list {
		y := x.(string)
		result = append(result, y)
	}

	return result, nil
}
