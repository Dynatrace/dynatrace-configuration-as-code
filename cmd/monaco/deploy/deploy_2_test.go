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

package deploy_test

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_checkUniquenessOfNameForClassicConfig(t *testing.T) {
	tests := []struct {
		name     string
		given    project.Project
		expected []error
	}{
		{
			name:     "no duplicates if from different environment",
			expected: nil,
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
						},
					},
					"env2": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env2",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "no duplicates if for different api",
			expected: nil,
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
							config.Config{
								Type:        config.ClassicApiType{Api: "custom-service-php"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "no duplicates if for api allow duplicate names",
			expected: nil,
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
							config.Config{
								Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by value parameters",
			expected: []error{
				errors.New("configuration with coordinates \"::config2\" and \"::config1\" have same \"name\" values"),
			},
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by environment parameter",
			expected: []error{
				errors.New("configuration with coordinates \"::config2\" and \"::config1\" have same \"name\" values"),
			},
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &environment.EnvironmentVariableParameter{
										Name:            "ENV_1",
										HasDefaultValue: true,
										DefaultValue:    "value",
									},
								},
							},
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &environment.EnvironmentVariableParameter{
										Name:            "ENV_1",
										HasDefaultValue: true,
										DefaultValue:    "value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by mix of value and environment parameter",
			expected: []error{
				errors.New("configuration with coordinates \"::config2\" and \"::config1\" have same \"name\" values"),
			},
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &environment.EnvironmentVariableParameter{
										Name:            "ENV_1",
										HasDefaultValue: true,
										DefaultValue:    "value",
									},
								},
							},
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &value.ValueParameter{
										Value: "value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate by reference parameter",
			expected: []error{
				errors.New("configuration with coordinates \"::config2\" and \"::config1\" have same \"name\" values"),
			},
			given: project.Project{
				Configs: map[project.EnvironmentName]project.ConfigsPerType{
					"env1": map[string][]config.Config{
						"test1": {
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config1",
								},
								Parameters: config.Parameters{
									config.NameParameter: &reference.ReferenceParameter{
										ParameterReference: parameter.ParameterReference{
											Config: coordinate.Coordinate{
												Project:  "projA",
												Type:     "typeA",
												ConfigId: "ID",
											},
											Property: "property",
										},
									},
								},
							},
							config.Config{
								Type:        config.ClassicApiType{Api: "app-detection-rule"},
								Environment: "env1",
								Coordinate: coordinate.Coordinate{
									ConfigId: "config2",
								},
								Parameters: config.Parameters{
									config.NameParameter: &reference.ReferenceParameter{
										ParameterReference: parameter.ParameterReference{
											Config: coordinate.Coordinate{
												Project:  "projA",
												Type:     "typeA",
												ConfigId: "ID",
											},
											Property: "property",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			errs := deploy.CheckUniquenessOfNameForClassicConfig([]project.Project{tc.given})
			assert.Equal(t, tc.expected, errs)
		})
	}
}
