/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package completion

import (
	"os"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
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

func AllAvailableApis(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	keys := []string{}
	for k := range api.NewApis() {
		keys = append(keys, k)
	}
	return keys, cobra.ShellCompDirectiveDefault
}

func EnvironmentFromEnvironmentfile(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	environmentFileName := cmd.Flag("environments").Value.String()
	environments, _ := environment.LoadEnvironmentList("", environmentFileName, afero.NewOsFs())

	keys := []string{}
	for k := range environments {
		keys = append(keys, k)
	}
	return keys, cobra.ShellCompDirectiveDefault
}

func EnvironmentFromManifest(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	manifestPath := args[0]
	manifest, _ := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           afero.NewOsFs(),
		ManifestPath: manifestPath,
	})

	keys := []string{}
	for k := range manifest.Environments {
		keys = append(keys, k)
	}
	return keys, cobra.ShellCompDirectiveDefault
}

func DeployProject(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

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
