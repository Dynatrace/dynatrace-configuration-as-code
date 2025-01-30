/**
 * @license
 * Copyright 2022 Dynatrace LLC
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

package completion

import (
	"os"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/slices"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

func DeleteCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return files.YamlExtensions, cobra.ShellCompDirectiveFilterFileExt
	} else if len(args) == 1 {
		return DeleteFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveDefault
	}
}

func DeployCompletion(c *cobra.Command, args []string, s string) ([]string, cobra.ShellCompDirective) {
	return SingleArgumentManifestFileCompletion(c, args, s)
}

func SingleArgumentManifestFileCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return files.YamlExtensions, cobra.ShellCompDirectiveFilterFileExt
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveFilterDirs
	}
}

func PurgeCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return files.YamlExtensions, cobra.ShellCompDirectiveFilterFileExt
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveDefault
	}
}

func listEnvVarNames() []string {
	env := os.Environ()
	names := make([]string, len(env))
	for i, e := range env {
		names[i] = strings.Split(e, "=")[0]
	}
	return names
}

func AllAvailableApis(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	value, ok := cmd.Flag("api").Value.(pflag.SliceValue)
	if !ok {
		return nil, cobra.ShellCompDirectiveError
	}

	allApis := maps.Keys(api.NewAPIs())

	return slices.Difference(allApis, value.GetSlice()), cobra.ShellCompDirectiveDefault
}

func EnvironmentByManifestFlag(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return loadEnvironmentsFromManifest(cmd.Flag("manifest").Value.String())
}

func EnvironmentByArg0(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	return loadEnvironmentsFromManifest(args[0])
}

func loadEnvironmentsFromManifest(manifestPath string) ([]string, cobra.ShellCompDirective) {
	man, _ := manifestloader.Load(&manifestloader.Context{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	return maps.Keys(man.Environments), cobra.ShellCompDirectiveDefault
}

func AccountsByManifestFlag(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return loadAccountsFromManifest(cmd.Flag("manifest").Value.String())
}

func loadAccountsFromManifest(manifestPath string) ([]string, cobra.ShellCompDirective) {
	man, _ := manifestloader.Load(&manifestloader.Context{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	return maps.Keys(man.Accounts), cobra.ShellCompDirectiveDefault
}

func ProjectsFromManifest(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {

	manifestPath := args[0]
	mani, _ := manifestloader.Load(&manifestloader.Context{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	return maps.Keys(mani.Projects), cobra.ShellCompDirectiveDefault
}

func DependencyGraphEncodingOptions(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"default", "json"}, cobra.ShellCompDirectiveDefault
}

// YamlFile autocompletes any *yaml file, as well as directories
func YamlFile(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return files.YamlExtensions, cobra.ShellCompDirectiveFilterFileExt
}

func DeleteFile() ([]string, cobra.ShellCompDirective) {
	return deepFileSearch(".", "delete.yaml"), cobra.ShellCompDirectiveDefault
}

func deepFileSearch(root, ext string) []string {

	var allMatchingFiles []string
	err := afero.Walk(afero.NewOsFs(), root, func(path string, info os.FileInfo, err error) error {

		if info == nil {
			return nil
		}
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ext) {
			allMatchingFiles = append(allMatchingFiles, path)
		}
		return nil
	})

	if err != nil {
		return allMatchingFiles
	}

	return allMatchingFiles

}

// EnvVarName autocompletes environment variable names
func EnvVarName(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	toComplete = strings.ToUpper(toComplete)

	var results []string
	for _, k := range listEnvVarNames() {
		if strings.HasPrefix(strings.ToUpper(k), toComplete) {
			results = append(results, k)
		}
	}

	return results, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveDefault
}
