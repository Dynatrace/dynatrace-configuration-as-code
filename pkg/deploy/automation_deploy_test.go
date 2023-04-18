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

package deploy

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	client "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeployAutomation(t *testing.T) {

	t.Run("happy day scenario", func(t *testing.T) {
		aut := automation{}

		c := NewMockautomationClient(gomock.NewController(t))
		c.EXPECT().Upsert(client.Workflows, "some_ID", []byte(`{"type": "json file"}`))
		aut.client = c

		givenMO := &config.Config{
			OriginObjectId: "some_ID",
			Type:           config.AutomationType{},
		}
		givenPayload := `{"type": "json file"}`

		_, errs := aut.deployAutomation(parameter.Properties{}, givenPayload, givenMO)

		assert.Emptyf(t, errs, "should be without errors, but recived %q", errs)
	})

	t.Run("invalid monaco config object type", func(t *testing.T) {
		aut := automation{client: NewMockautomationClient(gomock.NewController(t))}

		givenMO := &config.Config{
			Type: config.SettingsType{},
		}
		_, errs := aut.deployAutomation(parameter.Properties{}, "", givenMO)
		assert.Containsf(t, errs, fmt.Errorf("config was not of expected type %q, but %q", config.AutomationType{}.ID(), config.SettingsType{}.ID()), "recieved errors: %q", errs)
	})

	t.Run("Upsert automation monaco object without origin ID", func(t *testing.T) {
		aut := automation{}

		givenMO := &config.Config{
			Coordinate: coordinate.Coordinate{
				Project:  "test",
				Type:     "automation",
				ConfigId: "id",
			},
			OriginObjectId: "",
			Type:           config.AutomationType{},
		}
		givenPayload := `{"type": "json file"}`

		expectedID := idutils.GenerateUuidFromName(givenMO.Coordinate.String())
		c := NewMockautomationClient(gomock.NewController(t))
		c.EXPECT().Upsert(client.Workflows, expectedID, []byte(`{"type": "json file"}`))
		aut.client = c

		_, errs := aut.deployAutomation(parameter.Properties{}, givenPayload, givenMO)

		assert.Emptyf(t, errs, "should be without errors, but recived %q", errs)

	})
}
