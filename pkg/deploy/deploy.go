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

package deploy

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

type invalidJsonError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	error              error
}

func (e invalidJsonError) Unwrap() error {
	return e.error
}

var (
	// invalidJsonError must support unwrap function
	_ interface{ Unwrap() error } = (*invalidJsonError)(nil)
)

func (e invalidJsonError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e invalidJsonError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e invalidJsonError) Error() string {
	return e.error.Error()
}

type configDeployError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	Reason             string
}

func newConfigDeployError(conf *config.Config, reason string) configDeployError {
	return configDeployError{
		Config: conf.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       conf.Group,
			Environment: conf.Environment,
		},
		Reason: reason,
	}
}

func (e configDeployError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e configDeployError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e configDeployError) Error() string {
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
	_ configErrors.DetailedConfigError = (*configDeployError)(nil)
	_ configErrors.DetailedConfigError = (*ParameterReferenceError)(nil)
)

type knownEntityMap map[string]map[string]struct{}

// DeployConfigs deploys the given configs with the given apis via the given client
// NOTE: the given configs need to be sorted, otherwise deployment will
// probably fail, as references cannot be resolved
func DeployConfigs(client rest.DynatraceClient, apis api.ApiMap,
	sortedConfigs []config.Config, continueOnError, dryRun bool) []error {

	resolvedEntities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)
	knownEntityNames := createKnownEntityMap(apis)
	var errors []error

	for _, c := range sortedConfigs {
		c := c // to avoid implicit memory aliasing (gosec G601)

		if c.Skip {
			resolvedEntities[c.Coordinate] = parameter.ResolvedEntity{ //TODO where are entities used? why is this needed
				EntityName: c.Coordinate.ConfigId,
				Coordinate: c.Coordinate,
				Properties: parameter.Properties{},
				Skip:       true,
			}
			// if the config is skip we do not care if the same name
			// has already been used
			continue
		}

		var entity parameter.ResolvedEntity
		var deploymentErrors []error

		if c.Type.IsSettings() {
			entity, deploymentErrors = deploySetting(client, resolvedEntities, &c)
		} else {
			entity, deploymentErrors = deployConfig(client, apis, resolvedEntities, knownEntityNames, &c)
			if len(deploymentErrors) == 0 && entity.EntityName != "" {
				//known entity names only stored for Config APIs - if no error happened
				knownEntityNames[c.Coordinate.Type][entity.EntityName] = struct{}{}
			}
		}

		if deploymentErrors != nil {
			errors = append(errors, deploymentErrors...)

			if continueOnError || dryRun {
				continue
			} else {
				return errors
			}
		}

		resolvedEntities[entity.Coordinate] = entity
	}

	return errors
}

func createKnownEntityMap(apis map[string]api.Api) knownEntityMap {
	var result = make(knownEntityMap)

	for _, a := range apis {
		result[a.GetId()] = make(map[string]struct{})
	}

	return result

}

func deployConfig(client rest.ConfigClient, apis api.ApiMap, entities parameter.ResolvedEntities, knownEntityNames knownEntityMap, conf *config.Config) (parameter.ResolvedEntity, []error) {

	apiToDeploy := apis[conf.Coordinate.Type]
	if apiToDeploy == nil {
		return parameter.ResolvedEntity{}, []error{fmt.Errorf("unknown api `%s`. this is most likely a bug", conf.Type.Api)}
	}

	properties, errors := resolveProperties(conf, entities)
	if len(errors) > 0 {
		return parameter.ResolvedEntity{}, errors
	}

	configName, err := ExtractConfigName(conf, properties)
	if err != nil {
		errors = append(errors, err)
	} else {
		if _, found := knownEntityNames[apiToDeploy.GetId()][configName]; found && !apiToDeploy.IsNonUniqueNameApi() {
			errors = append(errors, newConfigDeployError(conf, fmt.Sprintf("duplicated config name `%s`", configName)))
		}
	}
	if len(errors) > 0 {
		return parameter.ResolvedEntity{}, errors
	}

	renderedConfig, err := renderConfig(conf, properties)
	if err != nil {
		return parameter.ResolvedEntity{}, []error{err}
	}

	if apiToDeploy.IsDeprecatedApi() {
		log.Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", apiToDeploy.GetId(), apiToDeploy.IsDeprecatedBy())
	}

	var entity api.DynatraceEntity
	if apiToDeploy.IsNonUniqueNameApi() {
		configId := conf.Coordinate.ConfigId
		projectId := conf.Coordinate.Project

		entityUuid := configId

		isUuidOrMeId := util.IsUuid(entityUuid) || util.IsMeId(entityUuid)
		if !isUuidOrMeId {
			entityUuid, err = util.GenerateUuidFromConfigId(projectId, configId)
			if err != nil {
				return parameter.ResolvedEntity{}, []error{newConfigDeployError(conf, err.Error())}
			}
		}

		entity, err = client.UpsertByEntityId(apiToDeploy, entityUuid, configName, []byte(renderedConfig))
	} else {
		entity, err = client.UpsertByName(apiToDeploy, configName, []byte(renderedConfig))
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

func resolveProperties(c *config.Config, entities map[coordinate.Coordinate]parameter.ResolvedEntity) (parameter.Properties, []error) {
	var errors []error

	parameters, sortErrs := topologysort.SortParameters(c.Group, c.Environment, c.Coordinate, c.Parameters)
	errors = append(errors, sortErrs...)

	properties, errs := ResolveParameterValues(c, entities, parameters)
	errors = append(errors, errs...)

	if len(errors) > 0 {
		return nil, errors
	}

	return properties, nil
}

func renderConfig(c *config.Config, properties parameter.Properties) (string, error) {
	renderedConfig, err := template.Render(c.Template, properties)
	if err != nil {
		return "", err
	}

	err = util.ValidateJson(renderedConfig, util.Location{
		Coordinate:       c.Coordinate,
		Group:            c.Group,
		Environment:      c.Environment,
		TemplateFilePath: c.Template.Name(),
	})

	if err != nil {
		return "", &invalidJsonError{
			Config: c.Coordinate,
			EnvironmentDetails: configErrors.EnvironmentDetails{
				Group:       c.Group,
				Environment: c.Environment,
			},
			error: err,
		}
	}

	return renderedConfig, nil
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

func deploySetting(client rest.SettingsClient, entities map[coordinate.Coordinate]parameter.ResolvedEntity, c *config.Config) (parameter.ResolvedEntity, []error) {

	settings, err := client.ListKnownSettings([]string{c.Type.Schema})
	if err != nil {
		// continue & dry run missing
		return parameter.ResolvedEntity{}, []error{fmt.Errorf("failed to list known settings: %w", err)}
	}

	properties, errors := resolveProperties(c, entities)
	if len(errors) > 0 {
		return parameter.ResolvedEntity{}, errors
	}

	renderedConfig, err := renderConfig(c, properties)
	if err != nil {
		return parameter.ResolvedEntity{}, []error{err}
	}

	e, err := client.Upsert(settings, rest.SettingsObject{
		Id:            c.Coordinate.ConfigId,
		Schema:        c.Type.Schema,
		SchemaVersion: c.Type.SchemaVersion,
		Scope:         c.Type.Scope,
		Content:       []byte(renderedConfig),
	})
	if err != nil {
		return parameter.ResolvedEntity{}, []error{newConfigDeployError(c, err.Error())}
	}

	properties[config.IdParameter] = e.Id
	properties[config.NameParameter] = e.Name

	return parameter.ResolvedEntity{
		EntityName: e.Name,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil

}

func ResolveParameterValues(
	conf *config.Config,
	entities map[coordinate.Coordinate]parameter.ResolvedEntity,
	parameters []topologysort.ParameterWithName,
) (parameter.Properties, []error) {

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
