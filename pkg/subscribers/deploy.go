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
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/events"
	"strconv"
)

type Detail struct {
	Type    string `json:"type"`
	Message string `json:"msg"`
}
type ConfigRecord struct {
	Type    string                `json:"type"`
	Time    string                `json:"time"`
	Config  coordinate.Coordinate `json:"config"`
	State   string                `json:"state"`
	Details []Detail              `json:"details"`
	Error   error                 `json:"error,omitempty"`
}

type DeploySubscriber struct {
	sink       events.Sink
	aggregator events.Aggregator[events.ConfigDeploymentFinishedEvent, ConfigRecord]
}

func NewDeploySubscriber(sink events.Sink) (*DeploySubscriber, error) {
	return &DeploySubscriber{
		sink: sink,
		aggregator: events.Aggregator[events.ConfigDeploymentFinishedEvent, ConfigRecord]{
			AggregationFunc: func(termEvent events.ConfigDeploymentFinishedEvent, ev []events.Event) ConfigRecord {
				cr := ConfigRecord{
					Type:   "DEPLOY",
					Time:   strconv.FormatInt(termEvent.Time().Unix(), 10),
					Config: termEvent.Config,
					State:  termEvent.State,
					Error:  termEvent.Error,
				}
				for _, e := range ev {
					if de, ok := e.(events.ConfigDeploymentLogEvent); ok {
						d := Detail{}
						d.Type = de.Type
						d.Message = de.Message
						cr.Details = append(cr.Details, d)
					}
				}
				return cr
			},
		},
	}, nil
}

func (fc *DeploySubscriber) Receive(e events.Event) events.Event {
	configRecord, ok := fc.aggregator.Add(e)
	if !ok {
		return e
	}
	jsonEvent, err := json.Marshal(configRecord)
	if err != nil {
		log.Error("could not marshal event: %v", err)
	}
	err = fc.sink.Write(jsonEvent)
	if err != nil {
		log.Error("could not write to file: %v", err)
	}
	return e
}

func (fc *DeploySubscriber) Stop() {
	fc.sink.Close()
}

func (fc *DeploySubscriber) EventTypes() []events.Event {
	return []events.Event{
		events.ConfigDeploymentFinishedEvent{},
		events.ConfigDeploymentLogEvent{},
	}
}
