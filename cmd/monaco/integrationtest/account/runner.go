//go:build integration

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

package account

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/account/internal"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

type options struct {
	fs     afero.Fs
	suffix string
}

func RunAccountTestCase(t *testing.T, path string, name string, fn func(options)) {
	// create a new reader for the path and write all files to the in mem fs
	fs := afero.NewCopyOnWriteFs(afero.NewBasePathFs(afero.NewOsFs(), path), afero.NewMemMapFs())

	suffix := integrationtest.GenerateTestSuffix(t, name)

	// add suffix to all resource-names
	appendSuffixForWorkspace(t, fs, suffix)

	fn(options{fs, suffix})
}

func appendSuffixForWorkspace(t *testing.T, fs afero.Fs, suffix string) {
	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: "manifest.yaml",
	})

	assert.NoError(t, errors.Join(errs...))

	for _, p := range m.Projects {
		ff, err := files.FindYamlFiles(fs, p.Path)
		assert.NoError(t, err)

		for _, file := range ff {
			content := unmarshal(t, fs, file)

			var full internal.FullFile
			err := mapstructure.Decode(content, &full)
			assert.NoError(t, err)

			for i := range full.Policies {
				full.Policies[i].Name = full.Policies[i].Name + suffix
			}

			for i := range full.Groups {
				full.Groups[i].Name = full.Groups[i].Name + suffix
			}

			for i := range full.Users {
				email := full.Users[i].Email
				s := strings.Split(email, "@")
				full.Users[i].Email = s[0] + "+" + suffix + "@" + s[1]
			}

			err = mapstructure.Decode(full, &content)
			assert.NoError(t, err)

			marshal(t, fs, file, content)
		}
	}
}

func unmarshal(t *testing.T, fs afero.Fs, path string) map[string]any {
	b, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)

	var obj map[string]any
	err = yaml.Unmarshal(b, &obj)
	assert.NoError(t, err)

	return obj
}

func marshal(t *testing.T, fs afero.Fs, path string, obj any) {
	b, err := yaml.Marshal(obj)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, path, b, 0644)
	assert.NoError(t, err)
}
