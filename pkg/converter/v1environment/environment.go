/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package v1environment

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"strings"
)

type EnvironmentV1 struct {
	id             string
	name           string
	group          string
	environmentUrl string
	envTokenName   string
}

func newEnvironmentsV1(maps map[string]map[string]string) (map[string]*EnvironmentV1, []error) {

	environments := make(map[string]*EnvironmentV1)
	errors := make([]error, 0)

	for id, details := range maps {
		environment, err := newEnvironmentV1(id, details)
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

func newEnvironmentV1(id string, properties map[string]string) (*EnvironmentV1, error) {

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

	environmentName, err := errutils.CheckProperty(properties, "name")
	if err != nil {
		return nil, fmt.Errorf("failed to parse config for environment %s: %w", id, err)
	}
	environmentUrl, err := errutils.CheckProperty(properties, "env-url")
	if err != nil {
		return nil, fmt.Errorf("failed to parse config for environment %s: %w", id, err)
	}
	envTokenName, err := errutils.CheckProperty(properties, "env-token-name")
	if err != nil {
		return nil, fmt.Errorf("failed to parse config for environment %s: %w", id, err)
	}

	return NewEnvironmentV1(id, environmentName, environmentGroup, environmentUrl, envTokenName), nil
}

func NewEnvironmentV1(id string, name string, group string, environmentUrl string, envTokenName string) *EnvironmentV1 {
	environmentUrl = strings.TrimSuffix(environmentUrl, "/")

	return &EnvironmentV1{
		id:             id,
		name:           name,
		group:          group,
		environmentUrl: environmentUrl,
		envTokenName:   envTokenName,
	}
}

func (s *EnvironmentV1) GetId() string {
	return s.id
}

func (s *EnvironmentV1) GetEnvironmentUrl() string {
	return s.environmentUrl
}

func (s *EnvironmentV1) GetTokenName() string {
	return s.envTokenName
}

func (s *EnvironmentV1) GetGroup() string {
	return s.group
}
