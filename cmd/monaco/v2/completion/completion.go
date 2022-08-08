/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package completion

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/spf13/cobra"
)

func AllAvailableApis(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	keys := []string{}
	for k := range api.NewApis() {
		keys = append(keys, k)
	}
	return keys, cobra.ShellCompDirectiveDefault
}
