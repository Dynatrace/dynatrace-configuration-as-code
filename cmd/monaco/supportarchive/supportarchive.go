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

package supportarchive

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/zip"
)

type ctxKey struct{}

// ContextWithSupportArchive creates a new child context that has a signal that the support archive is required.
func ContextWithSupportArchive(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, struct{}{})
}

func IsEnabled(ctx context.Context) bool {
	return ctx.Value(ctxKey{}) != nil
}

func Write(fs afero.Fs) error {
	timeAnchorStr := timeutils.TimeAnchor().Format(trafficlogs.TrafficLogFilePrefixFormat)
	zipFileName := "support-archive-" + timeAnchorStr + ".zip"
	ffState, err := writeFeatureFlagStateFile(fs, timeAnchorStr)
	if err != nil {
		return err
	}
	files := []string{
		trafficlogs.RequestFilePath(),
		trafficlogs.ResponseFilePath(),
		log.LogFilePath(),
		log.ErrorFilePath(),
		ffState,
	}

	if featureflags.LogMemStats.Enabled() {
		files = append(files, log.MemStatFilePath())
	}

	if reportFilename := os.Getenv(environment.DeploymentReportFilename); len(reportFilename) > 0 {
		files = append(files, reportFilename)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	log.Info("Saving support archive to " + filepath.Join(workingDir, zipFileName))
	return zip.Create(fs, zipFileName, files, false)
}

func writeFeatureFlagStateFile(fs afero.Fs, timeAnchor string) (filename string, err error) {
	s := featureflags.StateInfo()
	path := filepath.Join(log.LogDirectory, timeAnchor+"-featureflag_state.log")
	if err := afero.WriteFile(fs, path, []byte(s), 0644); err != nil {
		return "", err
	}
	return path, nil
}
