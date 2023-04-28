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

package deploy

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	configError "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/errors"
)

type ProjectErrors map[string]ApiErrors
type ApiErrors map[string]ConfigErrors
type ConfigErrors map[string][]configError.ConfigError

type GroupErrors map[string]EnvironmentErrors
type EnvironmentErrors map[string][]configError.DetailedConfigError

func printErrorReport(deploymentErrors []error) { // nolint:gocognit
	var configErrors []configError.ConfigError
	var generalErrors []error

	for _, err := range deploymentErrors {
		var configErr configError.ConfigError
		if errors.As(err, &configErr) {
			configErrors = append(configErrors, configErr)
		} else {
			generalErrors = append(generalErrors, err)
		}
	}

	if len(generalErrors) > 0 {
		log.Error("=== General Errors ===")
		for _, err := range generalErrors {
			log.Error(errutils.ErrorString(err))
		}
	}

	groupedConfigErrors := groupConfigErrors(configErrors)

	for project, apiErrors := range groupedConfigErrors {
		for api, configErrors := range apiErrors {
			for config, errs := range configErrors {
				var generalConfigErrors []configError.ConfigError
				var detailedConfigErrors []configError.DetailedConfigError

				for _, err := range errs {
					switch e := err.(type) {
					case configError.DetailedConfigError:
						detailedConfigErrors = append(detailedConfigErrors, e)
					default:
						generalConfigErrors = append(generalConfigErrors, e)
					}
				}

				groupErrors := groupEnvironmentConfigErrors(detailedConfigErrors)

				for _, err := range generalConfigErrors {
					log.Error("%s:%s:%s %s", project, api, config, errutils.ErrorString(err))
				}

				for group, environmentErrors := range groupErrors {
					for env, errs := range environmentErrors {
						for _, err := range errs {
							log.Error("%s(%s) %s:%s:%s %T %s", env, group, project, api, config, err, errutils.ErrorString(err))
						}
					}
				}
			}
		}
	}
}

func groupEnvironmentConfigErrors(errors []configError.DetailedConfigError) GroupErrors {
	groupErrors := make(GroupErrors)

	for _, err := range errors {
		locationDetails := err.LocationDetails()

		envErrors := groupErrors[locationDetails.Group]

		if envErrors == nil {
			envErrors = make(EnvironmentErrors)
			groupErrors[locationDetails.Group] = envErrors
		}

		envErrors[locationDetails.Environment] = append(envErrors[locationDetails.Environment], err)
	}

	return groupErrors
}

func groupConfigErrors(errors []configError.ConfigError) ProjectErrors {
	projectErrors := make(ProjectErrors)

	for _, err := range errors {
		coord := err.Coordinates()

		typeErrors := projectErrors[coord.Project]

		if typeErrors == nil {
			typeErrors = make(ApiErrors)
			typeErrors[coord.Type] = make(ConfigErrors)
			projectErrors[coord.Project] = typeErrors
		}

		configErrors := typeErrors[coord.Type]

		if configErrors == nil {
			configErrors = make(ConfigErrors)
			typeErrors[coord.Type] = configErrors
		}

		configErrors[coord.ConfigId] = append(configErrors[coord.ConfigId], err)
	}

	return projectErrors
}
