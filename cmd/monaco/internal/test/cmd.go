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

package test

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
	"regexp"
	"strings"
)

type Cmd struct {
	cmd string
	fs  afero.Fs
}

func Monacof(cmd string, args ...any) *Cmd {
	cmd = fmt.Sprintf(cmd, args...)

	cmd = regexp.MustCompile(`\s+`).ReplaceAllString(cmd, " ")
	cmd = strings.Trim(cmd, " ")
	cmd = strings.TrimPrefix(cmd, "monaco ")

	return &Cmd{cmd: cmd}
}

func (cmd *Cmd) WithFs(fs afero.Fs) *Cmd {
	cmd.fs = fs
	return cmd
}

func (cmd *Cmd) Run() (afero.Fs, error) {
	fs := cmd.fs
	if fs == nil {
		fs = afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())
	}
	fmt.Println(cmd)

	c := strings.TrimPrefix(cmd.String(), "monaco ")
	args := strings.Split(c, " ")

	cli := runner.BuildCli(fs)
	cli.SetArgs(args)

	return fs, cli.Execute()
}

func (cmd *Cmd) String() string {
	c := fmt.Sprintf("%s %s", "monaco", cmd.cmd)
	c = regexp.MustCompile(`\s+`).ReplaceAllString(c, " ")
	return c
}
