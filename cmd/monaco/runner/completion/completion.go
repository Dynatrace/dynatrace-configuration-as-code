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

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
	"github.com/spf13/pflag"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func DeleteCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile()
	} else if len(args) == 1 {
		return DeleteFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveDefault
	}
}

func MatchCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile()
	} else if len(args) == 1 {
		return MatchFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveDefault
	}
}

func DeployCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveFilterDirs
	}
}

func DownloadManifestCompletion(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile()
	} else if len(args) == 1 {
		return EnvironmentByArg0(c, args, toComplete)
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveNoFileComp
	}
}

func DownloadDirectCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		return listEnvVarNames(), cobra.ShellCompDirectiveDefault
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveNoFileComp
	}
}

func PurgeCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile()
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

func ConvertCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return EnvironmentFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveFilterDirs
	}
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
	man, _ := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	return maps.Keys(man.Environments), cobra.ShellCompDirectiveDefault
}

func ProjectsFromManifest(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {

	manifestPath := args[0]
	mani, _ := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	return maps.Keys(mani.Projects), cobra.ShellCompDirectiveDefault
}

func ManifestFile() ([]string, cobra.ShellCompDirective) {
	return files.YamlExtensions, cobra.ShellCompDirectiveFilterFileExt
}

func DeleteFile() ([]string, cobra.ShellCompDirective) {
	return deepFileSearch(".", "delete.yaml"), cobra.ShellCompDirectiveDefault
}

func MatchFile() ([]string, cobra.ShellCompDirective) {
	return deepFileSearch(".", "match.yaml"), cobra.ShellCompDirectiveDefault
}

func EnvironmentFile() ([]string, cobra.ShellCompDirective) {
	return files.YamlExtensions, cobra.ShellCompDirectiveFilterFileExt
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
