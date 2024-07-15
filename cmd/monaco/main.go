// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	monacoVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"github.com/spf13/afero"
	"net/http"
	"os"
)

func main() {
	// initial logging should be verbose even if it is too early for it to go to a file
	// furthermore it should honor the desired format, such as JSON
	// full logging is set up in PreRunE method of the root command, created with runner.BuildCli
	// that is the earliest point calls to log will be also written into files and adhere to user controlled verbosity
	log.PrepareLogging(nil, true, nil, false)

	var versionNotification string
	if !featureflags.Permanent[featureflags.SkipVersionCheck].Enabled() {
		go setVersionNotificationStr(&versionNotification)
	}

	fs := afero.NewOsFs()
	rootCmd := runner.BuildCli(fs)
	statusCode := runner.RunCmd(fs, rootCmd)
	notifyUser(versionNotification)
	os.Exit(statusCode)
}

func setVersionNotificationStr(msg *string) {
	currentVersion, err := version.ParseVersion(monacoVersion.MonitoringAsCode)
	if err != nil {
		log.WithFields(field.Error(err)).Debug("Can't parse current monaco version: %s", err)
		return
	}

	latestVersion, err := version.GetLatestVersion(context.TODO(), &http.Client{}, "https://api.github.com/repos/dynatrace/dynatrace-configuration-as-code/releases/latest")
	if err != nil {
		log.WithFields(field.Error(err)).Debug("Could not perform version check: %s", err)
		return
	}

	if currentVersion == version.UnknownVersion || latestVersion == version.UnknownVersion {
		return
	}

	if latestVersion.GreaterThan(currentVersion) {
		*msg = fmt.Sprintf("A newer version (%s) of Monaco is available! "+
			"You can download the latest release from here: https://github.com/Dynatrace/dynatrace-configuration-as-code/releases/latest\n", latestVersion)
	}
}

func notifyUser(msg string) {
	if msg == "" {
		return
	}
	log.Info(msg)
}
