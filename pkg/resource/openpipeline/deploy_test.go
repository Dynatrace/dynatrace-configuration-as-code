//go:build unit

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

package openpipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
)

var opConfigCoordinate = coordinate.Coordinate{
	Project:  "proj",
	Type:     "logs",
	ConfigId: "logs",
}

func TestDeployOpenPipelineConfig(t *testing.T) {

	opConfig := &config.Config{
		Type:       config.OpenPipelineType{Kind: "logs"},
		Coordinate: opConfigCoordinate,
		Template:   testutils.GenerateDummyTemplate(t),
		Parameters: config.Parameters{},
	}

	t.Run("Update succeeds", func(t *testing.T) {
		client := NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq("logs"), gomock.Eq([]byte("{}"))).Times(1).Return(api.Response{}, nil)

		result, err := runDeployTest(t, client, opConfig)
		assert.NoError(t, err)
		require.NotEmpty(t, result.Properties)
	})

	t.Run("Update fails", func(t *testing.T) {
		client := NewMockDeploySource(gomock.NewController(t))
		client.EXPECT().Update(gomock.Any(), gomock.Eq("logs"), gomock.Eq([]byte("{}"))).Times(1).Return(api.Response{}, errors.New("connection error"))
		_, err := runDeployTest(t, client, opConfig)
		assert.Error(t, err)
	})
}

func runDeployTest(t *testing.T, client DeploySource, c *config.Config) (entities.ResolvedEntity, error) {
	parameters, errs := c.ResolveParameterValues(entities.New())
	require.Empty(t, errs)
	return NewDeployAPI(client).Deploy(t.Context(), parameters, "{}", c)
}
