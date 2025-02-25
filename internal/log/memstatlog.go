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
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
)

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

func getStats() extendedStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return extendedStats{
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
}

func writeMemstatEntry(file afero.File, location string, stats extendedStats) error {
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

	_, err := file.WriteString(line)
	return err

}

func createMemStatFile(ctx context.Context, fs afero.Fs, name string) error {
	memStatFile, err := fs.Create(name)
	if err != nil {
		return fmt.Errorf("failed to open memstat file %q: %w", name, err)
	}

	defer func() {
		if err := memStatFile.Close(); err != nil {
			Warn("Failed to close memstat file: %s", err)
		}
	}()

	_, err = memStatFile.WriteString("heapAlloc, heapAllocMiB, heapAllocByte, heapObjects, lastGCRun, location, nextGCAtHeap, nextGCAtHeapMiB, nextGCAtHeapByte, numGCRuns, totalAlloc, totalAllocMiB, totalAllocByte, totalGCPauseNs,goroutineCount, ts\n")
	if err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-time.After(60 * time.Second):

			if err := writeMemstatEntry(memStatFile, "periodic", getStats()); err != nil {
				Warn("Failed to write entry in the memstat file: %s", err)
			}
		}
	}
}
