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
	"bufio"
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/spf13/afero"
	"time"
)

const (
	State_DEPL_SUCCESS  string = "SUCCESS"
	State_DEPL_ERR      string = "ERROR"
	State_DEPL_EXCLUDED string = "EXCLUDED"
	State_DEPL_SKIPPED  string = "SKIPPED"
)

type Record struct {
	Type    string                `json:"type"`
	Time    JSONTime              `json:"time"`
	Config  coordinate.Coordinate `json:"config"`
	State   string                `json:"state"`
	Details []Detail              `json:"details,omitempty"`
	Error   *string               `json:"error,omitempty"`
}

type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	s := time.Time(t).Format(time.RFC3339)
	return json.Marshal(s)
}

func (t *JSONTime) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	tVal, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}

	*t = JSONTime(tVal)
	return nil
}

func ReadReportFile(fs afero.Fs, filename string) ([]Record, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	var records []Record
	s := bufio.NewScanner(f)
	for s.Scan() {
		var r Record
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if s.Err() != nil {
		return nil, err
	}
	return records, nil
}
