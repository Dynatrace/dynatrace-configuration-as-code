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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// CustomMemMapFs embeds afero.MemMapFs and overrides the MkdirAll method
type CustomMemMapFs struct {
	afero.MemMapFs
}

// MkdirAll overrides the default implementation of MkdirAll
func (fs *CustomMemMapFs) MkdirAll(path string, perm os.FileMode) error {
	if fs.DirExists(path) {
		return fmt.Errorf("directory already exists: %s", path)
	}

	return fs.MemMapFs.MkdirAll(path, perm)
}

// DirExists checks if a directory exists in the file system
func (fs *CustomMemMapFs) DirExists(path string) bool {
	fi, err := fs.Stat(path)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func TestPrepareLogFile_ReturnsErrIfParentDirectoryAlreadyExists(t *testing.T) {
	fs := &CustomMemMapFs{}
	fs.MkdirAll(".logs", 0777)
	file, err := prepareLogFile(fs)
	assert.Nil(t, file)
	assert.Error(t, err)
}

func TestPrepareLogFile_ReturnsErrIfParentDirIsReadOnly(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	file, err := prepareLogFile(fs)
	assert.Nil(t, file)
	assert.Error(t, err)
}

func TestWithFields(t *testing.T) {
	logSpy := bytes.Buffer{}
	setDefaultLogger(loggers.LogOptions{ConsoleLoggingJSON: true, LogSpy: &logSpy})
	WithFields(
		loggers.Field{"Title", "Captain"},
		loggers.Field{"Name", "Iglo"},
		loggers.CoordinateF(coordinate.Coordinate{"p1", "t1", "c1"}),
		loggers.EnvironmentF("env1")).Info("Logging with %s", "fields")

	var data map[string]interface{}
	json.Unmarshal(logSpy.Bytes(), &data)
	assert.Equal(t, "Logging with fields", data["msg"])
	assert.Equal(t, "Captain", data["Title"])
	assert.Equal(t, "Iglo", data["Name"])
	assert.Equal(t, "p1", data["coordinate"].(map[string]interface{})["Project"])
	assert.Equal(t, "t1", data["coordinate"].(map[string]interface{})["Type"])
	assert.Equal(t, "c1", data["coordinate"].(map[string]interface{})["ConfigID"])
	assert.Equal(t, "p1:t1:c1", data["coordinate"].(map[string]interface{})["Reference"])
	assert.Equal(t, "env1", data["environment"])
}

func TestFromCtx(t *testing.T) {
	logSpy := bytes.Buffer{}
	setDefaultLogger(loggers.LogOptions{ConsoleLoggingJSON: true, LogSpy: &logSpy})
	c := coordinate.Coordinate{"p1", "t1", "c1"}
	e := "e1"

	logger := FromCtx(context.WithValue(context.WithValue(context.TODO(), CtxKeyCoord{}, c), CtxKeyEnv{}, CtxValEnv{Name: e}))
	logger.Info("Hi with context")

	var data map[string]interface{}
	json.Unmarshal(logSpy.Bytes(), &data)
	assert.Equal(t, "Hi with context", data["msg"])
	assert.Equal(t, "p1", data["coordinate"].(map[string]interface{})["Project"])
	assert.Equal(t, "t1", data["coordinate"].(map[string]interface{})["Type"])
	assert.Equal(t, "c1", data["coordinate"].(map[string]interface{})["ConfigID"])
	assert.Equal(t, "p1:t1:c1", data["coordinate"].(map[string]interface{})["Reference"])
	assert.Equal(t, "e1", data["environment"])

}
