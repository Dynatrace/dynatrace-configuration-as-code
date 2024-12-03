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

package config

import (
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	compoundParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	fileParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/file"
	listParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/list"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

const (
	// IdParameter is special. it is not allowed to be set via the config,
	// but needs to work as normal parameter otherwise (e.g. it can be referenced).
	IdParameter = "id"

	// NameParameter is special in that it needs to exist for a config.
	NameParameter = "name"

	// ScopeParameter is special. It is the set scope as a parameter.
	// A user must not set it as a parameter in the config.
	// It is only a parameter iff the config is a settings-config.
	ScopeParameter = "scope"

	// InsertAfterParameter is special. It points to another settings object and is used
	// for establishing an ordering between different settings objects.
	// It is only a parameter iff the config is a settings-config.
	InsertAfterParameter = "insertAfter"

	// SkipParameter is special in that config should be deployed or not
	SkipParameter = "skip"

	// NonUniqueNameConfigDuplicationParameter is a special parameter set on non-unique name API configurations
	// that appear multiple times in a project
	NonUniqueNameConfigDuplicationParameter = "__MONACO_NUN_API_DUP__"
)

// ReservedParameterNames holds all parameter names that may not be specified by a user in a config.
var ReservedParameterNames = []string{IdParameter, NameParameter, ScopeParameter, SkipParameter}

// Parameters defines a map of name to parameter
type Parameters map[string]parameter.Parameter

type TypeID string

type Type interface {
	// ID returns the type-id.
	ID() TypeID
}

// Config struct defining a configuration which can be deployed.
type Config struct {
	// template used to render the request send to the dynatrace api
	Template template.Template
	// coordinates which specify the location of this configuration
	Coordinate coordinate.Coordinate
	// group this config belongs to
	Group string
	// name of the environment this configuration is for
	Environment string
	// Type holds information of the underlying config type (classic, settings, entities)
	Type Type
	// map of all parameters which will be resolved and are then available
	// in the template
	Parameters Parameters

	// Skip flag indicates if the deployment of this configuration should be skipped. It is resolved during project loading.
	Skip bool

	// SkipForConversion is only used for converting v1-configs to v2-configs.
	// It is required as the object itself does only store the resolved 'skip' value, not the actual parameter.
	SkipForConversion parameter.Parameter

	// OriginObjectId is the DT object ID of the object when it was downloaded from an environment
	OriginObjectId string
}

func (c *Config) Render(properties map[string]interface{}) (string, error) {
	if c == nil || c.Template == nil {
		return "", nil
	}

	var templatePath string // include path in errors if we know it
	if t, ok := c.Template.(*template.FileBasedTemplate); ok {
		templatePath = t.FilePath()
	}

	renderedConfig, err := template.Render(c.Template, properties)
	if err != nil {
		return "", configErrors.InvalidJsonError{
			Location: c.Coordinate,
			EnvironmentDetails: configErrors.EnvironmentDetails{
				Group:       c.Group,
				Environment: c.Environment,
			},
			Err:              err,
			TemplateFilePath: templatePath,
		}
	}

	err = json.ValidateJson(renderedConfig, json.Location{
		Coordinate:       c.Coordinate,
		Group:            c.Group,
		Environment:      c.Environment,
		TemplateFilePath: templatePath,
	})

	if err != nil {
		return "", configErrors.InvalidJsonError{
			Location: c.Coordinate,
			EnvironmentDetails: configErrors.EnvironmentDetails{
				Group:       c.Group,
				Environment: c.Environment,
			},
			Err:              err,
			TemplateFilePath: templatePath,
		}
	}

	return renderedConfig, nil
}

// DefaultParameterParsers map defining a set of default parsers which can be used to load configurations
var DefaultParameterParsers = map[string]parameter.ParameterSerDe{
	refParam.ReferenceParameterType:           refParam.ReferenceParameterSerde,
	valueParam.ValueParameterType:             valueParam.ValueParameterSerde,
	envParam.EnvironmentVariableParameterType: envParam.EnvironmentVariableParameterSerde,
	compoundParam.CompoundParameterType:       compoundParam.CompoundParameterSerde,
	listParam.ListParameterType:               listParam.ListParameterSerde,
	fileParam.FileParameterType:               fileParam.FileParameterSerde,
}

func (c *Config) References() []coordinate.Coordinate {
	if c == nil {
		return nil
	}

	count := 0
	for _, p := range c.Parameters {
		count += len(p.GetReferences())
	}

	refs := make([]coordinate.Coordinate, 0, count)
	for _, p := range c.Parameters {
		references := p.GetReferences()
		for i := range references {
			refs = append(refs, references[i].Config)
		}
	}

	return refs
}

// HasRefTo returns true if he config has a reference to another config of the given type
func (c *Config) HasRefTo(configType string) bool {
	refs := c.References()
	for _, r := range refs {
		if r.Type == configType {
			return true
		}
	}
	return false
}

// EntityLookup is used in parameter resolution to fetch the resolved entity of deployed configuration
type EntityLookup interface {
	parameter.PropertyResolver

	GetResolvedEntity(config coordinate.Coordinate) (entities.ResolvedEntity, bool)
}

// ResolveParameterValues will resolve the values of all config.Parameters of a config.Config and return them as a parameter.Properties map.
// Resolving will ensure that parameters are resolved in the right order if they have dependencies between each other.
// To be able to resolve reference.ReferenceParameter values an EntityLookup needs to be provided, which contains all
// config.ResolvedEntity values of configurations that the config.Config could depend on.
// Ordering of configurations to ensure that possible dependency configurations are contained in teh EntityLookup is responsibility
// of the caller of ResolveParameterValues.
//
// ResolveParameterValues will return a slice of errors for any failures during sorting or resolving parameters.
func (c *Config) ResolveParameterValues(entities EntityLookup) (parameter.Properties, []error) {
	if c == nil {
		return nil, nil
	}

	var errors []error

	parameters, sortErrs := getSortedParameters(c)
	errors = append(errors, sortErrs...)

	properties, errs := resolveValues(c, entities, parameters)
	errors = append(errors, errs...)

	if len(errors) > 0 {
		return nil, errors
	}

	return properties, nil
}

func GetNameForConfig(c Config) (any, error) {
	nameParam, exist := c.Parameters[NameParameter]
	if !exist {
		return nil, fmt.Errorf("configuration %s has no 'name' parameter defined", c.Coordinate)
	}

	switch v := nameParam.(type) {
	case *valueParam.ValueParameter:
		return v.ResolveValue(parameter.ResolveContext{ParameterName: NameParameter})
	case *envParam.EnvironmentVariableParameter:
		return v.ResolveValue(parameter.ResolveContext{ParameterName: NameParameter})
	default:
		return c.Parameters[NameParameter], nil
	}
}
