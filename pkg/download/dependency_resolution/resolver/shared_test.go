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

package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceAll(t *testing.T) {
	tc := []struct {
		content  string
		key      string
		value    string
		expected string
	}{
		{
			"a12b",
			"12",
			"",
			"a12b",
		},
		{
			`"metric": "calc:service.dbcallsbookingservice",`,
			"calc:service.dbcallsbooking",
			"{{.calculatedmetricsservice__calcservicedbcalls__id}}",
			`"metric": "calc:service.dbcallsbookingservice",`,
		},
		{
			"1234 section_modified",
			"1234",
			"",
			"1234 section_modified",
		},
		{
			`
"id": "A",
"metric": "calc:service.test_hubert_webrequesturl",
"spaceAggregation": "AUTO",
`,
			"calc:service.test",
			"{{.calculatedmetricsservice__calcservicetest__id}}",
			`
"id": "A",
"metric": "calc:service.test_hubert_webrequesturl",
"spaceAggregation": "AUTO",
`,
		},
		{
			`"metricKey": "calc:apps.mobile.__easytravelmobile.useractionduration_new"`,
			"calc:apps.mobile.__easytravelmobile.useractionduration",
			"",
			`"metricKey": "calc:apps.mobile.__easytravelmobile.useractionduration_new"`,
		},
		{
			`"metricKey": "calc:apps.mobile.__easytravelmobile.useractionduration_new" should be replaced`,
			"calc:apps.mobile.__easytravelmobile.useractionduration_new",
			"{{.calc:apps.mobile.__easytravelmobile.useractionduration_new}}",
			`"metricKey": "{{.calc:apps.mobile.__easytravelmobile.useractionduration_new}}" should be replaced`,
		},
		{
			`"metricKey": "calc:synthetic.browser.apmwithautologin.domcomplete_3",`,
			"calc:synthetic.browser.apmwithautologin.domcomplete",
			"",
			`"metricKey": "calc:synthetic.browser.apmwithautologin.domcomplete_3",`,
		},
		{
			`"metricKey": "calc:synthetic.browser.apmwithautologin.domcomplete_3" should be replaced,`,
			"calc:synthetic.browser.apmwithautologin.domcomplete_3",
			"{{something}}",
			`"metricKey": "{{something}}" should be replaced,`,
		},
		{
			`"assignedEntities": [
        "c48504d9-085c-3e5d-9635-554fbcc12341"
      ],`,
			"1234",
			"",
			`"assignedEntities": [
        "c48504d9-085c-3e5d-9635-554fbcc12341"
      ],`,
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.content, func(t *testing.T) {

			c := replaceAll(tt.content, tt.key, tt.value)

			assert.Equal(t, tt.expected, c)
		})
	}
}
