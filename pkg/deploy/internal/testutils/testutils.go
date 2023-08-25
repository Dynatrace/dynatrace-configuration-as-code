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

package testutils

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func ToParameterMap(params []parameter.NamedParameter) map[string]parameter.Parameter {
	result := make(map[string]parameter.Parameter)

	for _, p := range params {
		result[p.Name] = p.Parameter
	}

	return result
}

func GenerateDummyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	assert.NoError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+newUUID.String(), "{}")
	return templ
}

func GenerateFaultyTemplate(t *testing.T) template.Template {
	newUUID, err := uuid.NewUUID()
	assert.NoError(t, err)
	templ := template.CreateTemplateFromString("deploy_test-"+newUUID.String(), "{")
	return templ
}
