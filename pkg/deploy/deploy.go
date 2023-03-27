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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
)

// DeployConfigsOptions defines additional options used by DeployConfigs
type DeployConfigsOptions struct {
	// ContinueOnErr states that the deployment continues even when there happens to be an
	// error while deploying a certain configuration
	ContinueOnErr bool
	// DryRun states that the deployment shall just run in dry-run mode, meaning
	// that actual deployment of the configuration to a tenant will be skipped
	DryRun bool
}

// DeployConfigs deploys the given configs with the given apis via the given client
// NOTE: the given configs need to be sorted, otherwise deployment will
// probably fail, as references cannot be resolved
func DeployConfigs(client client.Client, apis api.APIs, sortedConfigs []config.Config, opts DeployConfigsOptions) []error {
	entityMap := newEntityMap(apis)
	var errors []error

	for _, c := range sortedConfigs {
		c := c // to avoid implicit memory aliasing (gosec G601)

		if c.Skip {
			log.Info("\tSkipping deployment of config %s", c.Coordinate)

			entityMap.put(c.Coordinate, parameter.ResolvedEntity{
				EntityName: c.Coordinate.ConfigId,
				Coordinate: c.Coordinate,
				Properties: parameter.Properties{},
				Skip:       true,
			})
			continue
		}

		logAction, logVerb := getWordsForLogging(opts.DryRun)
		log.Info("\t%s config %s", logAction, c.Coordinate)

		var entity parameter.ResolvedEntity
		var deploymentErrors []error

		switch {
		case c.Type.IsEntities():
			log.Debug("Entities are not deployable, skipping entity type: %s", c.Type.EntitiesType)
		case c.Type.IsSettings():
			entity, deploymentErrors = deploySetting(client, entityMap, &c)
		default:
			entity, deploymentErrors = deployConfig(client, apis, entityMap, &c)
		}

		if deploymentErrors != nil {
			for _, err := range deploymentErrors {
				errors = append(errors, fmt.Errorf("failed to %s config %s: %w", logVerb, c.Coordinate, err))
			}

			if !opts.ContinueOnErr && !opts.DryRun {
				return errors
			}
		}
		entityMap.put(entity.Coordinate, entity)
	}

	return errors
}

// getWordsForLogging returns fitting action and verb words to clearly tell a user if configuration is
// deployed or validated when logging based on the dry-run boolean
func getWordsForLogging(isDryRun bool) (action, verb string) {
	if isDryRun {
		return "Validating", "validate"
	}
	return "Deploying", "deploy"
}

func deployConfig(configClient client.ConfigClient, apis api.APIs, entityMap *entityMap, conf *config.Config) (parameter.ResolvedEntity, []error) {

	if !apis.Contains(conf.Coordinate.Type) {
		return parameter.ResolvedEntity{}, []error{fmt.Errorf("unknown api `%s`. this is most likely a bug", conf.Type.Api)}
	}

	apiToDeploy := apis[conf.Coordinate.Type]
	properties, errors := resolveProperties(conf, entityMap.get())
	if len(errors) > 0 {
		return parameter.ResolvedEntity{}, errors
	}

	configName, err := extractConfigName(conf, properties)
	if err != nil {
		errors = append(errors, err)
	} else if entityMap.contains(apiToDeploy.ID, configName) && !apiToDeploy.NonUniqueName {
		errors = append(errors, newConfigDeployErr(conf, fmt.Sprintf("duplicated config name `%s`", configName)))
	}
	if len(errors) > 0 {
		return parameter.ResolvedEntity{}, errors
	}

	renderedConfig, err := conf.Render(properties)
	if err != nil {
		return parameter.ResolvedEntity{}, []error{err}
	}

	if apiToDeploy.DeprecatedBy != "" {
		log.Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", apiToDeploy.ID, apiToDeploy.DeprecatedBy)
	}

	var entity client.DynatraceEntity
	if apiToDeploy.NonUniqueName {
		entity, err = upsertNonUniqueNameConfig(configClient, apiToDeploy, conf, configName, renderedConfig)
	} else {
		entity, err = configClient.UpsertConfigByName(apiToDeploy, configName, []byte(renderedConfig))
	}

	if err != nil {
		return parameter.ResolvedEntity{}, []error{newConfigDeployErr(conf, err.Error())}
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

func upsertNonUniqueNameConfig(client client.ConfigClient, apiToDeploy api.API, conf *config.Config, configName string, renderedConfig string) (client.DynatraceEntity, error) {
	configId := conf.Coordinate.ConfigId
	projectId := conf.Coordinate.Project

	entityUuid := configId

	isUuidOrMeId := idutils.IsUuid(entityUuid) || idutils.IsMeId(entityUuid)
	if !isUuidOrMeId {
		entityUuid = idutils.GenerateUuidFromConfigId(projectId, configId)
	}

	return client.UpsertConfigByNonUniqueNameAndId(apiToDeploy, entityUuid, configName, []byte(renderedConfig))
}

func deploySetting(settingsClient client.SettingsClient, entityMap *entityMap, c *config.Config) (parameter.ResolvedEntity, []error) {
	properties, errors := resolveProperties(c, entityMap.get())
	if len(errors) > 0 {
		return parameter.ResolvedEntity{}, errors
	}

	scope, err := extractScope(properties)
	if err != nil {
		return parameter.ResolvedEntity{}, []error{err}
	}

	renderedConfig, err := c.Render(properties)
	if err != nil {
		return parameter.ResolvedEntity{}, []error{err}
	}

	entity, err := settingsClient.UpsertSettings(client.SettingsObject{
		Id:             c.Coordinate.ConfigId,
		SchemaId:       c.Type.SchemaId,
		SchemaVersion:  c.Type.SchemaVersion,
		Scope:          scope,
		Content:        []byte(renderedConfig),
		OriginObjectId: c.OriginObjectId,
	})
	if err != nil {
		return parameter.ResolvedEntity{}, []error{newConfigDeployErr(c, err.Error())}
	}

	name := fmt.Sprintf("[UNKNOWN NAME]%s", entity.Id)
	if configName, err := extractConfigName(c, properties); err == nil {
		name = configName
	} else {
		log.Warn("failed to extract name for Settings 2.0 object %q - ID will be used", entity.Id)
	}

	properties[config.IdParameter] = entity.Id
	properties[config.NameParameter] = name

	return parameter.ResolvedEntity{
		EntityName: name,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil

}

func extractScope(properties parameter.Properties) (string, error) {
	scope, ok := properties[config.ScopeParameter]
	if !ok {
		return "", fmt.Errorf("property '%s' not found, this is most likely a bug", config.ScopeParameter)
	}

	if scope == "" {
		return "", fmt.Errorf("resolved scope is empty")
	}

	return fmt.Sprint(scope), nil
}
