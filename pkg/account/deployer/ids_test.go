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

package deployer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdMap(t *testing.T) {
	tests := []struct {
		name     string
		action   func(*idMap)
		validate func(*idMap)
	}{
		{
			name: "AddBoundary",
			action: func(d *idMap) {
				d.addBoundary("local1", "remote1")
			},
			validate: func(d *idMap) {
				assert.Equal(t, "remote1", d.getBoundaryUUID("local1"))
			},
		},
		{
			name: "AddPolicy",
			action: func(d *idMap) {
				d.addPolicy("local1", "remote1")
			},
			validate: func(d *idMap) {
				assert.Equal(t, "remote1", d.getPolicyUUID("local1"))
			},
		},
		{
			name: "AddGroup",
			action: func(d *idMap) {
				d.addGroup("local2", "remote2")
			},
			validate: func(d *idMap) {
				assert.Equal(t, "remote2", d.getGroupUUID("local2"))
			},
		},
		{
			name: "AddBoundaries",
			action: func(d *idMap) {
				boundaries := map[string]remoteId{
					"local3": "remote3",
					"local4": "remote4",
				}
				d.addBoundaries(boundaries)
			},
			validate: func(d *idMap) {
				assert.Equal(t, "remote3", d.getBoundaryUUID("local3"))
				assert.Equal(t, "remote4", d.getBoundaryUUID("local4"))
			},
		},
		{
			name: "AddPolicies",
			action: func(d *idMap) {
				policies := map[string]remoteId{
					"local3": "remote3",
					"local4": "remote4",
				}
				d.addPolicies(policies)
			},
			validate: func(d *idMap) {
				assert.Equal(t, "remote3", d.getPolicyUUID("local3"))
				assert.Equal(t, "remote4", d.getPolicyUUID("local4"))
			},
		},
		{
			name: "AddMZones",
			action: func(d *idMap) {
				mzones := []ManagementZone{
					{Id: "mz1", Parent: "env1", Name: "zone1"},
					{Id: "mz2", Parent: "env2", Name: "zone2"},
				}
				d.addMZones(mzones)
			},
			validate: func(d *idMap) {
				assert.Equal(t, "mz1", d.getMZoneUUID("env1", "zone1"))
				assert.Equal(t, "mz2", d.getMZoneUUID("env2", "zone2"))
				assert.Equal(t, "", d.getMZoneUUID("env3", "zone3"))
			},
		},
		{
			name: "AddGroups",
			action: func(d *idMap) {
				groups := map[string]remoteId{
					"local5": "remote5",
					"local6": "remote6",
				}
				d.addGroups(groups)
			},
			validate: func(d *idMap) {
				assert.Equal(t, "remote5", d.getGroupUUID("local5"))
				assert.Equal(t, "remote6", d.getGroupUUID("local6"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idMap := newIdMap()
			tt.action(&idMap)
			tt.validate(&idMap)
		})
	}
}
