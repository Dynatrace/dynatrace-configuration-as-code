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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/log"
	"github.com/spf13/afero"
)

//go:generate mockgen -source=config.go -destination=config_mock.go -package=config Config

type Config interface {
	GetApi() api.Api
	HasDependencyOn(config Config) bool
	GetFilePath() string
	GetFullQualifiedId() string
	GetType() string
	GetId() string
	GetProject() string
	GetProperties() map[string]map[string]string
}

var dependencySuffixes = []string{".id", ".name"}

const SkipConfigDeploymentParameter = "skipDeployment"

type configImpl struct {
	id         string
	project    string
	properties map[string]map[string]string
	template   util.Template
	api        api.Api
	fileName   string
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
		return nil, fmt.Errorf("loading config %s failed with %w", project+string(os.PathSeparator)+id, err)
	}

	return newConfig(id, project, template, filterProperties(id, properties), api, fileName), nil
}

func NewConfigWithTemplate(id string, project string, fileName string, template util.Template,
	properties map[string]map[string]string, api api.Api) (Config, error) {
	return newConfig(id, project, template, filterProperties(id, properties), api, fileName), nil
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

func IsDependency(property string) bool {
	for _, suffix := range dependencySuffixes {
		if strings.HasSuffix(property, suffix) {
			return true
		}
	}
	return false
}

func SplitDependency(property string) (id string, access string, err error) {
	split := strings.Split(property, ".")
	if len(split) < 2 {
		return "", "", fmt.Errorf("property %s cannot be split", property)
	}
	firstPart, secondPart := split[0], split[1]

	if len(split) > 2 {
		log.Debug("property %s contains more than the single expected `.` separator, using last separator for split", property)
		secondPart = split[len(split)-1]
		firstPart = strings.TrimSuffix(property, "."+secondPart)
	}
	return filepath.ToSlash(firstPart), secondPart, nil
}

func (c *configImpl) GetApi() api.Api {
	return c.api
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
			if valueIndex != -1 && IsDependency(value[valueIndex:]) {
				valueString := value[:valueIndex]
				valueString = strings.TrimPrefix(valueString, string(os.PathSeparator))

				// if dependency is relative path:
				// projects, config type and location should match
				// e.g. - dep: management-zone/zone1.name
				// should match config.type and config.id
				if len(strings.Split(valueString, string(os.PathSeparator))) < 3 && c.GetProject() == config.GetProject() {
					if valueString == strings.Join([]string{config.GetType(), config.GetId()}, string(os.PathSeparator)) {
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
