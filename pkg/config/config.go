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

package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
)

//go:generate mockgen -source=config.go -destination=config_mock.go -package=config Config

type Config interface {
	GetConfigForEnvironment(environment environment.Environment, dict map[string]api.DynatraceEntity) ([]byte, error)
	IsSkipDeployment(environment environment.Environment) bool
	GetApi() api.Api
	GetObjectNameForEnvironment(environment environment.Environment, dict map[string]api.DynatraceEntity) (string, error)
	HasDependencyOn(config Config) bool
	GetFilePath() string
	GetFullQualifiedId() string
	GetType() string
	GetMeIdsOfEnvironment(environment environment.Environment) map[string]map[string]string
	GetId() string
	GetProject() string
	GetProperties() map[string]map[string]string
	GetRequiredByConfigIdList() []string
	addToRequiredByConfigIdList(config string)
}

var dependencySuffixes = []string{".id", ".name"}

const skipConfigDeploymentParameter = "skipDeployment"

type configImpl struct {
	id                  string
	project             string
	properties          map[string]map[string]string
	template            util.Template
	api                 api.Api
	objectName          string
	fileName            string
	requiredByConfigIds []string
}

// configFactory is used to create new Configs - this is needed for testing purposes
type ConfigFactory interface {
	NewConfig(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.Api) (Config, error)
}

type configFactoryImpl struct{}

func NewConfigFactory() ConfigFactory {
	return &configFactoryImpl{}
}

func NewConfig(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.Api) (Config, error) {

	template, err := util.NewTemplate(fs, fileName)
	if err != nil {
		return nil, fmt.Errorf("loading config %s failed with %s", project+string(os.PathSeparator)+id, err)
	}

	return newConfig(id, project, template, filterProperties(id, properties), api, fileName), nil
}

func NewConfigForDelete(id string, fileName string, properties map[string]map[string]string, api api.Api) Config {
	return newConfig(id, "", nil, filterProperties(id, properties), api, fileName)
}

func newConfig(id string, project string, template util.Template, properties map[string]map[string]string, api api.Api, fileName string) Config {
	return &configImpl{
		id:         id,
		project:    project,
		template:   template,
		properties: properties,
		api:        api,
		fileName:   fileName,
	}
}

func filterProperties(id string, properties map[string]map[string]string) map[string]map[string]string {

	result := make(map[string]map[string]string)
	configNameInID := strings.Split(id, ".")[0]
	for key, value := range properties {
		configNameInKey := strings.Split(key, ".")[0]
		if strings.HasPrefix(key, id) && configNameInID == configNameInKey {
			result[key] = value
		}
	}

	return result
}

func (c *configImpl) IsSkipDeployment(environment environment.Environment) bool {
	environmentKey := c.id + "." + environment.GetId()

	if properties, ok := c.properties[environmentKey]; ok {
		if value, ok := properties[skipConfigDeploymentParameter]; ok {
			return strings.EqualFold(value, "true")
		}
	}

	environmentGroupKey := c.id + "." + environment.GetGroup()

	if properties, ok := c.properties[environmentGroupKey]; ok {
		if value, ok := properties[skipConfigDeploymentParameter]; ok {
			return strings.EqualFold(value, "true")
		}
	}

	if properties, ok := c.properties[c.id]; ok {
		if value, ok := properties[skipConfigDeploymentParameter]; ok {
			return strings.EqualFold(value, "true")
		}
	}

	return false
}

func (c *configImpl) GetConfigForEnvironment(environment environment.Environment, dict map[string]api.DynatraceEntity) ([]byte, error) {
	filtered := copyProperties(c.properties)

	if len(filtered) == 0 {
		json, err := c.template.ExecuteTemplate(map[string]string{})

		if err != nil {
			return nil, err
		}

		err = util.ValidateJson(json, c.GetFilePath())

		if err != nil {
			return nil, err
		}

		return []byte(json), nil
	}

	environmentGroupKey := c.id + "." + environment.GetGroup()
	environmentKey := c.id + "." + environment.GetId()

	// collect all group and environment properties
	// environment override group properties
	for _, environment := range []string{environmentGroupKey, environmentKey} {
		filteredForEnvironment := filterProperties(environment, c.properties)
		if len(filteredForEnvironment) > 0 {
			for key, value := range filteredForEnvironment[environment] {
				_, ok := filtered[c.id]
				if !ok {
					filtered[c.id] = make(map[string]string)
				}

				filtered[c.id][key] = value
			}
		}
	}

	filtered, err := c.replaceDependencies(filtered, dict)

	if err != nil {
		return nil, err
	}

	json, err := c.template.ExecuteTemplate(filtered[c.id])

	if err != nil {
		return nil, err
	}

	json = strings.ReplaceAll(json, "&#34;", "\"")

	err = util.ValidateJson(json, c.GetFilePath())

	if err != nil {
		return nil, err
	}

	return []byte(json), nil
}

func (c *configImpl) GetObjectNameForEnvironment(environment environment.Environment, dict map[string]api.DynatraceEntity) (string, error) {
	environmentKey := c.id + "." + environment.GetId()
	environmentGroupKey := c.id + "." + environment.GetGroup()
	name := c.properties[environmentKey]["name"]
	// assign group value if exists
	if name == "" {
		name = c.properties[environmentGroupKey]["name"]
	}
	// assign default value
	if name == "" {
		name = c.properties[c.id]["name"]
	}
	if name == "" {
		return "", fmt.Errorf("could not find name property in config %s, please make sure `name` is defined", c.GetFullQualifiedId())
	}
	if isDependency(name) {
		return c.parseDependency(name, dict)
	}
	return name, nil
}

func copyProperties(original map[string]map[string]string) map[string]map[string]string {

	copies := make(map[string]map[string]string)
	for k, v := range original {
		copies[k] = make(map[string]string)
		for key, val := range v {
			copies[k][key] = val
		}
	}
	return copies
}

func (c *configImpl) replaceDependencies(data map[string]map[string]string, dict map[string]api.DynatraceEntity) (map[string]map[string]string, error) {
	var err error
	for k, v := range data {
		for k2, v2 := range v {
			if isDependency(v2) {
				data[k][k2], err = c.parseDependency(v2, dict)
				if err != nil {
					return data, err
				}
			}
		}
	}

	return data, nil
}

func (c *configImpl) parseDependency(dependency string, dict map[string]api.DynatraceEntity) (string, error) {

	// in case of an absolute path within the dependency:
	if strings.HasPrefix(dependency, string(os.PathSeparator)) {
		// remove prefix "/"
		dependency = dependency[1:]
	}

	id, access, err := splitDependency(dependency)
	if err != nil {
		return "", err
	}
	dtObject, ok := dict[id]
	if !ok {
		return "", errors.New("Id '" + id + "' was not available. Please make sure the reference exists.")
	}

	switch access {
	case "id":
		return dtObject.Id, nil
	case "name":
		return dtObject.Name, nil
	default:
		return "", fmt.Errorf("accessor %s not found for dependcy id %s", access, id)
	}
}

func isDependency(property string) bool {
	for _, suffix := range dependencySuffixes {
		if strings.HasSuffix(property, suffix) {
			return true
		}
	}
	return false
}

func splitDependency(property string) (id string, access string, err error) {
	split := strings.Split(property, ".")
	if len(split) < 2 {
		return "", "", fmt.Errorf("property %s cannot be split", property)
	}
	firstPart, secondPart := split[0], split[1]

	if len(split) > 2 {
		util.Log.Debug("\t\t\tproperty %s contains more than the single expected `.` separator, using last separator for split", property)
		secondPart = split[len(split)-1]
		firstPart = strings.TrimSuffix(property, "."+secondPart)
	}
	return firstPart, secondPart, nil
}

func (c *configImpl) GetApi() api.Api {
	return c.api
}

func (c *configImpl) GetRequiredByConfigIdList() []string {
	return c.requiredByConfigIds
}

func (c *configImpl) addToRequiredByConfigIdList(config string) {
	c.requiredByConfigIds = append(c.requiredByConfigIds, config)
}

func (c *configImpl) GetType() string {
	return c.api.GetId()
}

func (c *configImpl) GetId() string {
	return c.id
}

func (c *configImpl) GetProject() string {
	return c.project
}

func (c *configImpl) GetProperties() map[string]map[string]string {
	return c.properties
}

// HasDependencyOn checks if one config depends on the given parameter config
// Having a dependency means, that the config having the dependency needs to be applied AFTER the config it depends on
func (c *configImpl) HasDependencyOn(config Config) bool {
	for _, v := range c.properties {
		for _, value := range v {
			valueIndex := strings.LastIndex(value, ".")

			// Check dependencies only for values ending with suffixes
			// User can freely define values using dots, but .name$ and .id$ are reserved
			if valueIndex != -1 && isDependency(value[valueIndex:]) {
				valueString := value[:valueIndex]

				if strings.HasPrefix(valueString, string(os.PathSeparator)) {
					// remove prefix "/"
					valueString = valueString[1:]
				}

				// if dependency is relative path:
				// projects, config type and location should match
				// e.g. - dep: management-zone/zone1.name
				// should match config.type and config.id
				if len(strings.Split(valueString, string(os.PathSeparator))) < 3 && c.GetProject() == config.GetProject() {
					if valueString == strings.Join([]string{config.GetType(), config.GetId()}, string(os.PathSeparator)) {
						config.addToRequiredByConfigIdList(c.GetFullQualifiedId())
						return true
					}
				}

				// generate configuration path of configuration to be checked for dependency
				pathPart := []string{config.GetProject(), config.GetType(), config.GetId()}
				configFullPath := strings.Join(pathPart, string(os.PathSeparator))

				// if dependency is full path, than check if it's matching the
				// configuration value path
				// e.g. dep: /project1/management-zone/test-zone.name
				// will match configuration path of /cluster/project1/management-zone/test-zone
				// If we have 2 projects with exact same subprojects then it could cause some
				// id collisions. It is therefore advisable, to always use full paths in
				// multi-project environments
				if strings.HasSuffix(configFullPath, valueString) {
					config.addToRequiredByConfigIdList(c.GetFullQualifiedId())
					return true
				}
			}
		}
	}
	return false
}

// GetFilePath returns the path (file name) of the config json
func (c *configImpl) GetFilePath() string {
	return c.fileName
}

// GetFullQualifiedId returns the full qualified id of the config based on project, api and config id
func (c *configImpl) GetFullQualifiedId() string {
	return strings.Join([]string{c.GetProject(), c.GetApi().GetId(), c.GetId()}, string(os.PathSeparator))
}

// NewConfig creates a new Config
func (c *configFactoryImpl) NewConfig(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.Api) (Config, error) {
	config, err := NewConfig(fs, id, project, fileName, properties, api)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetMeIdsOfEnvironment returns the config's properties filtered by ME identifiers
func (c *configImpl) GetMeIdsOfEnvironment(environment environment.Environment) map[string]map[string]string {

	result := make(map[string]map[string]string)

	for name, props := range c.properties {

		if !strings.HasSuffix(name, environment.GetId()) {
			continue
		}

		for key, value := range props {

			if isMeId(value) {
				innerMap, ok := result[name]
				if !ok {
					innerMap = make(map[string]string)
					result[name] = innerMap
				}
				innerMap[key] = value
			}
		}
	}
	return result
}

// isMeId checks if the given value looks like an ME identifier
func isMeId(value string) bool {

	regExp := regexp.MustCompile(`[A-Z_]+-[A-Z0-9]{16}`)
	return regExp.Match([]byte(value))
}
