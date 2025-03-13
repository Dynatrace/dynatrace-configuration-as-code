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

package matcher

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

// ContainsInfoRecord asserts that an info record with a given message is present in the slice.
func ContainsInfoRecord(t *testing.T, records []report.Record, message string) {
	ContainsRecord(t, records, report.Record{
		Type:    report.TypeInfo,
		Time:    report.JSONTime{},
		State:   report.StateInfo,
		Message: message,
	}, true)
}

// ContainsRecord asserts that a given record should or should not exist within a slice. The comparison of the Error and Message is done via contains.
func ContainsRecord(t *testing.T, records []report.Record, wantedRecord report.Record, shouldExist bool) {
	t.Helper()

	if _, exists := FindRecord(records, wantedRecord); exists && !shouldExist {
		t.Errorf("Record %v exists in %v but should not exist", wantedRecord, records)
	} else if !exists && shouldExist {
		t.Errorf("Record %v does not exist in %v", wantedRecord, records)
	}
}

// FindRecord checks if a given record is in a slice and returns it and true if found, otherwise an empty record and false. The comparison of the Error and Message is done via contains.
func FindRecord(records []report.Record, wantedRecord report.Record) (report.Record, bool) {
	for _, record := range records {
		if isRecord(record, wantedRecord) {
			return record, true
		}
	}
	return report.Record{}, false
}

func isRecord(record, wanted report.Record) bool {
	if !cmp.Equal(record, wanted, cmpopts.IgnoreFields(report.Record{}, "Time", "Error", "Message", "Details")) {
		return false
	}
	if wanted.Error == "" && record.Error != "" {
		return false
	}
	if wanted.Message == "" && record.Message != "" {
		return false
	}

	return strings.Contains(record.Error, wanted.Error) && strings.Contains(record.Message, wanted.Message)
}
