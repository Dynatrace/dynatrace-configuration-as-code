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
	"strconv"
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
	Type    string
	Time    time.Time
	Config  coordinate.Coordinate
	State   string
	Details []Detail
	Error   error
}

func (r record) ToJSON() (string, error) {
	dto := struct {
		Type    string                `json:"type"`
		Time    string                `json:"time"`
		Config  coordinate.Coordinate `json:"config"`
		State   string                `json:"state"`
		Details []Detail              `json:"details,omitempty"`
		Error   error                 `json:"error,omitempty"`
	}{
		Type:    r.Type,
		Time:    strconv.FormatInt(r.Time.Unix(), 10),
		Config:  r.Config,
		State:   r.State,
		Details: r.Details,
		Error:   r.Error,
	}
	jsonEvent, err := json.Marshal(dto)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON: %w", err)
	}
	return string(jsonEvent), nil
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

type defaultReporter struct {
	queue                    chan record
	mu                       sync.Mutex
	wg                       sync.WaitGroup
	started                  time.Time
	ended                    time.Time
	deploymentFinishedCount  int
	deploymentsExcludedCount int
	deploymentsSkippedCount  int
}

func NewDefaultReporter(reportFilePath string) *defaultReporter {
	r := &defaultReporter{
		started: time.Now(),
		queue:   make(chan record, 32),
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if err := r.runRecorder(reportFilePath); err != nil {
			log.Error("Error recording deployment report: %s", err)
		}
	}()
	return r
}

func (d *defaultReporter) runRecorder(reportFilePath string) error {
	file, err := afero.NewOsFs().OpenFile(reportFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error open record file: %w", err)
	}

	writer := bufio.NewWriter(file)
	for r := range d.queue {
		d.updateSummaryFromRecord(r)
		b, err := r.ToJSON()
		if err != nil {
			return fmt.Errorf("unable to convert record: %w", err)
		}
		if _, err := fmt.Fprintln(writer, b); err != nil {
			return fmt.Errorf("unable to write record: %w", err)
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

	d.ended = r.Time
	switch r.State {
	case State_DEPL_SUCCESS:
		d.deploymentFinishedCount++
	case State_DEPL_EXCLUDED:
		d.deploymentsExcludedCount++
	case State_DEPL_SKIPPED:
		d.deploymentsSkippedCount++
	default:
		panic(fmt.Sprintf("unexpected state for deployment event: %s", r.State))
	}
}

func (d *defaultReporter) ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error) {
	d.queue <- record{
		Type:    "DEPLOY",
		Time:    time.Now(),
		Config:  config,
		State:   state,
		Details: details,
		Error:   err,
	}
}

func (d *defaultReporter) GetSummary() string {
	d.mu.Lock()
	defer d.mu.Unlock()

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Deployments success: %d\n", d.deploymentFinishedCount))
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
