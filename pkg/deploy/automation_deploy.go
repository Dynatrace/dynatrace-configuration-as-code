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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
)

func (ctx *automation) deployAutomation(properties parameter.Properties, renderedConfig string, c *config.Config) (*parameter.ResolvedEntity, []error) {
	_, ok := c.Type.(config.AutomationType)
	if !ok {
		return &parameter.ResolvedEntity{}, []error{fmt.Errorf("config was not of expected type %q, but %q", config.AutomationType{}.ID(), c.Type.ID())}
	}

	var id string
	var errs []error

	if c.OriginObjectId != "" {
		id = c.OriginObjectId
	} else {
		id = idutils.GenerateUuidFromName(c.Coordinate.String())
	}

	_, _ = ctx.client.Upsert(client.Workflows, id, []byte(renderedConfig))

	resolved := parameter.ResolvedEntity{
		EntityName: c.Coordinate.ConfigId,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}

	return &resolved, errs
}
