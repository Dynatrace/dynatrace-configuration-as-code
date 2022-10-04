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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/slices"
	"github.com/spf13/pflag"
	"os"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
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

func DeployCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveFilterDirs
	}
}

func ConvertCompletion(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return EnvironmentFile()
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveFilterDirs
	}
}

func AllAvailableApis(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	value, ok := cmd.Flag("specific-api").Value.(pflag.SliceValue)
	if !ok {
		return nil, cobra.ShellCompDirectiveError
	}

	allApis := maps.Keys(api.NewApis())

	return slices.Difference(allApis, value.GetSlice()), cobra.ShellCompDirectiveDefault
}

func EnvironmentByManifestFlag(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return loadEnvironmentsFromManifest(cmd.Flag("manifest").Value.String())
}

func EnvironmentByArg0(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	return loadEnvironmentsFromManifest(args[0])
}

func loadEnvironmentsFromManifest(manifestPath string) ([]string, cobra.ShellCompDirective) {
	man, _ := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	return maps.Keys(man.Environments), cobra.ShellCompDirectiveDefault
}

func ProjectsFromManifest(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {

	manifestPath := args[0]
	mani, _ := manifest.LoadManifest(&manifest.ManifestLoaderContext{
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
