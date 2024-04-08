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

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"time"
)

type InternalEvent struct {
	correlationId string
	time          time.Time
}

func (m InternalEvent) CorrelationId() string {
	return m.correlationId
}

func (m InternalEvent) Time() time.Time {
	return m.time
}

func NewInternalEventNow(correlationId string) InternalEvent {
	return InternalEvent{correlationId: correlationId, time: time.Now()}
}

func NewInternalEvent(correlationId string, time time.Time) InternalEvent {
	return InternalEvent{correlationId: correlationId, time: time}
}

type ConfigDeploymentFinishedEvent struct {
	InternalEvent
	Config coordinate.Coordinate
	State  string
	Error  error
}

type ConfigDeploymentLogEvent struct {
	InternalEvent
	Type    string
	Message string
}

type DeploymentStartedEvent struct {
	InternalEvent
	StartTime time.Time
}
type DeploymentEndedEvent struct {
	InternalEvent
	EndTime time.Time
}

const (
	State_DEPL_SUCCESS  string = "SUCCESS"
	State_DEPL_ERR      string = "ERROR"
	State_DEPL_EXCLUDED string = "EXCLUDED"
	State_DEPL_SKIPPED  string = "SKIPPED"
)
