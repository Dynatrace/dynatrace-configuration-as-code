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

package memory

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/pbnjay/memory"
	"math"
	"os"
	"runtime/debug"
)

const gibibyte = int64(1073741824)
const defaultLimit = gibibyte * 2
const defaultPercentTotal = 0.75

type getSystemMemoryF func() uint64

// SetDefaultLimit applies a soft memory limit for the runtime.
// If a user defined their own memory limit using the GOMEMLIMIT env var,
// this function does nothing and the requested limit is honored.
// As there is no simple portable way to find the available memory in the
// system - and we may not want to even consume a fixed percentage of that
// anyway - the limit is hardcoded in defaultLimit.
func SetDefaultLimit() bool {
	_, set := setLimit(memory.TotalMemory)
	return set
}

func setLimit(getSystemMemory getSystemMemoryF) (int64, bool) {
	// if there is a user defined limit, honor that instead
	if s, envVarSet := os.LookupEnv("GOMEMLIMIT"); envVarSet {
		log.Debug("Soft memory limit set via GOMEMLIMIT env var: %s", s)
		return 0, false
	}

	total := getSanitizedSysMem(getSystemMemory)
	limit := int64(defaultPercentTotal * float64(total))

	debug.SetMemoryLimit(limit)
	log.Debug("Default soft memory limit set: %s (%v%% of %s)", byteCountToHumanReadableUnit(uint64(limit)), defaultPercentTotal*100, byteCountToHumanReadableUnit(uint64(total)))
	return limit, true
}

func getSanitizedSysMem(f getSystemMemoryF) int64 {
	total := f()

	if total == 0 {
		// best guess 1GiB
		return gibibyte
	}

	if total > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(total)
}
