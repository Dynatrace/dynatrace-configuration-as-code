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

package logging

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

// readReport reads and returns all records in the specified report file and asserts that this succeeded.
func readReport(t *testing.T, fs afero.Fs, path string) []report.Record {
	t.Helper()

	records, err := report.ReadReportFile(fs, path)
	require.NoError(t, err, "file must exists and be readable")

	require.NotEmpty(t, records)

	return records
}

// AssertReport reads a report and asserts that it either indicates a successful or failed deployment depending on the value of succeed.
func AssertReport(t *testing.T, fs afero.Fs, path string, succeed bool) {
	t.Helper()

	records := readReport(t, fs, path)
	matcher.ContainsInfoRecord(t, records, "Monaco version")
	matcher.ContainsInfoRecord(t, records, "Deployment finished")
	matcher.ContainsInfoRecord(t, records, "Report finished")

	if succeed {
		for index, r := range records {
			assert.Containsf(t, []report.RecordState{report.StateSuccess, report.StateExcluded, report.StateSkipped, report.StateInfo}, r.State, "config at %d is with status %s", index, r.State)
		}
	}

	if !succeed {
		haveErrorRecord := false
		for _, r := range records {
			if "ERROR" == r.State {
				haveErrorRecord = true
				break
			}
		}
		if !haveErrorRecord {
			assert.Fail(t, "there is no record with ERROR status")
		}
	}
}
