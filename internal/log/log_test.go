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

package log

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPrepareLogFile_WorksIfParentDirectoryAlreadyExists(t *testing.T) {
	fs := testutils.TempFs(t)
	err := fs.MkdirAll(".logs", 0777)
	assert.NoError(t, err)
	file, errFile, err := prepareLogFiles(fs)
	assert.NotNil(t, file)
	assert.NotNil(t, errFile)
	assert.NoError(t, err)
}

func TestPrepareLogFile_ReturnsErrIfParentDirIsReadOnly(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	file, errFile, err := prepareLogFiles(fs)
	assert.Nil(t, file)
	assert.Nil(t, errFile)
	assert.Error(t, err)
}

func TestWithFields(t *testing.T) {
	logSpy := bytes.Buffer{}
	setDefaultLogger(loggers.LogOptions{JSONLogging: true, LogSpy: &logSpy})
	WithFields(
		field.Field{"Title", "Captain"},
		field.Field{"Name", "Iglo"},
		field.Coordinate(coordinate.Coordinate{"p1", "t1", "c1"}),
		field.Environment("env1", "group")).Info("Logging with %s", "fields")

	var data map[string]interface{}
	err := json.Unmarshal(logSpy.Bytes(), &data)
	assert.NoError(t, err)
	assert.Equal(t, "Logging with fields", data["msg"])
	assert.Equal(t, "Captain", data["Title"])
	assert.Equal(t, "Iglo", data["Name"])
	assert.Equal(t, "p1", data["coordinate"].(map[string]interface{})["project"])
	assert.Equal(t, "t1", data["coordinate"].(map[string]interface{})["type"])
	assert.Equal(t, "c1", data["coordinate"].(map[string]interface{})["configID"])
	assert.Equal(t, "p1:t1:c1", data["coordinate"].(map[string]interface{})["reference"])
	assert.Equal(t, "env1", data["environment"].(map[string]interface{})["name"])
	assert.Equal(t, "group", data["environment"].(map[string]interface{})["group"])
}

func TestFromCtx(t *testing.T) {
	logSpy := bytes.Buffer{}
	setDefaultLogger(loggers.LogOptions{JSONLogging: true, LogSpy: &logSpy})
	c := coordinate.Coordinate{"p1", "t1", "c1"}
	e := "e1"
	g := "g"

	logger := WithCtxFields(context.WithValue(context.WithValue(context.TODO(), CtxKeyCoord{}, c), CtxKeyEnv{}, CtxValEnv{Name: e, Group: g}))
	logger.Info("Hi with context")

	var data map[string]interface{}
	err := json.Unmarshal(logSpy.Bytes(), &data)
	assert.NoError(t, err)
	assert.Equal(t, "Hi with context", data["msg"])
	assert.Equal(t, "p1", data["coordinate"].(map[string]interface{})["project"])
	assert.Equal(t, "t1", data["coordinate"].(map[string]interface{})["type"])
	assert.Equal(t, "c1", data["coordinate"].(map[string]interface{})["configID"])
	assert.Equal(t, "p1:t1:c1", data["coordinate"].(map[string]interface{})["reference"])
	assert.Equal(t, "e1", data["environment"].(map[string]interface{})["name"])
	assert.Equal(t, "g", data["environment"].(map[string]interface{})["group"])

}
