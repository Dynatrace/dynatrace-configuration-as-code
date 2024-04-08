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

package events

import "sync"

type Aggregator[TERM Event, OUT any] struct {
	AggregationFunc  func(TERM, []Event) OUT
	correlatedEvents map[string][]Event
	mu               sync.Mutex
}

func (a *Aggregator[TERM, OUT]) Add(e Event) (OUT, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var o OUT
	if a.correlatedEvents == nil {
		a.correlatedEvents = make(map[string][]Event)
	}

	if et, ok := e.(TERM); ok {
		r := make([]Event, len(a.correlatedEvents[e.CorrelationId()]))
		copy(r, a.correlatedEvents[e.CorrelationId()])
		delete(a.correlatedEvents, e.CorrelationId())
		return a.AggregationFunc(et, r), true
	}
	a.correlatedEvents[e.CorrelationId()] = append(a.correlatedEvents[e.CorrelationId()], e)
	return o, false
}
