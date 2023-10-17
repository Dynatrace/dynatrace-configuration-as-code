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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	monacoVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"net/http"
	"os"
)

func main() {
	var versionNotification string
	go setVersionNotificationStr(&versionNotification)

	statusCode := runner.Run()
	notifyUser(versionNotification)
	os.Exit(statusCode)
}

func setVersionNotificationStr(msg *string) {
	currentVersion, err := version.ParseVersion(monacoVersion.MonitoringAsCode)
	if err != nil {
		log.Debug("Could not perform version check: %s", err)
		return
	}

	latestVersion, err := version.GetLatestVersion(context.TODO(), &http.Client{}, "https://api.github.com/repos/dynatrace/dynatrace-configuration-as-code/releases/latest")
	if err != nil {
		log.Debug("Could not perform version check: %s", err)
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
	fmt.Println(msg)
}
