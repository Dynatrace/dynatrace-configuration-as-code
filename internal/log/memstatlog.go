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

package log

import (
	"fmt"

	"runtime"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/spf13/afero"
)

var memstatFile afero.File

var LogMemStatsAsInfo bool

type extendedStats struct {
	runtime.MemStats
	readableTotal     string
	readableHeapAlloc string
	readableNextGC    string
	totalAllocMiB     float64
	heapAllocMiB      float64
	nextGCAtMiB       float64
	lastGCTimeUTC     time.Time
	goroutineCount    int
}

// LogMemStats creates a log line of memory stats which is useful for manually debugging/validating memory consumption.
// This is not used in general, but is highly useful when detailed memory information is needed - in which case it is
// nice to have a reusable method, rather than creating it again.
// Place this method where needed and supply location information - e.g. "before sort" and "after sort".
//
// This method will create a CSV file as well as write into the log.
//
// You can acquire further information - like mem stats sampled by minute or 10sec intervals - by creating a structured
// log and using the utility script tools/parse-memstats-from-json-log.sh to post-process the log file.
func LogMemStats(location string) { // nolint:unused

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	extended := extendedStats{
		MemStats:          stats,
		readableTotal:     strings.ByteCountToHumanReadableUnit(stats.TotalAlloc),
		readableHeapAlloc: strings.ByteCountToHumanReadableUnit(stats.HeapAlloc),
		readableNextGC:    strings.ByteCountToHumanReadableUnit(stats.NextGC),
		totalAllocMiB:     float64(stats.TotalAlloc) / 1024 / 1024,
		heapAllocMiB:      float64(stats.HeapAlloc) / 1024 / 1024,
		nextGCAtMiB:       float64(stats.NextGC) / 1024 / 1024,
		lastGCTimeUTC:     time.Unix(0, int64(stats.LastGC)).UTC(),
		goroutineCount:    runtime.NumGoroutine(),
	}

	if LogMemStatsAsInfo {
		writeLog(location, extended)
	}
	writeCsv(location, extended)
}

func writeLog(location string, stats extendedStats) { // nolint:unused
	WithFields(
		field.F("location", location),
		field.F("totalAlloc", stats.readableTotal),
		field.F("totalAllocMiB", stats.totalAllocMiB),
		field.F("totalAllocB", stats.TotalAlloc),
		field.F("heapAlloc", stats.readableHeapAlloc),
		field.F("heapAllocMiB", stats.heapAllocMiB),
		field.F("heapAllocB", stats.HeapAlloc),
		field.F("heapObjects", stats.HeapObjects),
		field.F("numGCRuns", stats.NumGC),
		field.F("lastGCRunTimestamp", stats.lastGCTimeUTC.String()),
		field.F("nextGCRunAt", stats.readableNextGC),
		field.F("nextGCRunAtMiB", stats.nextGCAtMiB),
		field.F("nextGCRunAtB", stats.NextGC),
		field.F("totalGCPauseNs", stats.PauseTotalNs),
		field.F("goroutineCount", stats.goroutineCount),
	).Info("### MEMSTATS ### %s ###\n- totalAlloc: %s\n- heapAlloc: %s\n- heapObjects: %d\n- GC runs: %d\n- Next GC at heap size: %s\n- totalGCPauseTime: %d ns",
		location,
		stats.readableTotal,
		stats.readableHeapAlloc,
		stats.HeapObjects,
		stats.NumGC,
		stats.readableNextGC,
		stats.PauseTotalNs)
}

func writeCsv(location string, stats extendedStats) { // nolint:unused
	if memstatFile != nil {
		//"heapAlloc, heapAllocMB, heapAllocByte, heapObjects, lastGCRun, "location", nextGCAtHeap, nextGCAtHeapMB, nextGCAtHeapByte, numGCRuns, totalAlloc, totalAllocMB, totalAllocByte, totalGCPauseNs, goroutineCount, ts\n"
		line := fmt.Sprintf("%v, %v, %v, %v, %v, %q, %v, %v, %v, %v, %v, %v, %v, %v, %v, %q\n",
			stats.readableHeapAlloc, stats.heapAllocMiB, stats.HeapAlloc,
			stats.HeapObjects,
			stats.lastGCTimeUTC.String(),
			location,
			stats.readableNextGC, stats.nextGCAtMiB, stats.NextGC,
			stats.NumGC, //numGCRuns
			stats.readableTotal, stats.totalAllocMiB, stats.TotalAlloc,
			stats.PauseTotalNs, //totalGCPauseNs
			stats.goroutineCount,
			time.Now().UTC().String(),
		)

		_, _ = memstatFile.WriteString(line)
	}
}

func createMemStatFile(fs afero.Fs, name string) error {
	f, err := fs.Create(name)
	if err != nil {
		return err
	}

	_, err = f.WriteString("heapAlloc, heapAllocMiB, heapAllocByte, heapObjects, lastGCRun, location, nextGCAtHeap, nextGCAtHeapMiB, nextGCAtHeapByte, numGCRuns, totalAlloc, totalAllocMiB, totalAllocByte, totalGCPauseNs,goroutineCount, ts\n")
	if err != nil {
		return err
	}

	memstatFile = f
	return nil
}

// LogPeriodicMemStats logs memory statistics periodically throughout the lifetime of a Monaco run.
func LogPeriodicMemStats() {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			LogMemStats("periodic")
		}
	}()
}
