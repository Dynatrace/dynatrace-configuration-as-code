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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	compoundParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/compound"
	envParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	listParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/list"
	refParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
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

	// SkipParameter is special in that config should be deployed or not
	SkipParameter = "skip"
)

// ReservedParameterNames holds all parameter names that may not be specified by a user in a config.
var ReservedParameterNames = []string{IdParameter, ScopeParameter, SkipParameter}

// Parameters defines a map of name to parameter
type Parameters map[string]parameter.Parameter

type Type struct {
	SchemaId,
	SchemaVersion,
	Api string
	EntitiesType string
}

// IsSettings returns true if SchemaId is not empty
func (t Type) IsSettings() bool {
	return t.SchemaId != ""
}

func (t Type) IsEntities() bool {
	return t.EntitiesType != ""
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
		return "", &configErrors.InvalidJsonError{
			Config: c.Coordinate,
			EnvironmentDetails: configErrors.EnvironmentDetails{
				Group:       c.Group,
				Environment: c.Environment,
			},
			WrappedError: err,
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
}

func (c *Config) References() []coordinate.Coordinate {

	refs := make([]coordinate.Coordinate, 0)

	for _, p := range c.Parameters {
		for _, r := range p.GetReferences() {
			refs = append(refs, r.Config)
		}
	}

	return refs
}
