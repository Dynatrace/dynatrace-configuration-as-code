//go:build unit

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

package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name            string
		given           []project.Project
		wantErrsContain map[string][]string
	}{
		{
			name: "no duplicates if from different environment",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
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
						"env2": project.ConfigsPerType{
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
		},
		{
			name: "no duplicates if for different api",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
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
		},
		{
			name: "no duplicates if for api allow duplicate names",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
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
		},
		{
			name: "duplicate by value parameters",
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1", "duplicate-name"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"test1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "duplicate-name",
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
											Value: "duplicate-name",
										},
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
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1", "default-name" +
					""},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
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
											DefaultValue:    "default-name",
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
											DefaultValue:    "default-name",
										},
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
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
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
		},
		{
			name: "duplicate by reference parameter",
			wantErrsContain: map[string][]string{
				"env1": {"::config2", "::config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
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
		},
		{
			name: "duplicate in different projects",
			wantErrsContain: map[string][]string{
				"env1": {"p2:type:config2", "p1:type:config1"},
			},
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"p1": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										Project:  "p1",
										Type:     "type",
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.NameParameter: &value.ValueParameter{
											Value: "value",
										},
									},
								},
							},
							"p2": {
								config.Config{
									Type:        config.ClassicApiType{Api: "app-detection-rule"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										Project:  "p2",
										Type:     "type",
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
		},
		{
			name: "settings scope as string value OK",
			given: []project.Project{
				{
					Configs: project.ConfigsPerTypePerEnvironments{
						"env1": project.ConfigsPerType{
							"builtin:setting": {
								config.Config{
									Type:        config.SettingsType{SchemaId: "builtin:setting"},
									Environment: "env1",
									Coordinate: coordinate.Coordinate{
										ConfigId: "config1",
									},
									Parameters: config.Parameters{
										config.ScopeParameter: &value.ValueParameter{
											Value: "HOST-12345",
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
			err := Validate(tc.given)
			if len(tc.wantErrsContain) == 0 {
				assert.NoError(t, err)
			} else {
				for env, errStrings := range tc.wantErrsContain {
					assert.ErrorContains(t, err, env)
					for _, s := range errStrings {
						assert.ErrorContains(t, err, s)
					}
				}
			}
		})
	}
}
