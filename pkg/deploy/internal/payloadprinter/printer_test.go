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
	"context"
	"os"
	"path/filepath"
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

func TestRecord_WritesFileWithCorrectPath(t *testing.T) {
	baseDir := t.TempDir()
	w := newWriterWithDir(baseDir)

	coord := coordinate.Coordinate{
		Project:  "myproject",
		Type:     "builtin:alerting.profile",
		ConfigId: "profile1",
	}

	w.Record(coord, "dev", `{"name":"test"}`, config.Parameters{})

	expectedPath := filepath.Join(baseDir, "dev", "myproject", "builtin-alerting.profile", "profile1.json")
	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, `{"name":"test"}`, string(content))
}

func TestRecord_RedactsEnvVarValues(t *testing.T) {
	t.Setenv("MY_SECRET", "s3cret-value")

	baseDir := t.TempDir()
	w := newWriterWithDir(baseDir)
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

	expectedPath := filepath.Join(baseDir, "dev", "proj", "settings", "cfg1.json")
	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "s3cret-value")
	assert.Contains(t, string(content), redactedValue)
	assert.Contains(t, string(content), "visible-name")
}

func TestRecord_CountsWrittenFiles(t *testing.T) {
	baseDir := t.TempDir()
	w := newWriterWithDir(baseDir)

	for i, id := range []string{"cfg1", "cfg2", "cfg3"} {
		_ = i
		w.Record(coordinate.Coordinate{Project: "p", Type: "t", ConfigId: id}, "dev", `{}`, config.Parameters{})
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	assert.Equal(t, 3, w.count)
}

func TestPayloadFilePath_SanitizesColons(t *testing.T) {
	w := newWriterWithDir("/base")
	coord := coordinate.Coordinate{
		Project:  "myproject",
		Type:     "builtin:some:type",
		ConfigId: "myid",
	}
	path := w.payloadFilePath(coord, "prod")
	assert.Contains(t, path, "builtin-some-type")
	assert.NotContains(t, path, ":")
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
