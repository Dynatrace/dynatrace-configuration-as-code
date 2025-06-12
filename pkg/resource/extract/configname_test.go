//go:build unit

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

package extract

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

func TestExtractConfigName(t *testing.T) {
	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	name := "test"

	properties := parameter.Properties{
		config.NameParameter: name,
	}

	val, err := ConfigName(&conf, properties)

	require.NoError(t, err)
	assert.Equal(t, name, val)
}

func TestExtractConfigNameShouldFailOnMissingName(t *testing.T) {
	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	properties := parameter.Properties{}

	_, err := ConfigName(&conf, properties)

	require.Errorf(t, err, "error should not be nil (error val: %s)", err)
}

func TestExtractConfigNameShouldFailOnNameWithNonStringType(t *testing.T) {
	conf := config.Config{
		Template: testutils.GenerateDummyTemplate(t),
		Coordinate: coordinate.Coordinate{
			Project:  "project1",
			Type:     "dashboard",
			ConfigId: "dashboard-1",
		},
		Environment: "development",
		Parameters:  map[string]parameter.Parameter{},
		Skip:        false,
	}

	properties := parameter.Properties{
		config.NameParameter: 1,
	}

	_, err := ConfigName(&conf, properties)

	require.Errorf(t, err, "error should not be nil (error val: %s)", err)
}
