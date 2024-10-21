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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/spf13/afero"
)

const (
	State_DEPL_SUCCESS  string = "SUCCESS"
	State_DEPL_ERR      string = "ERROR"
	State_DEPL_EXCLUDED string = "EXCLUDED"
	State_DEPL_SKIPPED  string = "SKIPPED"
)

type record struct {
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

type reporterContextKey struct{}

func NewContextWithReporter(ctx context.Context, r Reporter) context.Context {
	return context.WithValue(ctx, reporterContextKey{}, r)
}

func GetReporterFromContextOrDiscard(ctx context.Context) Reporter {
	v := ctx.Value(reporterContextKey{})
	if v == nil {
		return &discardReporter{}
	}
	switch v := v.(type) {
	case *defaultReporter:
		return v
	default:
		panic(fmt.Sprintf("unexpected value type for reporter context key: %T", v))
	}
}

type Reporter interface {
	ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error)
	GetSummary() string
	Stop()
}

type clock interface {
	Now() time.Time
}

type rtcClock struct{}

func (c *rtcClock) Now() time.Time {
	return time.Now()
}

type defaultReporter struct {
	queue                    chan record
	mu                       sync.Mutex
	wg                       sync.WaitGroup
	clock                    clock
	started                  time.Time
	ended                    time.Time
	deploymentsSuccessCount  int
	deploymentsErrorCount    int
	deploymentsExcludedCount int
	deploymentsSkippedCount  int
}

func NewDefaultReporter(fs afero.Fs, reportFilePath string) *defaultReporter {
	c := &rtcClock{}
	r := &defaultReporter{
		clock:   c,
		started: c.Now(),
		queue:   make(chan record, 32),
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if err := r.runRecorder(fs, reportFilePath); err != nil {
			log.Error("Error recording deployment report: %s", err)
		}
	}()
	return r
}

func (d *defaultReporter) runRecorder(fs afero.Fs, reportFilePath string) error {
	file, err := fs.OpenFile(reportFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error open record file: %w", err)
	}

	writer := bufio.NewWriter(file)
	for r := range d.queue {
		d.updateSummaryFromRecord(r)

		b, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("unable to convert record: %w", err)
		}

		if _, err := writer.Write(b); err != nil {
			return fmt.Errorf("unable to write record: %w", err)
		}

		if _, err := writer.WriteString("\n"); err != nil {
			return fmt.Errorf("unable to write newline: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("unable to flush record file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("unable to close record file: %w", err)
	}
	return nil
}

func (d *defaultReporter) updateSummaryFromRecord(r record) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.ended = time.Time(r.Time)
	switch r.State {
	case State_DEPL_SUCCESS:
		d.deploymentsSuccessCount++
	case State_DEPL_EXCLUDED:
		d.deploymentsExcludedCount++
	case State_DEPL_SKIPPED:
		d.deploymentsSkippedCount++
	case State_DEPL_ERR:
		d.deploymentsErrorCount++
	default:
		panic(fmt.Sprintf("unexpected state for deployment event: %s", r.State))
	}
}

func (d *defaultReporter) ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error) {
	d.queue <- record{
		Type:    "DEPLOY",
		Time:    JSONTime(d.clock.Now()),
		Config:  config,
		State:   state,
		Details: details,
		Error:   convertErrorToString(err),
	}
}

func convertErrorToString(err error) *string {
	if err == nil {
		return nil
	}
	errString := err.Error()
	return &errString
}

func (d *defaultReporter) GetSummary() string {
	d.mu.Lock()
	defer d.mu.Unlock()

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Deployments success: %d\n", d.deploymentsSuccessCount))
	sb.WriteString(fmt.Sprintf("Deployments errored: %d\n", d.deploymentsErrorCount))
	sb.WriteString(fmt.Sprintf("Deployments excluded: %d\n", d.deploymentsExcludedCount))
	sb.WriteString(fmt.Sprintf("Deployments skipped: %d\n", d.deploymentsSkippedCount))
	sb.WriteString(fmt.Sprintf("Deploy Start Time: %v\n", d.started.Format("20060102-150405")))
	sb.WriteString(fmt.Sprintf("Deploy End Time: %v\n", d.ended.Format("20060102-150405")))
	sb.WriteString(fmt.Sprintf("Deploy Duration: %v\n", d.ended.Sub(d.started)))
	return sb.String()
}

func (d *defaultReporter) Stop() {
	close(d.queue)
	d.wg.Wait()
}

type discardReporter struct{}

func (_ *discardReporter) ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error) {
}
func (_ *discardReporter) GetSummary() string { return "" }
func (_ *discardReporter) Stop()              {}
