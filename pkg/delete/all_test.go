//go:build unit

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

package delete_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
)

func TestDeleteAll_Segments(t *testing.T) {
	c := client.TestSegmentsClient{}

	t.Run("With Enabled Segment FF", func(t *testing.T) {
		t.Setenv(featureflags.Segments.EnvName(), "true")

		err := delete.All(t.Context(), client.ClientSet{SegmentClient: &c}, api.APIs{})
		assert.Error(t, err, "unimplemented")
	})

	t.Run("With Disabled Segment FF", func(t *testing.T) {
		t.Setenv(featureflags.Segments.EnvName(), "false")

		err := delete.All(t.Context(), client.ClientSet{SegmentClient: &c}, api.APIs{})
		assert.NoError(t, err)
	})
}

func TestDeleteAll_SLOv2(t *testing.T) {
	c := client.TestServiceLevelObjectiveClient{}

	t.Run("With Enabled SLOv2 FF", func(t *testing.T) {
		t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")

		err := delete.All(t.Context(), client.ClientSet{ServiceLevelObjectiveClient: &c}, api.APIs{})
		assert.Error(t, err, "unimplemented")
	})

	t.Run("With Disabled SLOv2 FF", func(t *testing.T) {
		t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "false")

		err := delete.All(t.Context(), client.ClientSet{ServiceLevelObjectiveClient: &c}, api.APIs{})
		assert.NoError(t, err)
	})
}
