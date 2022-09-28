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

func DeleteCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile(cmd, args, toComplete)
	} else if len(args) == 1 {
		return DeleteFile(cmd, args, toComplete)
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveDefault
	}
}

func DeployCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ManifestFile(cmd, args, toComplete)
	} else {
		return make([]string, 0), cobra.ShellCompDirectiveFilterDirs
	}
}

func ConvertCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return EnvironmentFile(cmd, args, toComplete)
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

func ProjectsFromManifest(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	manifestPath := args[0]
	manifest, _ := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	keys := []string{}
	for k := range manifest.Projects {
		keys = append(keys, k)
	}
	return keys, cobra.ShellCompDirectiveDefault
}

func ManifestFile(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"yaml"}, cobra.ShellCompDirectiveFilterFileExt
}

func DeleteFile(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return deepFileSeach(".", "delete.yaml"), cobra.ShellCompDirectiveDefault
}

func EnvironmentFile(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"yaml"}, cobra.ShellCompDirectiveFilterFileExt
}

func deepFileSeach(root, ext string) []string {

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
