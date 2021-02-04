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

package environment

import (
	"fmt"
	"os"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

type Environment interface {
	GetId() string
	GetEnvironmentUrl() string
	GetToken() (string, error)
	GetGroup() string
}

type environmentImpl struct {
	id             string
	name           string
	group          string
	environmentUrl string
	envTokenName   string
}

func NewEnvironments(maps map[string]map[string]string) (map[string]Environment, []error) {

	environments := make(map[string]Environment)
	errors := make([]error, 0)

	for id, details := range maps {
		environment, err := newEnvironment(id, details)
		if err != nil {
			errors = append(errors, err)
		} else {
			// create error instead of overwriting environments with same IDs
			if _, environmentAlreadyExists := environments[environment.GetId()]; !environmentAlreadyExists {
				environments[environment.GetId()] = environment
			} else {
				errors = append(errors, fmt.Errorf("environment `%s` is already defined, please use unique environment names", environment.GetId()))
			}
		}
	}

	return environments, errors
}

func newEnvironment(id string, properties map[string]string) (Environment, error) {

	// only one group per environment is allowed
	// ignore environments with leading or trailing `.`
	if strings.Count(id, ".") > 1 || strings.HasPrefix(id, ".") || strings.HasSuffix(id, ".") {
		return nil, fmt.Errorf("failed to parse group for environment %s", id)
	}

	environmentGroup := ""
	// does environment contain any groups
	if strings.Count(id, ".") == 1 {
		index := strings.Index(id, ".")
		environmentGroup = id[:index]
		id = id[index+1:]
	}

	// ignore environments where group matches environment name
	if id == environmentGroup {
		return nil, fmt.Errorf("group name must differ from environment name %s", id)
	}

	environmentName, nameErr := util.CheckProperty(properties, "name")
	environmentUrl, urlErr := util.CheckProperty(properties, "env-url")
	envTokenName, tokenErr := util.CheckProperty(properties, "env-token-name")

	if nameErr != nil || urlErr != nil || tokenErr != nil {
		return nil, fmt.Errorf("failed to parse config for environment %s (issues: %s %s %s)", id, nameErr, urlErr, tokenErr)
	}

	return NewEnvironment(id, environmentName, environmentGroup, environmentUrl, envTokenName), nil
}

func NewEnvironment(id string, name string, group string, environmentUrl string, envTokenName string) Environment {
	environmentUrl = strings.TrimSuffix(environmentUrl, "/")

	return &environmentImpl{
		id:             id,
		name:           name,
		group:          group,
		environmentUrl: environmentUrl,
		envTokenName:   envTokenName,
	}
}

func (s *environmentImpl) GetId() string {
	return s.id
}

func (s *environmentImpl) GetEnvironmentUrl() string {
	return s.environmentUrl
}

func (s *environmentImpl) GetToken() (string, error) {
	value := os.Getenv(s.envTokenName)
	if value == "" {
		return value, fmt.Errorf("environment variable " + s.envTokenName + " not found")
	}
	return value, nil
}

func (s *environmentImpl) GetGroup() string {
	return s.group
}
