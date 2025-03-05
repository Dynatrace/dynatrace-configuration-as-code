/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package file

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
)

func TestParseFileValueParameter(t *testing.T) {
	param, err := parseFileValueParameter(parameter.ParameterParserContext{
		Fs:    afero.NewMemMapFs(),
		Value: map[string]any{"path": "something.txt"},
	})

	fileParam := param.(*FileParameter)
	require.NoError(t, err)
	assert.Equal(t, "file", param.GetType())
	assert.Equal(t, "something.txt", fileParam.Path)
}

func TestWriteFileValueParameter(t *testing.T) {
	fileParam := &FileParameter{
		Path: "myfile",
	}

	context := parameter.ParameterWriterContext{
		Parameter: fileParam,
	}

	result, err := writeFileValueParameter(context)
	require.NoError(t, err)
	assert.Equal(t, "myfile", result["path"])
}

func TestWriteFileValueParameter_WrongType(t *testing.T) {
	fileParam := envParam.New("env")

	context := parameter.ParameterWriterContext{
		Parameter: fileParam,
	}

	result, err := writeFileValueParameter(context)
	require.Nil(t, result)
	assert.IsType(t, &parameter.ParameterWriterError{}, err)
}

func TestParseFileValueParameter_MissingPath(t *testing.T) {
	param, err := parseFileValueParameter(parameter.ParameterParserContext{})

	assert.Nil(t, param)
	assert.IsType(t, parameter.ParameterParserError{}, err)
}

func TestResolveValue(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "test-content", []byte("test-content"), 0644)

	param, err := parseFileValueParameter(parameter.ParameterParserContext{
		Fs:    fs,
		Value: map[string]any{"path": "test-content"},
	})
	assert.NoError(t, err)
	assert.Len(t, param.GetReferences(), 0)

	result, err := param.ResolveValue(parameter.ResolveContext{})
	require.NoError(t, err)
	assert.Equal(t, "test-content", result)
}

func TestResolveValueWithReferences(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "test-content", []byte("test-content {{ .ref1 }} - {{ .ref2 }}"), 0644)

	param, _ := parseFileValueParameter(parameter.ParameterParserContext{
		Fs:    fs,
		Value: map[string]any{"path": "test-content", "references": []any{"ref1", "ref2"}},
	})

	assert.Len(t, param.GetReferences(), 2)

	result, err := param.ResolveValue(parameter.ResolveContext{
		ResolvedParameterValues: map[string]interface{}{
			"ref1": "ref1-resolved",
			"ref2": "ref2-resolved",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "test-content ref1-resolved - ref2-resolved", result)
}

// TestResolveValueWithReferences_ParameterMissing tests that file parameter referencing a parameter that is not found causes an error.
func TestResolveValueWithReferences_ParameterMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "test-content", []byte("test-content {{ .ref1 }} - {{ .ref2 }}"), 0644)

	param, _ := parseFileValueParameter(parameter.ParameterParserContext{
		Fs:    fs,
		Value: map[string]any{"path": "test-content", "references": []any{"ref1", "ref2"}},
	})

	assert.Len(t, param.GetReferences(), 2)

	result, err := param.ResolveValue(parameter.ResolveContext{
		ResolvedParameterValues: map[string]interface{}{
			"ref1": "ref1-resolved",
		},
	})
	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestResolveValueWithReferences_ReferenceMissing tests that using an unreferenced parameter causes an error.
func TestResolveValueWithReferences_ReferenceMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "test-content", []byte("test-content {{ .ref1 }} - {{ .ref2 }}"), 0644)

	param, _ := parseFileValueParameter(parameter.ParameterParserContext{
		Fs:    fs,
		Value: map[string]any{"path": "test-content", "references": []any{"ref1"}},
	})

	assert.Len(t, param.GetReferences(), 1)

	result, err := param.ResolveValue(parameter.ResolveContext{
		ResolvedParameterValues: map[string]interface{}{
			"ref1": "ref1-resolved",
			"ref2": "ref2-resolved",
		},
	})
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestResolveValue_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	param, _ := parseFileValueParameter(parameter.ParameterParserContext{
		Fs:    fs,
		Value: map[string]any{"path": "something"},
	})

	result, err := param.ResolveValue(parameter.ResolveContext{})
	assert.Nil(t, result)
	assert.IsType(t, parameter.ParameterResolveValueError{}, err)
}
