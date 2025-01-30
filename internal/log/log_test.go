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
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

func TestPrepareLogging(t *testing.T) {
	type pathsWithPermission map[string]os.FileMode
	tests := []struct {
		name           string
		givenFolders   pathsWithPermission
		givenFiles     pathsWithPermission
		wantLogFile    bool
		wantErrLogFile bool
		wantError      bool
	}{
		{
			name:           "creates files if folder does not exists",
			wantLogFile:    true,
			wantErrLogFile: true,
		},
		{
			name:           "creates files if folder exists",
			givenFolders:   pathsWithPermission{".logs/": 0777},
			wantLogFile:    true,
			wantErrLogFile: true,
		},
		{
			name:           "fails if log folder exists as file",
			givenFiles:     pathsWithPermission{".logs": 0777},
			wantLogFile:    false,
			wantErrLogFile: false,
			wantError:      true,
		},
		{
			name:           "fails if log file creation fails",
			givenFolders:   pathsWithPermission{".logs/": 0777},
			givenFiles:     pathsWithPermission{LogFilePath(): 0000}, // logFile exists and can't be accessed
			wantLogFile:    false,
			wantErrLogFile: false,
			wantError:      true,
		},
		{
			name:           "creates log file even though err file creation fails",
			givenFolders:   pathsWithPermission{".logs/": 0777},
			givenFiles:     pathsWithPermission{ErrorFilePath(): 0000}, // errFile exists and can't be accessed
			wantLogFile:    true,
			wantErrLogFile: false,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup test fs with given files
			fs := testutils.TempFs(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for folder, perm := range tt.givenFolders {
				err := fs.MkdirAll(folder, perm)
				assert.NoError(t, err)
			}
			for file, perm := range tt.givenFiles {
				f, err := fs.Create(file)
				assert.NoError(t, err)
				assert.NoError(t, f.Close())

				err = fs.Chmod(file, perm)
				assert.NoError(t, err)
			}

			file, errFile, err := prepareLogFiles(ctx, fs)

			if tt.wantLogFile {
				assert.NotNil(t, file)
				assert.NoError(t, file.Close())
			} else {
				assert.Nil(t, file)
			}
			if tt.wantErrLogFile {
				assert.NotNil(t, errFile)
				assert.NoError(t, errFile.Close())
			} else {
				assert.Nil(t, errFile)
			}
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPrepareLogFile_ReturnsErrIfParentDirIsReadOnly is separate from the other log file tests, as there is no portable
// way of handling folder permissions on a real filesystem. (Go has no way to set Windows folder permissions:https://github.com/golang/go/issues/35042)
// Thus it tests the "can't write even the .logs/ dir" case by setting the whole afero FS to read only... In the real world,
// this would happen if the Windows folder is marked read only, or POSIX permissions don't allow writing to it.
func TestPrepareLogFile_ReturnsErrIfParentDirIsReadOnly(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	file, errFile, err := prepareLogFiles(ctx, fs)
	assert.Nil(t, file)
	assert.Nil(t, errFile)
	assert.Error(t, err)
}

func TestWithFields(t *testing.T) {
	logSpy := bytes.Buffer{}

	t.Setenv("MONACO_LOG_FORMAT", "json")
	PrepareLogging(context.TODO(), afero.NewOsFs(), false, &logSpy, false)

	WithFields(
		field.Field{Key: "Title", Value: "Captain"},
		field.Field{Key: "Name", Value: "Iglo"},
		field.Coordinate(coordinate.Coordinate{Project: "p1", Type: "t1", ConfigId: "c1"}),
		field.Environment("env1", "group")).Info(fmt.Sprintf("Logging with %s", "fields"))

	var data map[string]interface{}
	err := json.Unmarshal(logSpy.Bytes(), &data)
	require.NoError(t, err)
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

	t.Setenv("MONACO_LOG_FORMAT", "json")
	PrepareLogging(context.TODO(), afero.NewOsFs(), false, &logSpy, false)

	c := coordinate.Coordinate{Project: "p1", Type: "t1", ConfigId: "c1"}
	e := "e1"
	g := "g"

	logger := WithCtxFields(context.WithValue(context.WithValue(context.TODO(), CtxKeyCoord{}, c), CtxKeyEnv{}, CtxValEnv{Name: e, Group: g}))
	logger.Info("Hi with context")

	var data map[string]interface{}
	err := json.Unmarshal(logSpy.Bytes(), &data)
	require.NoError(t, err)
	assert.Equal(t, "Hi with context", data["msg"])
	assert.Equal(t, "p1", data["coordinate"].(map[string]interface{})["project"])
	assert.Equal(t, "t1", data["coordinate"].(map[string]interface{})["type"])
	assert.Equal(t, "c1", data["coordinate"].(map[string]interface{})["configID"])
	assert.Equal(t, "p1:t1:c1", data["coordinate"].(map[string]interface{})["reference"])
	assert.Equal(t, "e1", data["environment"].(map[string]interface{})["name"])
	assert.Equal(t, "g", data["environment"].(map[string]interface{})["group"])

}
