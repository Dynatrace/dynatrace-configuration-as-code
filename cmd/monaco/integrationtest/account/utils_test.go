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
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
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

func randomizeConfiguration(t *testing.T, fs afero.Fs, path string, randomStr string) {
	ff, err := files.FindYamlFiles(fs, path)
	require.NoError(t, err)
	for _, f := range ff {
		fileContent, err := afero.ReadFile(fs, f)
		if err != nil {
			t.Fatal(err)
		}
		fileContentRandomized := bytes.ReplaceAll(fileContent, []byte("%RAND%"), []byte(randomStr))
		err = afero.WriteFile(fs, f, fileContentRandomized, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}
}
