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

package account_test

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/writer"
)

func TestLoadAndReWriteAccountResources(t *testing.T) {
	testResources := "loader/testdata/multi"
	fs := afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())

	// LOAD RESOURCES FROM DISK
	resources, err := loader.Load(fs, testResources)
	assert.NoError(t, err)

	assert.NotEmpty(t, resources.Groups)
	assert.NotEmpty(t, resources.Policies)
	assert.NotEmpty(t, resources.Users)

	// WRITE IN-MEMORY REPRESENTATION TO DISK
	c := writer.Context{
		Fs:            fs,
		OutputFolder:  "test-folder",
		ProjectFolder: "test-project",
	}
	err = writer.Write(c, *resources)
	assert.NoError(t, err)

	// ASSERT FILES WRITTEN AS EXPECTED
	expectedOutputFolder := filepath.Join(c.OutputFolder, c.ProjectFolder)
	assertFileExists(t, c.Fs, filepath.Join(expectedOutputFolder, "users.yaml"))
	assertFileExists(t, c.Fs, filepath.Join(expectedOutputFolder, "groups.yaml"))
	assertFileExists(t, c.Fs, filepath.Join(expectedOutputFolder, "policies.yaml"))

	// ASSERT WRITTEN FILES MATCH ORIGINALS AFTER LOADING THEM FROM DISK
	writtenResources, err := loader.Load(fs, expectedOutputFolder)
	assert.NoError(t, err)
	assert.Equal(t, resources.Groups, writtenResources.Groups)
	assert.Equal(t, resources.Policies, writtenResources.Policies)
	assert.Equal(t, resources.Users, writtenResources.Users)
}

func assertFileExists(t *testing.T, fs afero.Fs, path string) {
	exists, err := afero.Exists(fs, path)
	assert.NoError(t, err)
	assert.True(t, exists, "expected file to exist %v", path)
}
