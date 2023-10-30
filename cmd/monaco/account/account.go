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
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Command(fs afero.Fs) *cobra.Command {
	command := &cobra.Command{
		Use:   "account <command>",
		Short: "Manage account management resources",
		Long: `Manage account management resources using Dynatrace Configuration as code for one or multiple accounts.

Examples:
	Deploy account management defined in a manifest:
		monaco account deploy manifest.yaml [--account <account-name-in-manifest>] [--project <project-defined-in-manifest>]
`,
	}

	command.AddCommand(deployCommand(fs))

	return command
}
