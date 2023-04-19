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

func (aut *Automation) deployAutomation(properties parameter.Properties, renderedConfig string, c *config.Config) (*parameter.ResolvedEntity, []error) {
	t, ok := c.Type.(config.AutomationType)
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

	var r *client.Response
	var e error
	switch t.Resource {
	case config.Workflow:
		r, e = aut.client.Upsert(client.Workflows, id, []byte(renderedConfig))
	case config.BusinessCalendar:
		r, e = aut.client.Upsert(client.BusinessCalendars, id, []byte(renderedConfig))
	case config.SchedulingRule:
		r, e = aut.client.Upsert(client.SchedulingRules, id, []byte(renderedConfig))
	default:
		r, e = nil, fmt.Errorf("unkonwn rsource type %q", t.Resource)
	}

	if e != nil {
		errs = append(errs, e)
	} else if r.Id != id {
		errs = append(errs, fmt.Errorf("ID of created object (%q) is different from given ID (%q)", r.Id, id))
	}

	resolved := parameter.ResolvedEntity{
		EntityName: c.Coordinate.ConfigId,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}

	return &resolved, errs
}
