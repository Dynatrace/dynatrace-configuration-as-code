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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	envParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	refParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
)

const (
	// IdParameter is special. it is not allowed to be set via the config,
	// but needs to work as normal parameter otherwise (e.g. it can be referenced).
	IdParameter = "id"

	// NameParameter is special in that it needs to exist for a config.
	NameParameter = "name"
)

// Parameters defines a map of name to parameter
type Parameters map[string]parameter.Parameter

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
	// map of all parameters which will be resolved and are then available
	// in the template
	Parameters Parameters
	// References specified by the paramter. needed so that the
	// configurations can be sorted in before the deployment phase.
	References []coordinate.Coordinate
	// Skip flag inidicating if the deployment of this configuration should be skipped
	Skip bool
}

// MatchReference is used to determine if this config represents the given coordinates
func (c *Config) MatchReference(ref coordinate.Coordinate) bool {
	return c.Coordinate.Match(ref)
}

// HasDependencyOn tests if this config has a dependency on the given config.
func (c *Config) HasDependencyOn(config Config) bool {
	for _, ref := range c.References {
		if config.MatchReference(ref) {
			return true
		}
	}

	return false
}

// DefaultParameterParsers map defining a set of default parsers which can be used to load configurations
var DefaultParameterParsers = map[string]parameter.ParameterSerDe{
	refParam.ReferenceParameterType:           refParam.ReferenceParameterSerde,
	valueParam.ValueParameterType:             valueParam.ValueParameterSerde,
	envParam.EnvironmentVariableParameterType: envParam.EnvironmentVariableParameterSerde,
}
