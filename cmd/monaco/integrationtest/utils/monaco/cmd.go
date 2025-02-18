//go:build unit || integration || download_restore || cleanup || nightly

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

package monaco

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
)

func NewTestFs() afero.Fs { return afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs()) }

// spacesRegex finds all sequential spaces
var spacesRegex = regexp.MustCompile(`\s+`)

// RunWithFs is the entrypoint to run monaco for all integration tests.
// It requires to specify the full command (`monaco [deploy]....`) and sets up the runner.
func RunWithFs(t *testing.T, fs afero.Fs, command string) error {
	// remove multiple spaces
	c := spacesRegex.ReplaceAllString(command, " ")
	c = strings.Trim(c, " ")

	const prefix = "monaco "

	if !strings.HasPrefix(c, prefix) {
		return fmt.Errorf("command must start with '%s'", prefix)
	}
	t.Logf("Running command: %s", command)
	c = strings.TrimPrefix(c, prefix)

	args := strings.Split(c, " ")

	cmd := runner.BuildCmd(fs)
	cmd.SetArgs(args)
	return runner.RunCmd(t.Context(), cmd)
}
