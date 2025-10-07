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
	"net/http"
	"os"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	monacoVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
)

func main() {
	os.Exit(executeMain(afero.NewOsFs()))
}

// executeMain is decoupled from main for testing reasons.
// it executes the command and returns the exiting code
func executeMain(fs afero.Fs) int {
	// initial logging should be verbose even if it is too early for it to go to a file
	// furthermore it should honor the desired format, such as JSON
	// full logging is set up in PreRunE method of the root command, created with runner.BuildCli
	// that is the earliest point calls to log will be also written into files and adhere to user controlled verbosity
	ctx := client.SetCustomHTTPClientInContext(context.Background())
	log.PrepareLogging(ctx, nil, true, nil, false, false)

	var versionNotification string
	if !featureflags.SkipVersionCheck.Enabled() {
		go setVersionNotificationStr(ctx, &versionNotification)
	}

	cmd, supportArchiveEnabled := runner.BuildCmd(fs)
	err := runner.RunCmd(ctx, cmd, fs, supportArchiveEnabled)
	notifyUser(versionNotification)
	if err != nil {
		return 1
	}
	return 0
}

func setVersionNotificationStr(ctx context.Context, msg *string) {
	currentVersion, err := version.ParseVersion(monacoVersion.MonitoringAsCode)
	if err != nil {
		log.With(log.ErrorAttr(err)).DebugContext(ctx, "Can't parse current monaco version: %s", err)
		return
	}

	latestVersion, err := version.GetLatestVersion(ctx, &http.Client{}, "https://api.github.com/repos/dynatrace/dynatrace-configuration-as-code/releases/latest")
	if err != nil {
		log.With(log.ErrorAttr(err)).DebugContext(ctx, "Could not perform version check: %s", err)
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
	log.Info("%s", msg)
}
