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

package version

import (
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"github.com/spf13/cobra"
)

func GetVersionCommand() (versionCmd *cobra.Command) {
	return &cobra.Command{
		Use:     "version",
		Short:   "Prints out the version of the monaco cli",
		Example: "monaco version",
		PreRun:  cmdutils.SilenceUsageCommand(),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("monaco version " + version.MonitoringAsCode)
		},
	}
}
