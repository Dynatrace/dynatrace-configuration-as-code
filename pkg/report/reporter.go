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

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

type reporterContextKey struct{}

// NewContextWithReporter returns a new Context associated with the specified Reporter.
func NewContextWithReporter(ctx context.Context, r Reporter) context.Context {
	return context.WithValue(ctx, reporterContextKey{}, r)
}

// GetReporterFromContextOrDiscard gets the Reporter associated with the Context or returns a discarding Reporter if none is available.
func GetReporterFromContextOrDiscard(ctx context.Context) Reporter {
	v := ctx.Value(reporterContextKey{})
	if v == nil {
		return &discardReporter{}
	}
	switch v := v.(type) {
	case Reporter:
		return v
	default:
		panic(fmt.Sprintf("unexpected value type for reporter context key: %T", v))
	}
}

// Reporter is a minimal interface for reporting events and retrieving summaries.
type Reporter interface {
	// ReportDeployment reports the result of deploying a config.
	ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error)

	// GetSummary returns a summary of all seen events as a string.
	GetSummary() string

	// Stop shuts down the Reporter, writing out all records.
	Stop()
}

// defaultReporter is a Reporter that writes events to a file.
type defaultReporter struct {
	queue                    chan Record
	mu                       sync.Mutex
	wg                       sync.WaitGroup
	clockFunc                func() time.Time
	started                  time.Time
	ended                    time.Time
	deploymentsSuccessCount  int
	deploymentsErrorCount    int
	deploymentsExcludedCount int
	deploymentsSkippedCount  int
}

// NewDefaultReporter creates a new Reporter that writes events as records as objects in a JSON lines file specified by reportFilePath.
func NewDefaultReporter(fs afero.Fs, reportFilePath string) Reporter {
	return newDefaultReporterWithClockFunc(fs, reportFilePath, func() time.Time { return time.Now() })
}

func newDefaultReporterWithClockFunc(fs afero.Fs, reportFilePath string, c func() time.Time) Reporter {
	r := &defaultReporter{
		clockFunc: c,
		started:   c(),
		queue:     make(chan Record, 32),
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

func (d *defaultReporter) updateSummaryFromRecord(r Record) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.ended = time.Time(r.Time)
	switch r.State {
	case StateDeploySuccess:
		d.deploymentsSuccessCount++
	case StateDeployExcluded:
		d.deploymentsExcludedCount++
	case StateDeploySkipped:
		d.deploymentsSkippedCount++
	case StateDeployError:
		d.deploymentsErrorCount++
	default:
		panic(fmt.Sprintf("unexpected state for deployment event: %s", r.State))
	}
}

// ReportDeployment reports the result of deploying a config.
func (d *defaultReporter) ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error) {
	d.queue <- Record{
		Type:    TypeDeploy,
		Time:    JSONTime(d.clockFunc()),
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

// GetSummary returns a summary of all seen events as a string.
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

// Stop shuts down the Reporter, writing out all records.
func (d *defaultReporter) Stop() {
	close(d.queue)
	d.wg.Wait()
}

type discardReporter struct{}

func (_ *discardReporter) ReportDeployment(config coordinate.Coordinate, state string, details []Detail, err error) {
}
func (_ *discardReporter) GetSummary() string { return "" }
func (_ *discardReporter) Stop()              {}
