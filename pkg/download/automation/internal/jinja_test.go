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

package internal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEscapeJinjaTemplates(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]byte("Hello, {{`{{`}}planet{{`}}`}}!"), EscapeJinjaTemplates([]byte(`Hello, {{planet}}!`)))
	assert.Equal([]byte("Hello , {{`{{`}} calendar(\"abcde\") {{`}}`}}"), EscapeJinjaTemplates([]byte(`Hello , {{ calendar("abcde") }}`)))
	assert.Equal([]byte("no jinja"), EscapeJinjaTemplates([]byte(`no jinja`)))
	assert.Equal([]byte("{{`{{`}}"), EscapeJinjaTemplates([]byte(`{{`)))
	assert.Equal([]byte("{"), EscapeJinjaTemplates([]byte(`{`)))
	assert.Equal([]byte("\\{"), EscapeJinjaTemplates([]byte(`\{`)))
	assert.Equal([]byte("{{`}}`}}"), EscapeJinjaTemplates([]byte(`}}`)))
	assert.Equal([]byte("}"), EscapeJinjaTemplates([]byte(`}`)))
	assert.Equal([]byte("\\}"), EscapeJinjaTemplates([]byte(`\}`)))
	assert.Equal([]byte(nil), EscapeJinjaTemplates(nil))
	assert.Equal([]byte("{{`{{`}} {{`}}`}}"), EscapeJinjaTemplates([]byte(`{{ }}`)))
}
