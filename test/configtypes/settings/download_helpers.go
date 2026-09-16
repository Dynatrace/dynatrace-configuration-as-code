//go:build integration

/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package settings

import (
	"path"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

// findDownloadedConfig loads a download output folder and returns the config of the given type whose
// payload carries the unique test suffix. It fails the test if no such config exists.
func findDownloadedConfig(t *testing.T, fs afero.Fs, downloadFolder string, projectName string, configType string, suffix string) config.Config {
	cfg, found := findConfig(t, downloadedConfigsOfType(t, fs, downloadFolder, projectName, configType), suffix)
	require.Truef(t, found, "expected a %q config with suffix %q in %q", configType, suffix, downloadFolder)
	return cfg
}

// requireConfigNotDownloaded fails the test if a config of the given type carrying the unique test suffix
// exists in the download output folder.
func requireConfigNotDownloaded(t *testing.T, fs afero.Fs, downloadFolder string, projectName string, configType string, suffix string) {
	_, found := findConfig(t, downloadedConfigsOfType(t, fs, downloadFolder, projectName, configType), suffix)
	require.Falsef(t, found, "expected no %q config with suffix %q in %q", configType, suffix, downloadFolder)
}

// downloadedConfigsOfType loads the manifest and projects from a download output folder and returns the
// configs of the given type. It returns nil when the download is empty, i.e. monaco found nothing to
// download and therefore did not create the project folder.
func downloadedConfigsOfType(t *testing.T, fs afero.Fs, downloadFolder string, projectName string, configType string) []config.Config {
	exists, err := afero.DirExists(fs, path.Join(downloadFolder, projectName))
	require.NoError(t, err)
	if !exists {
		return nil
	}

	mani, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: path.Join(downloadFolder, "manifest.yaml"),
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	require.Empty(t, errs, "unexpected error loading downloaded manifest")

	projects, errs := project.LoadProjects(t.Context(), fs, project.ProjectLoaderContext{
		WorkingDir:      downloadFolder,
		Manifest:        mani,
		ParametersSerde: config.DefaultParameterParsers,
	}, nil)
	require.Empty(t, errs, "unexpected error loading downloaded projects")

	if len(projects) == 0 {
		return nil
	}
	return projects[0].Configs[projectName][configType]
}

// findConfig looks for a config whose JSON payload has a name (or displayName) with the given suffix.
// we don't have anything to identify the config we deployed besides that name
// This one is not written into the YAML, only the JSON
// Therefore, we have to look into every JSON payload to find the correct config.
func findConfig(t *testing.T, configs []config.Config, suffix string) (config.Config, bool) {
	type contentStruct struct {
		Name        string `yaml:"name"`
		DisplayName string `yaml:"displayName"`
	}
	for _, cfg := range configs {
		content, err := cfg.Template.Content()
		require.NoError(t, err)

		var contentMap contentStruct
		require.NoError(t, yaml.Unmarshal([]byte(content), &contentMap))

		if strings.HasSuffix(contentMap.Name, suffix) || strings.HasSuffix(contentMap.DisplayName, suffix) {
			return cfg, true
		}
	}
	return config.Config{}, false
}
