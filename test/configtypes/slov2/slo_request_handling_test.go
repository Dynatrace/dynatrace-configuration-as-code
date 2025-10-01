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

package slov2

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	runner2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestIntegrationSloV1AndSloV2(t *testing.T) {
	configFolder := "testdata/slo-v1-and-slo-v2/"
	manifest := configFolder + "manifest.yaml"

	t.Run("slo-v1 to slo-v2", func(t *testing.T) {
		runner2.Run(t, configFolder,
			runner2.Options{
				runner2.WithSuffix("slo-v1-to-slo-v2"),
			},
			func(fs afero.Fs, _ runner2.TestContext) {
				logOutput := strings.Builder{}
				cmd, _ := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
				cmd.SetArgs([]string{"deploy", "--verbose", manifest, "--continue-on-error", "--project", "slo-v1-to-slo-v2"})
				err := cmd.Execute()

				assert.ErrorContains(t, err, "2 deployment errors occurred")

				runLog := strings.ToLower(logOutput.String())

				assert.Contains(t, runLog, "tried to deploy an slo-v1 configuration to slo-v2")
			})
	})

	t.Run("slo-v2 to slo-v1", func(t *testing.T) {
		runner2.Run(t, configFolder,
			runner2.Options{
				runner2.WithSuffix("slo-v2-to-slo-v1"),
			},
			func(fs afero.Fs, _ runner2.TestContext) {
				logOutput := strings.Builder{}
				cmd, _ := runner.BuildCmdWithLogSpy(testutils.CreateTestFileSystem(), &logOutput)
				cmd.SetArgs([]string{"deploy", "--verbose", manifest, "--continue-on-error", "--project", "slo-v2-to-slo-v1"})
				err := cmd.Execute()

				assert.ErrorContains(t, err, "2 deployment errors occurred")

				runLog := strings.ToLower(logOutput.String())

				assert.Contains(t, runLog, "tried to deploy an slo-v2 configuration to slo-v1")
			})
	})
}
