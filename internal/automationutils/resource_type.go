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

package automationutils

import (
	"fmt"
	automationAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

func ClientResourceTypeFromConfigType(resource config.AutomationResource) (automationAPI.ResourceType, error) {
	switch resource {
	case config.Workflow:
		return automationAPI.Workflows, nil
	case config.BusinessCalendar:
		return automationAPI.BusinessCalendars, nil
	case config.SchedulingRule:
		return automationAPI.SchedulingRules, nil
	default:
		return -1, fmt.Errorf("unknown automation resource type %q", resource)
	}
}
