//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package template

import (
	"reflect"
	"testing"
	templ "text/template" // nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template
)

const (
	simpleTemplateString       = "{ \"key\": {{ .val }} }"
	invalidTemplateString      = "{ \"key\": {{ .val }"
	templateStringWithNewlines = `{ "key":
{{ .val }}
}`
	templateString = "Follow the {{.color}} {{ .animal }}"
)

func TestParseTemplate(t *testing.T) {

	emptyTemplate, _ := templ.New("").Option("missingkey=error").Parse("")
	expectedTemplate, _ := templ.New("id").Option("missingkey=error").Parse(simpleTemplateString)

	type args struct {
		id      string
		content string
	}
	tests := []struct {
		name    string
		args    args
		want    *templ.Template
		wantErr bool
	}{
		{
			name: "doesn't fail on empty input",
			args: args{
				"",
				"",
			},
			want:    emptyTemplate,
			wantErr: false,
		},
		{
			name: "parses template",
			args: args{
				"id",
				simpleTemplateString,
			},
			want:    expectedTemplate,
			wantErr: false,
		},
		{
			name: "returns error on incomplete template",
			args: args{
				"id",
				invalidTemplateString,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTemplate(tt.args.id, tt.args.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTemplate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name            string
		givenTemplate   Template
		givenProperties map[string]interface{}
		want            string
		wantErr         bool
	}{
		{
			"renders simple template",
			&fileBasedTemplate{
				path:    "a path",
				content: simpleTemplateString,
			},
			map[string]interface{}{"val": "the-key"},
			`{ "key": the-key }`,
			false,
		},
		{
			"renders template #1",
			&fileBasedTemplate{
				path:    "a path",
				content: templateString,
			},
			map[string]interface{}{"color": "white", "animal": "rabbit"},
			`Follow the white rabbit`,
			false,
		},
		{
			"renders template #2",
			&fileBasedTemplate{
				path:    "a path",
				content: templateString,
			},
			map[string]interface{}{"color": "white", "animal": "cow"},
			`Follow the white cow`,
			false,
		},
		{
			"renders template - random characters in property",
			&fileBasedTemplate{
				path:    "a path",
				content: templateString,
			},
			map[string]interface{}{"color": "white", "animal": "rabbit$=co\\/\\/=chicken"},
			`Follow the white rabbit$=co\/\/=chicken`,
			false,
		},
		{
			"fails if referenced property is not defined",
			&fileBasedTemplate{
				path:    "a path",
				content: simpleTemplateString,
			},
			map[string]interface{}{}, // 'val' used in template but not defined as property
			"",
			true,
		},
		{
			"fails if one referenced property is not defined",
			&fileBasedTemplate{
				path:    "a path",
				content: templateString,
			},
			map[string]interface{}{"color": "white"}, // 'val' used in template but not defined as property
			"",
			true,
		},
		{
			"fails if template string is invalid",
			&fileBasedTemplate{
				path:    "a path",
				content: invalidTemplateString,
			},
			map[string]interface{}{"val": "the-key"},
			"",
			true,
		},
		{
			"escapes any newlines when rendering template",
			&fileBasedTemplate{
				path:    "a path",
				content: templateStringWithNewlines,
			},
			map[string]interface{}{"val": "the-key"},
			"{ \"key\":\nthe-key\n}",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.givenTemplate, tt.givenProperties)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Render() got = %v, want %v", got, tt.want)
			}
		})
	}
}
