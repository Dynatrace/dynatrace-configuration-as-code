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

package automation

import (
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

var automationTypesToResources = map[config.AutomationType]automation.ResourceType{
	config.AutomationType{Resource: config.Workflow}:         automation.Workflows,
	config.AutomationType{Resource: config.BusinessCalendar}: automation.BusinessCalendars,
	config.AutomationType{Resource: config.SchedulingRule}:   automation.SchedulingRules,
}

var resourceTypeToAutomationResource = map[automation.ResourceType]config.AutomationResource{
	automation.Workflows:         config.Workflow,
	automation.BusinessCalendars: config.BusinessCalendar,
	automation.SchedulingRules:   config.SchedulingRule,
}
