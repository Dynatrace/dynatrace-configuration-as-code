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

//go:build unit

package payloadprinter

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

func TestGetWriterFromContext_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, GetWriterFromContext(ctx))
}

func TestGetWriterFromContext_ReturnsWriter(t *testing.T) {
	ctx := NewContextWithWriter(context.Background())
	assert.NotNil(t, GetWriterFromContext(ctx))
}

func TestWriteEntry_BasicOutput(t *testing.T) {
	var buf bytes.Buffer
	coord := coordinate.Coordinate{
		Project:  "myproject",
		Type:     "builtin:alerting.profile",
		ConfigId: "profile1",
	}

	writeEntry(&buf, coord, "dev", `{"name":"test"}`)

	output := buf.String()
	assert.Contains(t, output, "--- Rendered payload:")
	assert.Contains(t, output, "myproject:builtin:alerting.profile:profile1")
	assert.Contains(t, output, "environment: dev")
	assert.Contains(t, output, `{"name":"test"}`)
	assert.Contains(t, output, "---")
}

func TestRecord_RedactsEnvVarValues(t *testing.T) {
	t.Setenv("MY_SECRET", "s3cret-value")

	w := &Writer{}
	coord := coordinate.Coordinate{
		Project:  "proj",
		Type:     "settings",
		ConfigId: "cfg1",
	}

	params := config.Parameters{
		"token": envParam.New("MY_SECRET"),
		"name":  &valueParam.ValueParameter{Value: "visible-name"},
	}

	w.Record(coord, "dev", `{"name":"visible-name","token":"s3cret-value"}`, params)

	require.Len(t, w.entries, 1)
	assert.NotContains(t, w.entries[0].rendered, "s3cret-value")
	assert.Contains(t, w.entries[0].rendered, redactedValue)
	assert.Contains(t, w.entries[0].rendered, "visible-name")
}

func TestRedactSecrets_NoEnvParams(t *testing.T) {
	params := config.Parameters{
		"name": &valueParam.ValueParameter{Value: "test"},
	}
	result := redactSecrets(`{"name":"test"}`, params)
	assert.Equal(t, `{"name":"test"}`, result)
}

func TestRedactSecrets_EmptyEnvVar(t *testing.T) {
	t.Setenv("EMPTY_VAR", "")

	params := config.Parameters{
		"token": envParam.New("EMPTY_VAR"),
	}
	input := `{"token":""}`
	result := redactSecrets(input, params)
	// Empty values should not be redacted (would replace all empty strings)
	assert.Equal(t, input, result)
}

func TestRedactSecrets_UnsetEnvVar(t *testing.T) {
	params := config.Parameters{
		"token": envParam.New("DEFINITELY_NOT_SET_VAR_12345"),
	}
	input := `{"token":"fallback"}`
	result := redactSecrets(input, params)
	// Unresolvable env vars are skipped gracefully
	assert.Equal(t, input, result)
}

func TestRedactSecrets_MultipleEnvParams(t *testing.T) {
	t.Setenv("SECRET_A", "alpha")
	t.Setenv("SECRET_B", "beta")

	params := config.Parameters{
		"a": envParam.New("SECRET_A"),
		"b": envParam.New("SECRET_B"),
	}
	result := redactSecrets(`{"a":"alpha","b":"beta"}`, params)
	assert.NotContains(t, result, "alpha")
	assert.NotContains(t, result, "beta")
	assert.Contains(t, result, redactedValue)
}
