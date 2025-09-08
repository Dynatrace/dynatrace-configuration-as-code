/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

// TestReplaceAllWithDocumentRestrictionsDisabled tests that replaceAll works as expected with document restrictions disabled.
func TestReplaceAllWithDocumentRestrictionsDisabled(t *testing.T) {
	t.Setenv(featureflags.RestrictDocumentReferenceCreation.EnvName(), "false")
	tc := []struct {
		content  string
		key      string
		value    string
		confType config.Type
		expected string
	}{
		{
			content:  "a12b",
			key:      "12",
			value:    "",
			confType: config.ClassicApiType{Api: "api"},
			expected: "a12b",
		},
		{
			content:  `"metric": "calc:service.dbcallsbookingservice",`,
			key:      "calc:service.dbcallsbooking",
			value:    "{{.calculatedmetricsservice__calcservicedbcalls__id}}",
			confType: config.ClassicApiType{Api: "api"},
			expected: `"metric": "calc:service.dbcallsbookingservice",`,
		},
		{
			content:  "1234 section_modified",
			key:      "1234",
			value:    "",
			confType: config.ClassicApiType{Api: "api"},
			expected: "1234 section_modified",
		},
		{
			content: `
"id": "A",
"metric": "calc:service.test_hubert_webrequesturl",
"spaceAggregation": "AUTO",
`,
			key:      "calc:service.test",
			value:    "{{.calculatedmetricsservice__calcservicetest__id}}",
			confType: config.ClassicApiType{Api: "api"},
			expected: `
"id": "A",
"metric": "calc:service.test_hubert_webrequesturl",
"spaceAggregation": "AUTO",
`,
		},
		{
			content:  `"metricKey": "calc:apps.mobile.__easytravelmobile.useractionduration_new"`,
			key:      "calc:apps.mobile.__easytravelmobile.useractionduration",
			value:    "",
			confType: config.ClassicApiType{Api: "api"},
			expected: `"metricKey": "calc:apps.mobile.__easytravelmobile.useractionduration_new"`,
		},
		{
			content:  `"metricKey": "calc:apps.mobile.__easytravelmobile.useractionduration_new" should be replaced`,
			key:      "calc:apps.mobile.__easytravelmobile.useractionduration_new",
			value:    "{{.calc:apps.mobile.__easytravelmobile.useractionduration_new}}",
			confType: config.ClassicApiType{Api: "api"},
			expected: `"metricKey": "{{.calc:apps.mobile.__easytravelmobile.useractionduration_new}}" should be replaced`,
		},
		{
			content:  `"metricKey": "calc:synthetic.browser.apmwithautologin.domcomplete_3",`,
			key:      "calc:synthetic.browser.apmwithautologin.domcomplete",
			value:    "",
			confType: config.ClassicApiType{Api: "api"},
			expected: `"metricKey": "calc:synthetic.browser.apmwithautologin.domcomplete_3",`,
		},
		{
			content:  `"metricKey": "calc:synthetic.browser.apmwithautologin.domcomplete_3" should be replaced,`,
			key:      "calc:synthetic.browser.apmwithautologin.domcomplete_3",
			value:    "{{something}}",
			confType: config.ClassicApiType{Api: "api"},
			expected: `"metricKey": "{{something}}" should be replaced,`,
		},
		{
			content: `"assignedEntities": [
        "c48504d9-085c-3e5d-9635-554fbcc12341"
      ],`,
			key:      "1234",
			value:    "",
			confType: config.ClassicApiType{Api: "api"},
			expected: `"assignedEntities": [
        "c48504d9-085c-3e5d-9635-554fbcc12341"
      ],`,
		},
		{
			content:  `{"content": "Go [here](https://env.dynatrace.com/ui/document/id)"}`,
			key:      "id",
			value:    "{{.document__notebook__id}}",
			confType: config.DocumentType{Kind: "dashboard"},
			expected: `{"content": "Go [here](https://env.dynatrace.com/ui/document/{{.document__notebook__id}})"}`,
		},
		{
			content:  `{"content": "See \"id\""}`,
			key:      "id",
			value:    "{{.document__notebook__id}}",
			confType: config.DocumentType{Kind: "dashboard"},
			expected: `{"content": "See \"{{.document__notebook__id}}\""}`,
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.content, func(t *testing.T) {
			t.Parallel()

			c := replaceAll(tt.content, tt.key, tt.value, tt.confType)

			assert.Equal(t, tt.expected, c)
		})
	}
}

// TestReplaceAllWithDocumentRestrictionsEnabled tests that replaceAll works as expected with document restrictions enabled.
func TestReplaceAllWithDocumentRestrictionsEnabled(t *testing.T) {
	t.Setenv(featureflags.RestrictDocumentReferenceCreation.EnvName(), "true")
	tc := []struct {
		name     string
		content  string
		key      string
		value    string
		confType config.Type
		expected string
	}{
		{
			name:     "document ID part of URL should be replaced in document config",
			content:  `{"content": "Go [here](https://env.dynatrace.com/ui/document/id)"}`,
			key:      "id",
			value:    "{{.document__notebook__id}}",
			confType: config.DocumentType{Kind: "dashboard"},
			expected: `{"content": "Go [here](https://env.dynatrace.com/ui/document/{{.document__notebook__id}})"}`,
		},
		{
			name:     "document ID  in quotes should not be replaced in document config",
			content:  `{"content": "See \"id\""}`,
			key:      "id",
			value:    "{{.document__notebook__id}}",
			confType: config.DocumentType{Kind: "dashboard"},
			expected: `{"content": "See \"id\""}`,
		},
		{
			name:     "id part of URL should be replaced in other config types",
			content:  `{"content": "Go [here](https://env.dynatrace.com/ui/object/id)"}`,
			key:      "id",
			value:    "{{.config__id}}",
			confType: config.ClassicApiType{Api: "api"},
			expected: `{"content": "Go [here](https://env.dynatrace.com/ui/object/{{.config__id}})"}`,
		},
		{
			name:     "ID in quotes should be replaced in other configs",
			content:  `{"content": "See \"id\""}`,
			key:      "id",
			value:    "{{.config__id}}",
			confType: config.ClassicApiType{Api: "api"},
			expected: `{"content": "See \"{{.config__id}}\""}`,
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := replaceAll(tt.content, tt.key, tt.value, tt.confType)

			assert.Equal(t, tt.expected, c)
		})
	}
}
