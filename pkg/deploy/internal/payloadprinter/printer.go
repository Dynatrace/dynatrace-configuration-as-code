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

package payloadprinter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
)

const redactedValue = "***"

// Writer collects rendered payloads and writes them to a file in the .logs directory.
type Writer struct {
	mu      sync.Mutex
	entries []entry
}

type entry struct {
	coord    coordinate.Coordinate
	env      string
	rendered string
}

type ctxPayloadWriterKey struct{}

// NewContextWithWriter returns a new context with a payload Writer attached.
func NewContextWithWriter(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxPayloadWriterKey{}, &Writer{})
}

// GetWriterFromContext returns the Writer from the context, or nil if not set.
func GetWriterFromContext(ctx context.Context) *Writer {
	w, _ := ctx.Value(ctxPayloadWriterKey{}).(*Writer)
	return w
}

// Record adds a rendered payload entry to the writer.
// Environment variable parameter values are redacted before storing.
func (w *Writer) Record(coord coordinate.Coordinate, env string, renderedConfig string, params config.Parameters) {
	redacted := redactSecrets(renderedConfig, params)

	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries = append(w.entries, entry{coord: coord, env: env, rendered: redacted})
}

// Finish writes all collected payloads to a file in the .logs directory
// and logs the file path for the user.
func (w *Writer) Finish() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.entries) == 0 {
		return
	}

	filePath := payloadFilePath()
	if err := os.MkdirAll(log.LogDirectory, 0750); err != nil {
		log.Error("Failed to create log directory for rendered payloads: %v", err)
		return
	}

	f, err := os.Create(filePath)
	if err != nil {
		log.Error("Failed to create rendered payloads file: %v", err)
		return
	}
	defer f.Close()

	for _, e := range w.entries {
		writeEntry(f, e.coord, e.env, e.rendered)
	}

	log.Info("Rendered payloads (%d configs) written to %s", len(w.entries), filePath)
}

func payloadFilePath() string {
	timestamp := timeutils.TimeAnchor().Format(log.LogFileTimestampPrefixFormat)
	return filepath.Join(log.LogDirectory, timestamp+"-rendered-payloads.txt")
}

// writeEntry writes a single payload entry to the given writer.
func writeEntry(w io.Writer, coord coordinate.Coordinate, env string, rendered string) {
	fmt.Fprintf(w, "--- Rendered payload: %s | environment: %s ---\n", coord, env)
	fmt.Fprintln(w, rendered)
	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)
}

// redactSecrets replaces resolved environment variable parameter values in the rendered config with a redacted placeholder.
func redactSecrets(renderedConfig string, params config.Parameters) string {
	for name, p := range params {
		if envP, ok := p.(*envParam.EnvironmentVariableParameter); ok {
			val, err := envP.ResolveValue(parameter.ResolveContext{ParameterName: name})
			if err != nil {
				continue
			}
			if strVal, ok := val.(string); ok && strVal != "" {
				renderedConfig = strings.ReplaceAll(renderedConfig, strVal, redactedValue)
			}
		}
	}
	return renderedConfig
}
