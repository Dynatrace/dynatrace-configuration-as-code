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

package report

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClock struct{ t time.Time }

func (c *testClock) Now() time.Time { return c.t }

// TestReporter_ContextWithNoReporterDiscards tests that the Recorder obtained from an context without the default one discards.
func TestReporter_ContextWithNoReporterDiscards(t *testing.T) {
	ctx := context.TODO()
	reporter := GetReporterFromContextOrDiscard(ctx)
	require.NotNil(t, reporter)

	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard"}, State_DEPL_SUCCESS, nil, nil)
	reporter.Stop()
	assert.Empty(t, reporter.GetSummary(), "discarding Reporter should not return a summary")
}

// TestReporter_ContextWithDefaultReporterCollectsEvents tests that the Reporter obtained from an context with the default one attached collects events.
func TestReporter_ContextWithDefaultReporterCollectsEvents(t *testing.T) {

	reportFilename := "test_report.jsonl"
	fs := &afero.MemMapFs{}

	testTime := time.Unix(time.Now().Unix(), 0)

	r := NewDefaultReporter(fs, reportFilename)
	r.clock = &testClock{t: testTime}
	ctx := NewContextWithReporter(context.TODO(), r)

	reporter := GetReporterFromContextOrDiscard(ctx)
	require.NotNil(t, reporter)

	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, State_DEPL_SUCCESS, nil, nil)
	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard2"}, State_DEPL_ERR, []Detail{Detail{Type: TypeError, Message: "error"}}, errors.New("an error"))
	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard3"}, State_DEPL_SKIPPED, []Detail{Detail{Type: TypeInfo, Message: "skipped"}}, nil)
	reporter.ReportDeployment(coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard4"}, State_DEPL_EXCLUDED, nil, nil)

	reporter.Stop()

	exists, err := afero.Exists(fs, reportFilename)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NotEmpty(t, reporter.GetSummary(), "summary should not be empty")

	records, err := readReportFile(fs, reportFilename)
	require.NoError(t, err)

	assert.Len(t, records, 4)
	anError := "an error"
	assertRecordsEqual(t, records[0], record{Type: "DEPLOY", Time: JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard1"}, State: "SUCCESS", Details: nil, Error: nil})
	assertRecordsEqual(t, records[1], record{Type: "DEPLOY", Time: JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard2"}, State: "ERROR", Details: []Detail{Detail{Type: TypeError, Message: "error"}}, Error: &anError})
	assertRecordsEqual(t, records[2], record{Type: "DEPLOY", Time: JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard3"}, State: "SKIPPED", Details: []Detail{Detail{Type: TypeInfo, Message: "skipped"}}, Error: nil})
	assertRecordsEqual(t, records[3], record{Type: "DEPLOY", Time: JSONTime(testTime), Config: coordinate.Coordinate{Project: "test", Type: "dashboard", ConfigId: "my-dashboard4"}, State: "EXCLUDED", Details: nil, Error: nil})
}

func assertRecordsEqual(t *testing.T, expected record, actual record) {
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, time.Time(expected.Time).Format(time.RFC3339), time.Time(actual.Time).Format(time.RFC3339))
	assert.Equal(t, expected.Config, actual.Config)
	assert.Equal(t, expected.State, actual.State)
	assert.Equal(t, expected.Details, actual.Details)
	assert.Equal(t, expected.Error, actual.Error)
}

func readReportFile(fs afero.Fs, filename string) ([]record, error) {
	b, err := afero.ReadFile(fs, filename)
	if err != nil {
	}

	contents := strings.TrimSpace(string(b))
	lines := strings.Split(contents, "\n")
	records := []record{}
	for _, line := range lines {
		r := record{}
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
