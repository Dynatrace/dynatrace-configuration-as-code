//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package v2

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

func TestLoadingReport(t *testing.T) {
	targetEnvironment := "platform_env"
	reportFile := fmt.Sprintf("report%s.jsonl", time.Now().Format(trafficlogs.TrafficLogFilePrefixFormat))

	t.Setenv(environment.DeploymentReportFilename, reportFile)

	testcases := []struct {
		Name       string
		Manifest   string
		WantRecord []report.Record
	}{
		{
			Name:     "invalid extension is reported",
			Manifest: "manifest.ya",
			WantRecord: []report.Record{
				{
					Type:    report.TypeLoad,
					Time:    report.JSONTime(time.Now()),
					State:   report.StateError,
					Message: "",
					Error:   "wrong format for manifest file!",
				},
			},
		},
		{
			Name:     "not existing file",
			Manifest: "not-existing.yaml",
			WantRecord: []report.Record{
				{
					Type:    report.TypeLoad,
					Time:    report.JSONTime(time.Now()),
					State:   report.StateError,
					Message: "",
					Error:   "manifest file does not exist",
				},
			},
		},
		{
			Name:     "valid and invalid configs",
			Manifest: "test-resources/references/invalid-configs-manifest.yaml",
			WantRecord: []report.Record{
				{

					Type:   report.TypeLoad,
					Time:   report.JSONTime(time.Now()),
					State:  report.StateSuccess,
					Config: &coordinate.Coordinate{Project: "invalid-classic-with-settings", Type: "builtin:alerting.profile", ConfigId: "profile"},
				},
				{
					Type:   report.TypeLoad,
					Time:   report.JSONTime(time.Now()),
					State:  report.StateSuccess,
					Config: &coordinate.Coordinate{Project: "invalid-classic-with-settings", Type: "builtin:management-zones", ConfigId: "zone"},
				},
				{
					Type:  report.TypeLoad,
					Time:  report.JSONTime(time.Now()),
					State: report.StateError,
					Error: "cannot parse definition in `invalid-classic-with-settings\\config.yaml`: config api type (notification) configuration can only reference IDs of other config api types - parameter \"alertingProfileId\" references \"builtin:alerting.profile\" type",
				},
			},
		},
		{
			Name:     "duplicated keys error message is logged",
			Manifest: "test-resources/configs-with-duplicate-ids/manifest.yaml",
			WantRecord: []report.Record{
				{
					Type:   report.TypeLoad,
					Time:   report.JSONTime(time.Now()),
					State:  report.StateError,
					Config: &coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "profile"},
					Error:  "Config IDs need to be unique to project/type, found duplicate `project:alerting-profile:profile`",
				},
			},
		},
		{
			Name:     "missing key-user-action scope is logged",
			Manifest: "test-resources/key-user-action-without-scope/manifest.yaml",
			WantRecord: []report.Record{
				{
					Type:   report.TypeLoad,
					Time:   report.JSONTime(time.Now()),
					State:  report.StateError,
					Config: &coordinate.Coordinate{Project: "project", Type: "key-user-actions-web", ConfigId: "action"},
					Error:  "scope parameter of config of type 'key-user-actions-web' with ID 'action' needs to be a reference parameter to another web-application config",
				},
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			fs := testutils.CreateTestFileSystem()

			fmt.Println(testcase.Manifest)
			deployError := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --verbose --dry-run", testcase.Manifest, targetEnvironment))
			assert.Error(t, deployError)

			records, err := report.ReadReportFile(fs, reportFile)
			require.NoError(t, err, "report file must exists and be readable")

			for _, wantedRecord := range testcase.WantRecord {
				matcher.ContainsRecord(t, records, wantedRecord, true)
				// there should not be a success and an error record (e.g., loading worked, but it is a duplicate)
				if wantedRecord.State == report.StateError && wantedRecord.Config != nil {
					matcher.ContainsRecord(t, records, report.Record{
						Type:   report.TypeLoad,
						Time:   report.JSONTime{},
						Config: wantedRecord.Config,
						State:  report.StateSuccess,
					}, false)
				}
			}
		})
	}
}
