//go:build account_integration

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
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

type options struct {
	fs          afero.Fs
	accountUUID string
	accountName string
	randomStr   string
	randomize   func(string) string
}

func RunAccountTestCase(t *testing.T, path string, manifestFileName string, name string, fn func(map[account.AccountInfo]*accounts.Client, options)) {
	fs := afero.NewCopyOnWriteFs(afero.NewBasePathFs(afero.NewOsFs(), path), afero.NewMemMapFs())
	randomStr := randomizeYAMLResources(t, fs, name)
	accClients := createAccountClientsFromManifest(t, fs, manifestFileName)

	accountUUID, found := os.LookupEnv("ACCOUNT_UUID")
	if !found {
		t.Error("ACCOUNT_UUID environment variable must be set")
	}
	fn(accClients, options{accountName: "monaco-test-account", accountUUID: accountUUID, fs: fs, randomStr: randomStr, randomize: randomizeFn(randomStr)})
}

// createAccountClientsFromManifest creates a map of accountInfo --> account client for a given manifest
func createAccountClientsFromManifest(t *testing.T, fs afero.Fs, manifestFileName string) map[account.AccountInfo]*accounts.Client {
	m, errs := manifestloader.Load(&manifestloader.Context{Fs: fs, ManifestPath: manifestFileName, Opts: manifestloader.Options{RequireAccounts: true}})
	require.NoError(t, errors.Join(errs...))
	accClients, err := dynatrace.CreateAccountClients(t.Context(), m.Accounts)
	require.NoError(t, err)
	return accClients
}

// randomizeYAMLResources loops over each *.yaml file, replaces %RAND% with a random string and returns the random string
// that was used
func randomizeYAMLResources(t *testing.T, fs afero.Fs, name string) string {
	randStr := runner.GenerateTestSuffix(t, name)
	ff, err := files.FindYamlFiles(fs, ".")
	require.NoError(t, err)
	for _, file := range ff {
		fileContent, err := afero.ReadFile(fs, file)
		require.NoError(t, err)
		fileContentRandomized := randomizeFn(randStr)(string(fileContent))
		err = afero.WriteFile(fs, file, []byte(fileContentRandomized), 0644)
		require.NoError(t, err)
	}
	return randStr
}

func randomizeFn(suffix string) func(in string) string {
	return func(in string) string {
		return strings.ReplaceAll(in, "%RAND%", suffix)
	}
}
