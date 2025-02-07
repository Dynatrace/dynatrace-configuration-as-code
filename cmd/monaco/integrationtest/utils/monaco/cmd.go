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
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
)

// The Monaco is the entry point for integration tests. It accepts a command in a way it would be called via a CLI. `monaco` keyword can be omitted from the command.
// To execute command invoke Run() method
//
// monaco.Monaco("monaco download manifest.yaml --enviroment=my_env").Run()

func Run(command string) error {
	return RunWithFs(newFs(), command)
}

func Runf(command string, args ...any) error {
	return RunWithFSf(newFs(), command, args...)
}

func newFs() afero.Fs { return afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs()) }

func RunWithFSf(fs afero.Fs, command string, args ...any) error {
	return RunWithFs(fs, fmt.Sprintf(command, args...))
}

func RunWithFs(fs afero.Fs, command string) error {
	// remove multiple spaces
	c := regexp.MustCompile(`\s+`).ReplaceAllString(command, " ")
	c = strings.Trim(c, " ")

	if !strings.HasPrefix(c, "monaco ") {
		panic("Command must start with 'monaco'")
	}
	fmt.Println(c)
	c = strings.TrimPrefix(c, "monaco ")

	args := strings.Split(c, " ")

	cmd := runner.BuildCmd(fs)
	cmd.SetArgs(args)
	return runner.RunCmd(context.TODO(), fs, cmd)
}
