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

// Writer writes rendered payloads as individual JSON files under a timestamped directory in .logs.
type Writer struct {
	mu      sync.Mutex
	baseDir string
	count   int
}

type ctxPayloadWriterKey struct{}

// NewContextWithWriter returns a new context with a payload Writer attached.
// All payload files will be written under .logs/{timestamp}-payloads/.
func NewContextWithWriter(ctx context.Context) context.Context {
	timestamp := timeutils.TimeAnchor().Format(log.LogFileTimestampPrefixFormat)
	baseDir := filepath.Join(log.LogDirectory, timestamp+"-payloads")
	return context.WithValue(ctx, ctxPayloadWriterKey{}, &Writer{baseDir: baseDir})
}

// newWriterWithDir creates a Writer using the given base directory. Used in tests.
func newWriterWithDir(baseDir string) *Writer {
	return &Writer{baseDir: baseDir}
}

// GetWriterFromContext returns the Writer from the context, or nil if not set.
func GetWriterFromContext(ctx context.Context) *Writer {
	w, _ := ctx.Value(ctxPayloadWriterKey{}).(*Writer)
	return w
}

// Record writes the rendered payload for a config to its own JSON file immediately.
// The file is placed at {baseDir}/{env}/{project}/{type}/{configId}.json.
// Colons in the type name are replaced with dashes for filesystem compatibility.
// Environment variable parameter values are redacted before writing.
func (w *Writer) Record(coord coordinate.Coordinate, env string, renderedConfig string, params config.Parameters) {
	redacted := redactSecrets(renderedConfig, params)
	filePath := w.payloadFilePath(coord, env)

	if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		log.Error("Failed to create payload directory %s: %v", filepath.Dir(filePath), err)
		return
	}
	if err := os.WriteFile(filePath, []byte(redacted), 0640); err != nil {
		log.Error("Failed to write payload file %s: %v", filePath, err)
		return
	}

	w.mu.Lock()
	w.count++
	w.mu.Unlock()
}

// Finish logs a summary of the written payload files.
func (w *Writer) Finish() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.count == 0 {
		return
	}
	log.Info("Rendered payloads (%d configs) written to %s", w.count, w.baseDir)
}

// payloadFilePath returns the file path for a given config coordinate and environment.
// Colons in the type are replaced with dashes (e.g. builtin:alerting.profile → builtin-alerting.profile).
func (w *Writer) payloadFilePath(coord coordinate.Coordinate, env string) string {
	safeType := strings.ReplaceAll(coord.Type, ":", "-")
	return filepath.Join(w.baseDir, env, coord.Project, safeType, coord.ConfigId+".json")
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
