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

package yamlcreator

import (
	"io/fs"
	"path/filepath"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

//go:generate mockgen -source=yaml_creator.go -destination=yaml_creator_mock.go -package=yamlcreator YamlCreator

//YamlCreator implements method to create the yaml configuration file
type YamlCreator interface {
	ReadYamlFile(fs afero.Fs, configSubPath string, apiId string) error
	WriteYamlFile(fs afero.Fs, path string, name string) error
	AddConfig(name string, rawName string)
	UpdateConfig(entityId string, entityName string, jsonFileName string)
}

// YamlConfig defines the structure for the config file for each API including meta data
type YamlConfig struct {
	Config                   []map[string]string
	Detail                   map[string][]DetailConfig
	EnvironmentName          string
	generateUuidFromConfigId func(projectUniqueId string, configId string) (string, error)
	marshalYaml              func(in interface{}) (out []byte, err error)
	unmarshalYaml            func(text string, fileName string) (error, map[string]map[string]string)
	readFile                 func(fs afero.Fs, filename string) ([]byte, error)
	writeFile                func(fs afero.Fs, filename string, data []byte, perm fs.FileMode) error
}

// DetailConfig contains name and meta data for each API and entity
type DetailConfig struct {
	Name           string `yaml:"name"`
	Id             string `yaml:"id"`
	ConfigFileName string `yaml:"configFileName"`
	IsDownloaded   bool   `yaml:"isDownloaded"`
}

// CleansedYamlConfig defines the structure for the config file for each API
type CleansedYamlConfig struct {
	Config []map[string]string               `yaml:"config"`
	Detail map[string][]CleansedDetailConfig `yaml:",inline"`
}

// CleansedDetailConfig sets the default properties to be set replace in each json file
type CleansedDetailConfig struct {
	Name string `yaml:"name"`
}

//NewYamlConfig return a new yaml struct with Config and Detail as fields
func NewYamlConfig(environmentName string) *YamlConfig {
	yamlConfig := YamlConfig{
		EnvironmentName:          environmentName,
		generateUuidFromConfigId: util.GenerateUuidFromConfigId,
		marshalYaml:              yaml.Marshal,
		unmarshalYaml:            util.UnmarshalYaml,
		readFile:                 afero.ReadFile,
		writeFile:                afero.WriteFile,
	}
	yamlConfig.Detail = make(map[string][]DetailConfig)
	return &yamlConfig
}

//isTopLevelConfigurationYamlKey checks if a given yaml key is the toplevel element defining further Configs
func isTopLevelConfigurationYamlKey(key string) bool {
	return key == "config"
}

//AddConfig allows to add new configs to the yaml file
func (yc *YamlConfig) AddConfig(name string, rawName string) {

	config := DetailConfig{Name: rawName}
	mp := make(map[string]string)
	mp[name] = name + ".json"
	yc.Config = append(yc.Config, mp)
	yc.Detail[name] = append(yc.Detail[name], config)
}

// UpdateConfig allows updating configs in the yaml file
func (yc *YamlConfig) UpdateConfig(entityId string, entityName string, jsonFileName string) {
	configId := yc.getConfigName(entityId)

	detailConfig := DetailConfig{
		Name:           entityName,
		Id:             entityId,
		ConfigFileName: jsonFileName,
		IsDownloaded:   true,
	}

	yc.Detail[configId] = []DetailConfig{detailConfig}

	for i, config := range yc.Config {
		for k := range config {
			if k == configId {
				// If config is found, return
				yc.Config[i][k] = jsonFileName
				return
			}
		}
	}

	// If no config was found, create new one
	config := map[string]string{}
	config[configId] = jsonFileName
	yc.Config = append(yc.Config, config)
}

func (yc *YamlConfig) getConfigName(entityId string) string {
	for _, v := range yc.Detail {
		config := v[0]
		if config.Id == entityId {
			return config.Name
		}
	}

	return entityId
}

func (yc *YamlConfig) cleanseYamlConfig() CleansedYamlConfig {
	yamlConfig := CleansedYamlConfig{
		Config: []map[string]string{},
		Detail: map[string][]CleansedDetailConfig{},
	}

	for k, v := range yc.Detail {
		// isDownloaded specifies whether config was downloaded, i.e. currently exists in environment
		// If it wasn't downloaded, it also won't be added to the config yaml.
		isDownloaded := v[0].IsDownloaded

		if isDownloaded {
			entityName := v[0].Name
			configFileName := v[0].ConfigFileName

			yamlConfig.Detail[k] = []CleansedDetailConfig{}

			cleansedDetailConfig := CleansedDetailConfig{
				Name: entityName,
			}

			yamlConfig.Detail[k] = append(yamlConfig.Detail[k], cleansedDetailConfig)

			cleansedConfig := map[string]string{}
			cleansedConfig[k] = configFileName

			yamlConfig.Config = append(yamlConfig.Config, cleansedConfig)
		}
	}

	return yamlConfig
}

//WriteYamlFile transforms the struct into a physical file on disk
func (yc *YamlConfig) WriteYamlFile(fs afero.Fs, path string, name string) error {
	yamlConfig := yc.cleanseYamlConfig()
	fullPath := filepath.Join(path, name+".yaml")

	data, err := yc.marshalYaml(yamlConfig)
	if err != nil {
		util.Log.Error("error parsing yaml file: %v", err)
		return err
	}

	err = yc.writeFile(fs, fullPath, data, 0664)
	if err != nil {
		util.Log.Error("error creating yaml file %s", name)
		return err
	}

	return nil
}

func (yc *YamlConfig) parseConfigs(unmarshaledData map[string]map[string]string) {
	for k, v := range unmarshaledData {
		if isTopLevelConfigurationYamlKey(k) {
			for configId, configJson := range v {
				configItem := make(map[string]string)
				configItem[configId] = configJson

				yc.Config = append(yc.Config, configItem)
			}
		}
	}
}

func (yc *YamlConfig) findConfigFileName(configId string) string {
	for _, v := range yc.Config {
		for potentialId, configFileName := range v {
			if potentialId == configId {
				return configFileName
			}
		}
	}

	return ""
}

func (yc *YamlConfig) parseConfigDetails(unmarshaledData map[string]map[string]string) error {
	for configId := range unmarshaledData {
		if !isTopLevelConfigurationYamlKey(configId) {
			// As of May 2022, download does not support project structure
			// Fallback to environment unique id
			environmentUniqueConfigId := yc.EnvironmentName

			entityUuid, err := yc.generateUuidFromConfigId(environmentUniqueConfigId, configId)
			if err != nil {
				return err
			}

			configFileName := yc.findConfigFileName(configId)

			detailConfig := DetailConfig{
				Name:           configId,
				Id:             entityUuid,
				ConfigFileName: configFileName,
			}

			yc.Detail[configId] = []DetailConfig{
				detailConfig,
			}
		}
	}

	return nil
}

// ReadYamlFile reads an potentially existing config yaml
func (yc *YamlConfig) ReadYamlFile(fs afero.Fs, configSubPath string, apiId string) error {
	configFileName := apiId + ".yaml"
	fullPath := filepath.Join(configSubPath, configFileName)

	fileData, err := yc.readFile(fs, fullPath)
	if err != nil {
		return err
	}

	err, unmarshaledData := yc.unmarshalYaml(string(fileData), configFileName)
	if err != nil {
		return err
	}

	yc.parseConfigs(unmarshaledData)

	err = yc.parseConfigDetails(unmarshaledData)
	if err != nil {
		return err
	}

	return nil
}
