//go:build account_integration

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
)

func createMZone(t *testing.T) {
	command := "deploy resources/mzones/manifest.yaml"
	printCommand(command)

	cli := runner.BuildCmd(afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs()))
	cli.SetArgs(strings.Split(command, " "))
	err := cli.Execute()
	require.NoError(t, err)

}

func printCommand(c string) {
	fmt.Printf("%s %s\n", "monaco", c)
}

// randomizeAndResolveConfiguration replaces %RAND% with a random value and resolves environment variables set via ${MY_ENV}
func randomizeAndResolveConfiguration(t *testing.T, fs afero.Fs, path string, randomStr string) {
	ff, err := files.FindYamlFiles(fs, path)
	require.NoError(t, err)
	for _, f := range ff {
		fileContent, err := afero.ReadFile(fs, f)
		if err != nil {
			t.Fatal(err)
		}
		contentWithEnv := os.ExpandEnv(string(fileContent))
		fileContentRandomized := strings.ReplaceAll(contentWithEnv, "%RAND%", randomStr)
		err = afero.WriteFile(fs, f, []byte(fileContentRandomized), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assertElementNotInSlice[K any](t *testing.T, sl []K, check func(el K) bool) {
	_, found := getElementInSlice(sl, check)
	assert.False(t, found)
}

func assertElementInSlice[K any](t *testing.T, sl []K, check func(el K) bool) (*K, bool) {
	e, found := getElementInSlice(sl, check)
	assert.True(t, found)
	return e, found
}

func getElementInSlice[K any](sl []K, check func(el K) bool) (*K, bool) {
	for _, e := range sl {
		if check(e) {
			return &e, true
		}
	}
	return nil, false
}
