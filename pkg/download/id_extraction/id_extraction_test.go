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

package id_extraction

import (
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/deploy"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2/topologysort"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractIDsIntoYAML(t *testing.T) {
	tests := []struct {
		name  string
		given project.ConfigsPerType
		want  project.ConfigsPerType
	}{
		{
			"does nothing on empty input",
			project.ConfigsPerType{},
			project.ConfigsPerType{},
		},
		{
			"does nothing if configs don't contain meIDs",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
		},
		{
			"finds and extracts meID to parameter",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_HOST_GROUP_1234567890123456": "HOST_GROUP-1234567890123456",
							}),
						},
					},
				},
			},
		},
		{
			"finds and extracts UUID to parameter",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "00b173f7-99ab-36e6-a365-170a7c42d364", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_00b173f7_99ab_36e6_a365_170a7c42d364": "00b173f7-99ab-36e6-a365-170a7c42d364",
							}),
						},
					},
				},
			},
		},
		{
			"finds and extracts several meIDs to parameters",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "SYNTHETIC_LOCATION-0000000000000089" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_0000000000000089 }}" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_HOST_GROUP_1234567890123456":         "HOST_GROUP-1234567890123456",
								"id_SYNTHETIC_LOCATION_0000000000000089": "SYNTHETIC_LOCATION-0000000000000089",
							}),
						},
					},
				},
			},
		},
		{
			"creates only a single parameter for repeated meID",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "HOST_GROUP-1234567890123456" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_HOST_GROUP_1234567890123456": "HOST_GROUP-1234567890123456",
							}),
						},
					},
				},
			},
		},
		{
			"creates only a single parameter for repeated UUIDs",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "00b173f7-99ab-36e6-a365-170a7c42d364", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "00b173f7-99ab-36e6-a365-170a7c42d364" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_00b173f7_99ab_36e6_a365_170a7c42d364": "00b173f7-99ab-36e6-a365-170a7c42d364",
							}),
						},
					},
				},
			},
		},
		{
			"correctly extracts an updates all configs",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "HOST_GROUP-1234567890123456" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "SYNTHETIC_LOCATION-0000000000000089" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
				"other-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "00b173f7-99ab-36e6-a365-170a7c42d364", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_HOST_GROUP_1234567890123456": "HOST_GROUP-1234567890123456",
							}),
						},
					},
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_0000000000000089 }}" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_HOST_GROUP_1234567890123456":         "HOST_GROUP-1234567890123456",
								"id_SYNTHETIC_LOCATION_0000000000000089": "SYNTHETIC_LOCATION-0000000000000089",
							}),
						},
					},
				},
				"other-type": []config.Config{
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_4242424242424242 }}", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_SYNTHETIC_LOCATION_4242424242424242": "SYNTHETIC_LOCATION-4242424242424242",
							}),
						},
					},
					{
						Template: template.CreateTemplateFromString("test-tmpl", `{ "key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_4242424242424242 }}", "details": { "d1_key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_SYNTHETIC_LOCATION_4242424242424242":  "SYNTHETIC_LOCATION-4242424242424242",
								"id_00b173f7_99ab_36e6_a365_170a7c42d364": "00b173f7-99ab-36e6-a365-170a7c42d364",
							}),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIDsIntoYAML(tt.given)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractedTemplatesRenderCorrectly(t *testing.T) {
	given := project.ConfigsPerType{
		"test-type": []config.Config{
			{
				Template: template.CreateTemplateFromString("test-tmpl-1", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "HOST_GROUP-1234567890123456" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
			{
				Template: template.CreateTemplateFromString("test-tmpl-2", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "SYNTHETIC_LOCATION-0000000000000089" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
		},
		"other-type": []config.Config{
			{
				Template: template.CreateTemplateFromString("test-tmpl-3", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
			{
				Template: template.CreateTemplateFromString("test-tmpl-4", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
			{
				Template: template.CreateTemplateFromString("test-tmpl-5", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "HOST_GROUP-1234567890123456", "d2_key": "00b173f7-99ab-36e6-a365-170a7c42d364" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
		},
	}

	got := ExtractIDsIntoYAML(given)

	for _, cfgs := range got {
		for _, c := range cfgs {
			sortedParams, errs := topologysort.SortParameters("", "", c.Coordinate, c.Parameters)
			assert.Empty(t, errs)
			props, errs := deploy.ResolveParameterValues(&c, nil, sortedParams)
			assert.Empty(t, errs)
			_, err := c.Render(props)
			assert.NoError(t, err)
		}
	}
}
