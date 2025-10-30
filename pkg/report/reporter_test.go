//go:build unit

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package report_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils/matcher"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

// TestReporter_ContextWithNoReporterDiscards tests that the Recorder obtained from an context without the default one discards.
func TestReporter_ContextWithNoReporterDiscards(t *testing.T) {
	reporter := report.GetReporterFromContextOrDiscard(t.Context())
	require.NotNil(t, reporter)

	reporter.ReportSuccessfulDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard"}, "object-id", nil)
	reporter.Stop()
	assert.Empty(t, reporter.GetSummary(), "discarding Reporter should not return a summary")
}

// TestReporter_ContextWithDefaultReporterCollectsEvents tests that the Reporter obtained from an context with the default one attached collects events.
func TestReporter_ContextWithDefaultReporterCollectsEvents(t *testing.T) {

	reportFilename := "test_report.jsonl"
	fs := testutils.TempFs(t)

	testTime := time.Unix(time.Now().Unix(), 0).UTC()

	r := report.NewDefaultReporterWithClockFunc(fs, reportFilename, func() time.Time { return testTime })
	ctx := report.NewContextWithReporter(t.Context(), r)

	reporter := report.GetReporterFromContextOrDiscard(ctx)
	require.NotNil(t, reporter)

	reporter.ReportInfo("startup")
	reporter.ReportLoading(report.StateSuccess, nil, "", &coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"})
	reporter.ReportLoading(report.StateError, errors.New("my-error"), "my-message", nil)
	reporter.ReportCaching(report.StateWarn, "my-warning")
	reporter.ReportSuccessfulDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, "object-id", nil)
	reporter.ReportFailedDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard2"}, []report.Detail{{Type: report.DetailTypeError, Message: "error"}}, errors.New("an error"))
	reporter.ReportSkippedDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard3"}, []report.Detail{{Type: report.DetailTypeInfo, Message: "skipped"}})
	reporter.ReportExcludedDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard4"}, []report.Detail{{Type: report.DetailTypeInfo, Message: "excluded"}})

	reporter.Stop()

	exists, err := afero.Exists(fs, reportFilename)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NotEmpty(t, reporter.GetSummary(), "summary should not be empty")

	records, err := report.ReadReportFile(fs, reportFilename)
	require.NoError(t, err)

	require.Len(t, records, 8)
	anError := "An error"

	matcher.ContainsRecord(t, records, report.Record{Type: "INFO", Time: report.JSONTime(testTime), State: "INFO", Message: "startup"}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "LOAD", Time: report.JSONTime(testTime), Config: &coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, State: "SUCCESS"}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "LOAD", Time: report.JSONTime(testTime), State: "ERROR", Error: "My-error", Message: "my-message"}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "CACHE", Time: report.JSONTime(testTime), State: "WARNING", Message: "my-warning"}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: &coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, State: "SUCCESS", ObjectID: "object-id", Details: nil, Error: ""}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: &coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard2"}, State: "ERROR", Details: []report.Detail{{Type: report.DetailTypeError, Message: "error"}}, Error: anError}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: &coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard3"}, State: "SKIPPED", Details: []report.Detail{{Type: report.DetailTypeInfo, Message: "skipped"}}, Error: ""}, true)
	matcher.ContainsRecord(t, records, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: &coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard4"}, State: "EXCLUDED", Details: nil, Error: ""}, true)
}

// TestReporter_CorrectSummaryIfNoReportsMade tests that the summary is correct even if no reports are made.
// It also tests the basic structure of the summary itself.
func TestReporter_CorrectSummaryIfNoReportsMade(t *testing.T) {
	reportFilename := "test_report.jsonl"
	fs := testutils.TempFs(t)

	testTime := time.Unix(time.Now().Unix(), 0).UTC()

	r := report.NewDefaultReporterWithClockFunc(fs, reportFilename, func() time.Time { return testTime })
	r.Stop()

	summary := r.GetSummary()

	assert.Contains(t, summary, "Deployments success: 0\n")
	assert.Contains(t, summary, "Deployments errored: 0\n")
	assert.Contains(t, summary, "Deployments excluded: 0\n")
	assert.Contains(t, summary, "Deployments skipped: 0\n")
	assert.Contains(t, summary, fmt.Sprintf("Deploy start time: %s\n", testTime.Format("20060102-150405")))
	assert.Contains(t, summary, fmt.Sprintf("Deploy end time: %s\n", testTime.Format("20060102-150405")))
	assert.Contains(t, summary, "Deploy duration: 0s\n")
}
