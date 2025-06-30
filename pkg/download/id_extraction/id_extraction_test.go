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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	ref "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "00b173f7-99ab-36e6-a365-170a7c42d364", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "SYNTHETIC_LOCATION-0000000000000089" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_0000000000000089 }}" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "HOST_GROUP-1234567890123456" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "00b173f7-99ab-36e6-a365-170a7c42d364", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "00b173f7-99ab-36e6-a365-170a7c42d364" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "HOST_GROUP-1234567890123456" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "SYNTHETIC_LOCATION-0000000000000089" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
				"other-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "00b173f7-99ab-36e6-a365-170a7c42d364", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_HOST_GROUP_1234567890123456": "HOST_GROUP-1234567890123456",
							}),
						},
					},
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_HOST_GROUP_1234567890123456 }}", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_0000000000000089 }}" } }`),
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
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
						},
					},
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_4242424242424242 }}", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
						Parameters: config.Parameters{
							"baseParam": value.New("base-value"),
							"extractedIDs": value.New(map[string]string{
								"id_SYNTHETIC_LOCATION_4242424242424242": "SYNTHETIC_LOCATION-4242424242424242",
							}),
						},
					},
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{ "key": "{{ .extractedIDs.id_SYNTHETIC_LOCATION_4242424242424242 }}", "details": { "d1_key": "{{ .extractedIDs.id_00b173f7_99ab_36e6_a365_170a7c42d364 }}", "d2_key": "d2_val" } }`),
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
			got, gotErr := ExtractIDsIntoYAML(tt.given)
			assert.NoError(t, gotErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScopeParameterIsTreatedAsParameter(t *testing.T) {
	t.Setenv(featureflags.ExtractScopeAsParameter.EnvName(), "1")

	tests := []struct {
		name  string
		given project.ConfigsPerType
		want  project.ConfigsPerType
	}{
		{
			"scope parameter treated as separate param",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", "{}"),
						Parameters: config.Parameters{
							"scope": value.New("HOST-123456789"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", "{}"),
						Parameters: config.Parameters{
							"scope": &ref.ReferenceParameter{ParameterReference: parameter.ParameterReference{Property: baseParamID + ".id_HOST_123456789"}},
							"extractedIDs": value.New(map[string]string{
								"id_HOST_123456789": "HOST-123456789",
							}),
						},
					},
				},
			},
		},
		{
			"invalid param key chars (dot,hyphen) are removed from scope parameters",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", "{}"),
						Parameters: config.Parameters{
							"scope": value.New("my.magic.metric-key"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", "{}"),
						Parameters: config.Parameters{
							"scope": &ref.ReferenceParameter{ParameterReference: parameter.ParameterReference{Property: baseParamID + ".id_my_magic_metric_key"}},
							"extractedIDs": value.New(map[string]string{
								"id_my_magic_metric_key": "my.magic.metric-key",
							}),
						},
					},
				},
			},
		},
		{
			"scope parameter with environment value is not treated as separate param",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", "{}"),
						Parameters: config.Parameters{
							"scope": value.New("environment"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", "{}"),
						Parameters: config.Parameters{
							"scope": value.New("environment"),
						},
					},
				},
			},
		},
		{
			"scope parameter with environment value is not treated as separate param",
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("test-tmpl", `{"key": "value\nCLOUD_APPLICATION-0000000000000000"}`),
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"test-type": []config.Config{
					{
						Template: template.NewInMemoryTemplate("test-tmpl", `{"key": "value\n{{ .extractedIDs.id_CLOUD_APPLICATION_0000000000000000 }}"}`),
						Parameters: config.Parameters{
							"extractedIDs": value.New(map[string]string{"id_CLOUD_APPLICATION_0000000000000000": "CLOUD_APPLICATION-0000000000000000"}),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := ExtractIDsIntoYAML(tt.given)
			assert.NoError(t, gotErr)

			if diff := cmp.Diff(got, tt.want, cmpopts.EquateComparable(template.InMemoryTemplate{})); diff != "" {
				t.Errorf("ExtractIDsIntoYAML() returned diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestExtractedTemplatesRenderCorrectly(t *testing.T) {
	given := project.ConfigsPerType{
		"test-type": []config.Config{
			{
				Template: template.NewInMemoryTemplate("test-tmpl-1", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "HOST_GROUP-1234567890123456" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
			{
				Template: template.NewInMemoryTemplate("test-tmpl-2", `{ "key": "HOST_GROUP-1234567890123456", "details": { "d1_key": "AWS_RELATIONAL_DATABASE_SERVICE", "d2_key": "SYNTHETIC_LOCATION-0000000000000089" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
		},
		"other-type": []config.Config{
			{
				Template: template.NewInMemoryTemplate("test-tmpl-3", `{ "key": "value", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
			{
				Template: template.NewInMemoryTemplate("test-tmpl-4", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "d1_val", "d2_key": "d2_val" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
			{
				Template: template.NewInMemoryTemplate("test-tmpl-5", `{ "key": "SYNTHETIC_LOCATION-4242424242424242", "details": { "d1_key": "HOST_GROUP-1234567890123456", "d2_key": "00b173f7-99ab-36e6-a365-170a7c42d364" } }`),
				Parameters: config.Parameters{
					"baseParam": value.New("base-value"),
				},
			},
		},
	}

	got, gotErr := ExtractIDsIntoYAML(given)
	assert.NoError(t, gotErr)

	for _, cfgs := range got {
		for _, c := range cfgs {
			props, errs := c.ResolveParameterValues(nil)
			assert.Empty(t, errs)
			_, err := c.Render(props)
			assert.NoError(t, err)
		}
	}
}

func TestFindAllIds(t *testing.T) {

	tc := []struct {
		in          string
		expectedIds []string
	}{
		{"", nil},
		{
			"HOST-0123456789ABCDEF",
			[]string{"HOST-0123456789ABCDEF"},
		},
		{
			"f1614cf1-4f6e-4187-b303-af4beb42268c",
			[]string{"f1614cf1-4f6e-4187-b303-af4beb42268c"},
		},
		{
			`{"HOST": "HOST-0123456789ABCDEF", "id": "f1614cf1-4f6e-4187-b303-af4beb42268c"}`,
			[]string{"f1614cf1-4f6e-4187-b303-af4beb42268c", "HOST-0123456789ABCDEF"},
		},
		{
			"HELLO-Imnotanentityidbutstilliwasdetectedassuch",
			nil,
		},
		{
			`\nABC-0000000000000000`,
			[]string{"ABC-0000000000000000"},
		},
		{
			`\rABC-0000000000000000`,
			[]string{"ABC-0000000000000000"},
		},
		{
			`\tABC-0000000000000000`,
			[]string{"ABC-0000000000000000"},
		},
		{
			`\\tABC-0000000000000000`,
			[]string{"ABC-0000000000000000"},
		},
		{
			`\nf1614cf1-4f6e-4187-b303-af4beb42268c`,
			[]string{"f1614cf1-4f6e-4187-b303-af4beb42268c"},
		},
	}

	for _, tt := range tc {
		t.Run(tt.in, func(t *testing.T) {

			foundIds := findAllIds(tt.in)
			assert.ElementsMatch(t, tt.expectedIds, foundIds)
		})
	}

}
