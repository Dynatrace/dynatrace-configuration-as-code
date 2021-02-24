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

// Environment object structure
type Environment interface {
	GetId() string
	GetURL() string
	GetToken() (string, error)
	GetGroup() string
	IsCluster() bool
}

type environmentImpl struct {
	id      string
	name    string
	group   string
	URL     string
	token   string
	envType string
}

// NewEnvironments creates a map of environment objects. Key is environment ID, value is Environment object
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
	url, urlErr := util.CheckProperty(properties, "url")
	token, tokenErr := util.CheckProperty(properties, "token")
	envType, envTypeErr := util.CheckProperty(properties, "type")

	// deprecated
	envUrl, envUrlErr := util.CheckProperty(properties, "env-url")
	envToken, envTokenErr := util.CheckProperty(properties, "env-token-name")

	if envUrlErr == nil && len(envUrl) > 0 {
		util.Log.Warn("You are using 'env-url' property in the environment file. This property is going to be deprecated in v2.0.0. Replace with a property 'url', instead.")
		url = envUrl
		urlErr = nil
	}

	if envTokenErr == nil && len(envToken) > 0 {
		util.Log.Warn("You are using 'env-token-name' property in the environment file. This property is going to be deprecated in v2.0.0. Replace with a property 'token', instead.")
		token = envToken
		tokenErr = nil
	}

	if nameErr != nil || urlErr != nil || tokenErr != nil || envTypeErr != nil {
		return nil, fmt.Errorf("failed to parse config for environment %s (issues: %s %s %s %s)", id, nameErr, urlErr, tokenErr, envTypeErr)
	}

	return NewEnvironment(id, environmentName, environmentGroup, url, token, envType), nil
}

// NewEnvironment creates a new environment object
func NewEnvironment(id string, name string, group string, url string, token string, envType string) Environment {
	url = strings.TrimSuffix(url, "/")

	return &environmentImpl{
		id:      id,
		name:    name,
		group:   group,
		URL:     url,
		token:   token,
		envType: envType,
	}
}

func (s *environmentImpl) GetId() string {
	return s.id
}

func (s *environmentImpl) GetURL() string {
	return s.URL
}

func (s *environmentImpl) GetToken() (string, error) {
	value := os.Getenv(s.token)
	if value == "" {
		return value, fmt.Errorf("environment variable " + s.token + " not found")
	}
	return value, nil
}

func (s *environmentImpl) GetGroup() string {
	return s.group
}

func (s *environmentImpl) IsCluster() bool {
	return s.envType == "cluster"
}
