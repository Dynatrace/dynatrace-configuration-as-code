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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

type testClock struct{ t time.Time }

func (c *testClock) Now() time.Time { return c.t }

// TestReporter_ContextWithNoReporterDiscards tests that the Recorder obtained from an context without the default one discards.
func TestReporter_ContextWithNoReporterDiscards(t *testing.T) {
	ctx := context.TODO()
	reporter := report.GetReporterFromContextOrDiscard(ctx)
	require.NotNil(t, reporter)

	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard"}, report.State_DEPL_SUCCESS, nil, nil)
	reporter.Stop()
	assert.Empty(t, reporter.GetSummary(), "discarding Reporter should not return a summary")
}

// TestReporter_ContextWithDefaultReporterCollectsEvents tests that the Reporter obtained from an context with the default one attached collects events.
func TestReporter_ContextWithDefaultReporterCollectsEvents(t *testing.T) {

	reportFilename := "test_report.jsonl"
	fs := &afero.MemMapFs{}

	testTime := time.Unix(time.Now().Unix(), 0).UTC()

	r := report.NewDefaultReporterWithClock(fs, reportFilename, &testClock{t: testTime})
	ctx := report.NewContextWithReporter(context.TODO(), r)

	reporter := report.GetReporterFromContextOrDiscard(ctx)
	require.NotNil(t, reporter)

	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, report.State_DEPL_SUCCESS, nil, nil)
	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard2"}, report.State_DEPL_ERR, []report.Detail{report.Detail{Type: report.TypeError, Message: "error"}}, errors.New("an error"))
	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard3"}, report.State_DEPL_SKIPPED, []report.Detail{report.Detail{Type: report.TypeInfo, Message: "skipped"}}, nil)
	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard4"}, report.State_DEPL_EXCLUDED, nil, nil)

	reporter.Stop()

	exists, err := afero.Exists(fs, reportFilename)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NotEmpty(t, reporter.GetSummary(), "summary should not be empty")

	records, err := report.ReadReportFile(fs, reportFilename)
	require.NoError(t, err)

	require.Len(t, records, 4)
	anError := "an error"
	assert.Equal(t, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, State: "SUCCESS", Details: nil, Error: nil}, records[0])
	assert.Equal(t, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard2"}, State: "ERROR", Details: []report.Detail{report.Detail{Type: report.TypeError, Message: "error"}}, Error: &anError}, records[1])
	assert.Equal(t, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard3"}, State: "SKIPPED", Details: []report.Detail{report.Detail{Type: report.TypeInfo, Message: "skipped"}}, Error: nil}, records[2])
	assert.Equal(t, report.Record{Type: "DEPLOY", Time: report.JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard4"}, State: "EXCLUDED", Details: nil, Error: nil}, records[3])
}
