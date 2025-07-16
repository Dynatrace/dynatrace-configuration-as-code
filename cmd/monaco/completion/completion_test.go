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

package completion_test

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
)

func TestEnvironmentByArg0(t *testing.T) {
	manifestPath := "testdata/manifest.yaml"
	envs, _ := completion.EnvironmentByArg0(nil, []string{manifestPath}, "")
	assert.ElementsMatch(t, envs, []string{"env1", "env2"})
}

func TestAccountsByManifestFlag(t *testing.T) {
	manifestPath := "testdata/manifest.yaml"
	cmd := getAccountDeployCommand(account.Command(afero.NewOsFs()).Commands())
	require.NotNil(t, cmd)
	err := cmd.Flag("manifest").Value.Set(manifestPath)
	require.NoError(t, err)

	envs, _ := completion.AccountsByManifestFlag(cmd, []string{}, "")
	assert.ElementsMatch(t, envs, []string{"account1", "account2"})
}

func getAccountDeployCommand(commands []*cobra.Command) (command *cobra.Command) {
	for _, cmd := range commands {
		if strings.Contains(cmd.Name(), "deploy") {
			return cmd
		}
	}
	return nil
}
