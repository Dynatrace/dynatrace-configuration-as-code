//go:build integration

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

package skip

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestSkip(t *testing.T) {

	projectFolder := "testdata/skip-test/"
	manifest := projectFolder + "manifest.yaml"

	type given struct {
		environment  string
		skipVarValue bool
	}

	type want struct {
		deployedConfigIDs []string
		skippedConfigIDs  []string
	}

	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			"without env override or skip via env_var",
			given{
				environment:  "environment1",
				skipVarValue: false,
			},
			want{
				deployedConfigIDs: []string{"Basic Tag", "Env Var Skipped Tag"},
				skippedConfigIDs:  []string{"Skipped Value Tag", "Environment Override Deployed Tag"},
			},
		},
		{
			"with env override",
			given{
				environment:  "environment2",
				skipVarValue: false,
			},
			want{
				deployedConfigIDs: []string{"Basic Tag", "Env Var Skipped Tag", "Environment Override Deployed Tag"},
				skippedConfigIDs:  []string{"Skipped Value Tag"},
			},
		},
		{
			"with skip via env var",
			given{
				environment:  "environment1",
				skipVarValue: true,
			},
			want{
				deployedConfigIDs: []string{"Basic Tag"},
				skippedConfigIDs:  []string{"Skipped Value Tag", "Environment Override Deployed Tag", "Env Var Skipped Tag"},
			},
		},
		{
			"with env override and skip via env var",
			given{
				environment:  "environment2",
				skipVarValue: true,
			},
			want{
				deployedConfigIDs: []string{"Basic Tag", "Environment Override Deployed Tag"},
				skippedConfigIDs:  []string{"Skipped Value Tag", "Env Var Skipped Tag"},
			},
		},
	}

	loadedManifest := runner.LoadManifest(t, afero.OsFs{}, manifest, "")
	clients := make(map[string]client.SettingsClient)

	for name, def := range loadedManifest.Environments.SelectedEnvironments {
		set := runner.CreateDynatraceClients(t, def)
		clients[name] = set.SettingsClient
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			runner.Run(t, projectFolder,
				runner.Options{
					runner.WithManifestPath(manifest),
					runner.WithSuffix("SkipTest"),
					runner.WithEnvironment(tt.given.environment),
				},
				func(fs afero.Fs, tc runner.TestContext) {

					testCaseVar := "SKIPPED_VAR_" + tc.Suffix
					t.Setenv(testCaseVar, strconv.FormatBool(tt.given.skipVarValue))

					err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
					assert.NoError(t, err)

					client, ok := clients[tt.given.environment]
					assert.True(t, ok, "expected to find client for environment ", tt.given.environment)

					log.Info("Asserting configs were deployed: %v", tt.want.deployedConfigIDs)
					for _, id := range tt.want.deployedConfigIDs {
						assertTestConfig(t, tc, client, tt.given.environment, id, true)
					}

					log.Info("Asserting configs were skipped: %v", tt.want.skippedConfigIDs)
					for _, id := range tt.want.skippedConfigIDs {
						assertTestConfig(t, tc, client, tt.given.environment, id, false)
					}

				})
		})
	}
}

func assertTestConfig(t *testing.T, tc runner.TestContext, client client.SettingsClient, envName string, configID string, shouldExist bool) {
	configID = fmt.Sprintf("%s_%s", configID, tc.Suffix)

	assert2.AssertSetting(t, client, config.SettingsType{SchemaId: "builtin:tags.auto-tagging"}, envName, shouldExist, config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  "project",
			Type:     "builtin:tags.auto-tagging",
			ConfigId: configID,
		},
	})
}
