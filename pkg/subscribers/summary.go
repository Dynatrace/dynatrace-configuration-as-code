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

package subscribers

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/events"
	"strings"
	"time"
)

type SummarySubscriber struct {
	Sink                     events.Sink
	started                  time.Time
	ended                    time.Time
	deploymentFinishedCount  int
	deploymentsExcludedCount int
	deploymentsSkippedCount  int
}

func (s *SummarySubscriber) Receive(event events.Event) events.Event {
	switch ev := event.(type) {
	case events.DeploymentStartedEvent:
		s.started = ev.StartTime
	case events.DeploymentEndedEvent:
		s.ended = ev.EndTime
	case events.ConfigDeploymentFinishedEvent:
		if ev.State == events.State_DEPL_SUCCESS {
			s.deploymentFinishedCount++
			break
		}
		if ev.State == events.State_DEPL_EXCLUDED {
			s.deploymentsExcludedCount++
			break
		}
		if ev.State == events.State_DEPL_SKIPPED {
			s.deploymentsSkippedCount++
			break
		}
	}
	return event
}

func (s *SummarySubscriber) Stop() {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Deplyoments success: %d\n", s.deploymentFinishedCount))
	sb.WriteString(fmt.Sprintf("Deployments excluded: %d\n", s.deploymentsExcludedCount))
	sb.WriteString(fmt.Sprintf("Deployments skipped: %d\n", s.deploymentsSkippedCount))
	sb.WriteString(fmt.Sprintf("Deploy Start Time: %v\n", s.started.Format("20060102-150405")))
	sb.WriteString(fmt.Sprintf("Deploy End Time: %v\n", s.ended.Format("20060102-150405")))
	sb.WriteString(fmt.Sprintf("Deploy Duration: %v\n", s.ended.Sub(s.started)))
	s.Sink.Write([]byte(sb.String()))
	s.Sink.Close()
}

func (s *SummarySubscriber) EventTypes() []events.Event {
	return []events.Event{
		events.DeploymentStartedEvent{},
		events.DeploymentEndedEvent{},
		events.ConfigDeploymentFinishedEvent{}}
}
