//go:build unit

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

package log_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
)

// TestPrepareLogging_LogFormat tests that logging is in plain, JSON, or color format if specified via MONACO_LOG_FORMAT environment variable.
func TestPrepareLogging_LogFormat(t *testing.T) {
	const resetColorControlSequence = "\x1b[0m"

	tests := []struct {
		name         string
		logFormatEnv string
		wantJSON     bool
		wantColor    bool
	}{
		{
			name:         "no log format value does not use JSON or color",
			logFormatEnv: "",
			wantJSON:     false,
			wantColor:    false,
		},
		{
			name:         "log format JSON uses JSON and no color",
			logFormatEnv: "json",
			wantJSON:     true,
			wantColor:    false,
		},

		{
			name:         "color does not use JSON but does use color",
			logFormatEnv: "color",
			wantJSON:     false,
			wantColor:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.logFormatEnv) > 0 {
				t.Setenv("MONACO_LOG_FORMAT", tt.logFormatEnv)
			}

			builder := strings.Builder{}
			log.PrepareLogging(t.Context(), nil, true, &builder, false, false)
			log.Debug("hello")

			o := map[string]any{}
			err := json.Unmarshal([]byte(builder.String()), &o)
			if tt.wantJSON {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			if tt.wantColor {
				assert.Contains(t, builder.String(), resetColorControlSequence, "wanted color but log entry does not contain default color sequence")
			} else {
				assert.NotContains(t, builder.String(), resetColorControlSequence, "did not want color but log entry contains default color sequence")
			}
		})
	}
}

// TestPrepareLogging_SupportsUTC tests that logs are written using time entries in UTC if explicitly enabled.
// The case where the feature is disabled is not tested as the logger may still log in UTC due to the testing environment setup.
func TestPrepareLogging_SupportsUTC(t *testing.T) {
	t.Setenv("MONACO_LOG_FORMAT", "json")
	t.Setenv("MONACO_LOG_TIME", "utc")

	builder := strings.Builder{}
	log.PrepareLogging(t.Context(), nil, true, &builder, false, false)
	log.Debug("hello")

	// assert that entry is JSON and can be unmarshaled
	o := map[string]any{}
	err := json.Unmarshal([]byte(builder.String()), &o)
	require.NoError(t, err)

	// assert that the entry has a time field that ends with Z, an indication of UTC
	entrytime, containsTime := o["time"].(string)
	assert.True(t, containsTime)
	assert.True(t, strings.HasSuffix(entrytime, "Z"))
}

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
			givenFiles:     pathsWithPermission{log.LogFilePath(): 0000}, // logFile exists and can't be accessed
			wantLogFile:    false,
			wantErrLogFile: false,
			wantError:      true,
		},
		{
			name:           "creates log file even though err file creation fails",
			givenFolders:   pathsWithPermission{".logs/": 0777},
			givenFiles:     pathsWithPermission{log.ErrorFilePath(): 0000}, // errFile exists and can't be accessed
			wantLogFile:    true,
			wantErrLogFile: false,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup test fs with given files
			fs := testutils.TempFs(t)

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

			file, errFile, err := log.PrepareLogFiles(t.Context(), fs, false)

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

// TestPrepareLogFiles_ReturnsErrIfParentDirIsReadOnly is separate from the other log file tests, as there is no portable
// way of handling folder permissions on a real filesystem. (Go has no way to set Windows folder permissions:https://github.com/golang/go/issues/35042)
// Thus it tests the "can't write even the .logs/ dir" case by setting the whole afero FS to read only... In the real world,
// this would happen if the Windows folder is marked read only, or POSIX permissions don't allow writing to it.
func TestPrepareLogFile_ReturnsErrIfParentDirIsReadOnly(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	file, errFile, err := log.PrepareLogFiles(t.Context(), fs, false)
	assert.Nil(t, file)
	assert.Nil(t, errFile)
	assert.Error(t, err)
}

// TestDefaultLoggerFunctions tests that the default logger functions work as expected.
func TestDefaultLoggerFunctions(t *testing.T) {

	builder := strings.Builder{}
	log.PrepareLogging(t.Context(), nil, true, &builder, false, false)

	t.Run("debug", func(t *testing.T) {
		builder.Reset()
		log.Debug("code %s reached", "here")

		assert.Contains(t, builder.String(), "code here reached")
		assert.Contains(t, strings.ToLower(builder.String()), "debug")
	})

	t.Run("info", func(t *testing.T) {
		builder.Reset()
		log.Info("code %s reached", "here")

		assert.Contains(t, builder.String(), "code here reached")
		assert.Contains(t, strings.ToLower(builder.String()), "info")
	})

	t.Run("warn", func(t *testing.T) {
		builder.Reset()
		log.Warn("code %s reached", "here")

		assert.Contains(t, builder.String(), "code here reached")
		assert.Contains(t, strings.ToLower(builder.String()), "warn")
	})

	t.Run("error", func(t *testing.T) {
		builder.Reset()
		log.Error("code %s reached", "here")

		assert.Contains(t, builder.String(), "code here reached")
		assert.Contains(t, strings.ToLower(builder.String()), "error")
	})
}
