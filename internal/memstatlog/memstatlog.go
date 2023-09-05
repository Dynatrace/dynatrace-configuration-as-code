/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package memstatlog

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"runtime"
)

// Write creates a log line of memory stats which is useful for manually debugging/validating memory consumption.
// This is not used in general, but is highly useful when detailed memory information is needed - in which case it is
// nice to have a reusable method, rather than creating it again.
// Place this method where needed and supply location information - e.g. "before sort" and "after sort".
func Write(location string) { // nolint:unused
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	totalAlloc := byteCountToHumanReadableUnit(stats.TotalAlloc)
	heapAlloc := byteCountToHumanReadableUnit(stats.HeapAlloc)

	log.WithFields(
		field.F("location", location),
		field.F("totalAlloc", totalAlloc),
		field.F("heapAlloc", heapAlloc),
		field.F("heapObjects", stats.HeapObjects),
		field.F("numGCRuns", stats.NumGC),
		field.F("totalGCPauseNs", stats.PauseTotalNs),
	).Info("### MEMSTATS ### %s ###\n- totalAlloc: %s\n- heapAlloc: %s\n- heapObjects: %d\n- GC runs: %d\n- totalGCPauseTime: %d ns",
		location,
		totalAlloc,
		heapAlloc,
		stats.HeapObjects,
		stats.NumGC,
		stats.PauseTotalNs)

}

func byteCountToHumanReadableUnit(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
