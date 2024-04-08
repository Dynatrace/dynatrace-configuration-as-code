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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/events"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestSink struct {
	writtenBytes []byte
}

func (t *TestSink) Write(p []byte) error {
	t.writtenBytes = append(t.writtenBytes, p...)
	return nil
}

func (t *TestSink) Close() error {
	return nil
}

func TestNewFileCollector(t *testing.T) {

	testSink := TestSink{}
	fc, err := NewDeploySubscriber(&testSink)
	assert.NoError(t, err)
	assert.NotNil(t, fc)
}

func TestCollect(t *testing.T) {
	testSink := TestSink{}

	fc, _ := NewDeploySubscriber(&testSink)
	e1 := events.ConfigDeploymentLogEvent{
		InternalEvent: events.NewInternalEvent("abcde", time.Time{}),
		Type:          "WARN",
		Message:       "Something is sus",
	}

	e2 := events.ConfigDeploymentLogEvent{
		InternalEvent: events.NewInternalEvent("abcde", time.Time{}),
		Type:          "ERROR",
		Message:       "Something went wrong",
	}

	e3 := events.ConfigDeploymentFinishedEvent{
		InternalEvent: events.NewInternalEvent("abcde", time.Time{}),

		Config: coordinate.Coordinate{
			Project:  "p1",
			Type:     "t1",
			ConfigId: "id1",
		},
		State: "SUCCESS",
		Error: nil,
	}

	fc.Receive(e1)
	fc.Receive(e2)
	fc.Receive(e3)

	assert.Equal(t, []byte(`{"type":"DEPLOY","time":"-62135596800","config":{"project":"p1","type":"t1","configId":"id1"},"state":"SUCCESS","details":[{"type":"WARN","msg":"Something is sus"},{"type":"ERROR","msg":"Something went wrong"}]}`), testSink.writtenBytes)
}
