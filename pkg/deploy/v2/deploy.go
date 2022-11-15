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

package v2

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

type InvalidJsonError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	error              error
}

func (e InvalidJsonError) Unwrap() error {
	return e.error
}

var (
	// InvalidJsonError must support unwrap function
	_ (interface{ Unwrap() error }) = (*InvalidJsonError)(nil)
)

func (e InvalidJsonError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e InvalidJsonError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e InvalidJsonError) Error() string {
	return e.error.Error()
}

type ConfigDeployError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	Reason             string
}

func newConfigDeployError(conf *config.Config, reason string) ConfigDeployError {
	return ConfigDeployError{
		Config: conf.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       conf.Group,
			Environment: conf.Environment,
		},
		Reason: reason,
	}
}

func (e ConfigDeployError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e ConfigDeployError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e ConfigDeployError) Error() string {
	return e.Reason
}

type ParameterReferenceError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	Parameter          string
	Reference          parameter.ParameterReference
	Reason             string
}

func newParameterReferenceError(coord coordinate.Coordinate, group string, env string,
	param string, ref parameter.ParameterReference, reason string) ParameterReferenceError {
	return ParameterReferenceError{
		Config: coord,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       group,
			Environment: env,
		},
		Parameter: param,
		Reference: ref,
		Reason:    reason,
	}
}

func (e ParameterReferenceError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e ParameterReferenceError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e ParameterReferenceError) Error() string {
	return fmt.Sprintf("parameter `%s` cannot reference `%s`: %s",
		e.Parameter, e.Reference, e.Reason)
}

var (
	_ configErrors.DetailedConfigError = (*ConfigDeployError)(nil)
	_ configErrors.DetailedConfigError = (*ParameterReferenceError)(nil)
)

type knownEntityMap map[string]map[string]struct{}

// DeployConfigs deploys the given configs with the given apis via the given client
// NOTE: the given configs need to be sorted, otherwise deployment will
// probably fail, as references cannot be resolved
func DeployConfigs(client rest.DynatraceClient, apis map[string]api.Api,
	sortedConfigs []config.Config, continueOnError, dryRun bool) []error {

	entities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)
	var errors []error

	knownEntityNames := createKnownEntityMap(apis)

	for i, config := range sortedConfigs {
		coord := config.Coordinate

		if config.Skip {
			entities[coord] = parameter.ResolvedEntity{
				EntityName: coord.Config,
				Coordinate: coord,
				Properties: parameter.Properties{},
				Skip:       true,
			}

			// if the config is skip we do not care if the same name
			// has already been used

			continue
		}

		apiToDeploy := apis[coord.Type]
		if apiToDeploy == nil {
			errors = append(errors, fmt.Errorf("unknown api `%s`. this is most likely a bug", coord.Type))

			if continueOnError || dryRun {
				continue
			} else {
				return errors
			}
		}

		entity, deploymentErrors := deployConfig(client, apiToDeploy, entities, knownEntityNames, &sortedConfigs[i], dryRun)

		if deploymentErrors != nil {
			errors = append(errors, deploymentErrors...)

			if continueOnError || dryRun {
				continue
			} else {
				return errors
			}
		}

		knownEntityNames[config.Coordinate.Type][entity.EntityName] = struct{}{}
		entities[entity.Coordinate] = entity
	}

	return errors
}

func createKnownEntityMap(apis map[string]api.Api) knownEntityMap {
	var result = make(knownEntityMap)

	for _, api := range apis {
		result[api.GetId()] = make(map[string]struct{})
	}

	return result

}

func deployConfig(
	client rest.DynatraceClient,
	theApi api.Api,
	entities parameter.ResolvedEntities,
	knownEntityNames knownEntityMap,
	conf *config.Config,
	dryRun bool,
) (parameter.ResolvedEntity, []error) {

	var errors []error

	parameters, sortErrs := topologysort.SortParameters(conf.Group, conf.Environment, conf.Coordinate, conf.Parameters)
	errors = append(errors, sortErrs...)

	properties, errs := ResolveParameterValues(client, conf, entities, parameters, dryRun)

	errors = append(errors, errs...)

	configName, err := ExtractConfigName(conf, properties)

	if err != nil {
		errors = append(errors, err)
	} else {
		if _, found := knownEntityNames[theApi.GetId()][configName]; found && !theApi.IsNonUniqueNameApi() {
			errors = append(errors, newConfigDeployError(conf, fmt.Sprintf("duplicated config name `%s`", configName)))
		}
	}

	if errors != nil {
		return parameter.ResolvedEntity{}, errors
	}

	renderedConfig, err := template.Render(conf.Template, properties)

	if err != nil {
		return parameter.ResolvedEntity{}, []error{err}
	}

	err = util.ValidateJson(renderedConfig, util.Location{
		Coordinate:       conf.Coordinate,
		Group:            conf.Group,
		Environment:      conf.Environment,
		TemplateFilePath: conf.Template.Name(),
	})

	if err != nil {
		return parameter.ResolvedEntity{}, []error{&InvalidJsonError{
			Config: conf.Coordinate,
			EnvironmentDetails: configErrors.EnvironmentDetails{
				Group:       conf.Group,
				Environment: conf.Environment,
			},
			error: err,
		}}
	}

	if theApi.IsDeprecatedApi() {
		log.Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", theApi.GetId(), theApi.IsDeprecatedBy())
	}

	var entity api.DynatraceEntity
	if theApi.IsNonUniqueNameApi() {
		configId := conf.Coordinate.Config
		projectId := conf.Coordinate.Project

		entityUuid := configId

		isUuidOrMeId := util.IsUuid(entityUuid) || util.IsMeId(entityUuid)
		if !isUuidOrMeId {
			entityUuid, err = util.GenerateUuidFromConfigId(projectId, configId)
			if err != nil {
				return parameter.ResolvedEntity{}, []error{newConfigDeployError(conf, err.Error())}
			}
		}

		entity, err = client.UpsertByEntityId(theApi, entityUuid, configName, []byte(renderedConfig))
	} else {
		entity, err = client.UpsertByName(theApi, configName, []byte(renderedConfig))
	}

	if err != nil {
		return parameter.ResolvedEntity{}, []error{newConfigDeployError(conf, err.Error())}
	}

	properties[config.IdParameter] = entity.Id
	properties[config.NameParameter] = entity.Name

	return parameter.ResolvedEntity{
		EntityName: entity.Name,
		Coordinate: conf.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil
}

func ExtractConfigName(conf *config.Config, properties parameter.Properties) (string, error) {
	val, found := properties[config.NameParameter]

	if !found {
		return "", newConfigDeployError(conf, "missing `name` for config")
	}

	name, success := val.(string)

	if !success {
		return "", newConfigDeployError(conf, "`name` in config is not of type string")
	}

	return name, nil
}

func ResolveParameterValues(client rest.DynatraceClient, conf *config.Config,
	entities map[coordinate.Coordinate]parameter.ResolvedEntity, parameters []topologysort.ParameterWithName,
	dryRun bool) (parameter.Properties, []error) {

	var errors []error

	properties := make(parameter.Properties)

	for _, container := range parameters {
		name := container.Name
		param := container.Parameter

		errs := validateParameterReferences(conf.Coordinate, conf.Group, conf.Environment, entities, name, param)

		if errs != nil {
			errors = append(errors, errs...)
			continue
		}

		val, err := param.ResolveValue(parameter.ResolveContext{
			ResolvedEntities:        entities,
			ConfigCoordinate:        conf.Coordinate,
			Group:                   conf.Group,
			Environment:             conf.Environment,
			ParameterName:           name,
			ResolvedParameterValues: properties,
			Client:                  client,
			DryRun:                  dryRun,
		})

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if name == config.NameParameter {
			properties[name] = util.ToString(val)
		} else {
			properties[name] = val
		}
	}

	if len(errors) > 0 {
		// we want to return the partially resolved properties here, to find
		// more errors in the outer logic
		return properties, errors
	}

	return properties, nil
}

func validateParameterReferences(configCoordinates coordinate.Coordinate,
	group string, environment string,
	entities map[coordinate.Coordinate]parameter.ResolvedEntity,
	paramName string, param parameter.Parameter) (errors []error) {

	for _, ref := range param.GetReferences() {
		// we have to ignore references to the same config,
		// as they will never be resolved before we validate
		// the parameters
		if ref.Config == configCoordinates {
			// parameters referencing themselves makes no sense
			if ref.Property == paramName {
				errors = append(errors, newParameterReferenceError(configCoordinates, group, environment, paramName, ref, "parameter referencing itself"))
			}

			continue
		}

		entity, found := entities[ref.Config]

		if !found {
			errors = append(errors, newParameterReferenceError(configCoordinates, group, environment, paramName, ref, "referenced config not found"))
			continue
		}

		if entity.Skip {
			errors = append(errors, newParameterReferenceError(configCoordinates, group, environment, paramName, ref, "referencing skipped config"))
			continue
		}
	}

	return errors
}
