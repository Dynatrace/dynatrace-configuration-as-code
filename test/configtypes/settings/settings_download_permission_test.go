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
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestPermissionDownload(t *testing.T) {
	configFolder := "testdata/download-acl"
	manifestFile := path.Join(configFolder, "manifest.yaml")

	proj := "project"
	env := "platform_env"
	appId := "app:my.dynatrace.github.connector:connection"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifestFile),
			runner.WithSuffix("permission"),
			runner.WithEnvironment(env),
		},
		func(fs afero.Fs, ctx runner.TestContext) {
			// create
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --project=%s --verbose", manifestFile, env, proj))
			require.NoError(t, err, "create: did not expect error")

			// download
			err = monaco.Run(t, fs, fmt.Sprintf("monaco download --manifest=%s --environment=%s --project=proj --output-folder=download --verbose -s %s", manifestFile, env, appId))
			require.NoError(t, err, "download: did not expect error")

			// load downloaded manifest
			mani, errs := manifestloader.Load(&manifestloader.Context{
				Fs:           fs,
				ManifestPath: "download/manifest.yaml",
				Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
			})
			assert.Empty(t, errs, "unexpected error loading manifest")

			projects, errs := project.LoadProjects(t.Context(), fs, project.ProjectLoaderContext{
				WorkingDir:      "download",
				Manifest:        mani,
				ParametersSerde: config.DefaultParameterParsers,
			}, nil)
			require.Empty(t, errs, "unexpected error loading project")
			require.Len(t, projects, 1, "expected one project")

			// find config with the correct name and check permissions
			projectAndEnvName := "proj_" + env // for manifest downloads proj + env name
			allConfigs := projects[0].Configs[projectAndEnvName]
			require.NotNil(t, allConfigs)
			configs := allConfigs[appId]

			cfg, err := findConfig(configs, ctx.Suffix)
			require.NoError(t, err, "config not found")

			assert.Equal(t, *cfg.Type.(config.SettingsType).AllUserPermission, config.WritePermission)
		})
}

// findConfig looks for a config that has a JSON payload with a name that has the given suffix.
// we don't have anything to identify the config we deployed besides the name
// This one is not written into the YAML, only the JSON
// Therefore, we have to look into every JSON file to find the correct config name
func findConfig(configs []config.Config, suffix string) (config.Config, error) {
	type contentStruct struct {
		Name string `yaml:"name"`
	}
	for _, cfg := range configs {
		content, err := cfg.Template.Content()
		if err != nil {
			return config.Config{}, err
		}
		var contentMap contentStruct
		err = yaml.Unmarshal([]byte(content), &contentMap)

		if err != nil {
			return config.Config{}, err
		}

		if strings.HasSuffix(contentMap.Name, suffix) {
			return cfg, nil
		}
	}
	return config.Config{}, errors.New("config not found")
}
