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
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"sync"
)

type idMap struct {
	polIds map[localId]remoteId
	pMu    sync.RWMutex
	grIds  map[localId]remoteId
	grMu   sync.RWMutex
	mzIds  []accountmanagement.ManagementZoneResourceDto
	mzMu   sync.RWMutex
}

func newIdMap() idMap {
	return idMap{
		polIds: make(map[localId]remoteId),
		pMu:    sync.RWMutex{},
		grIds:  make(map[localId]remoteId),
		grMu:   sync.RWMutex{},
		mzIds:  []accountmanagement.ManagementZoneResourceDto{},
		mzMu:   sync.RWMutex{},
	}
}
func (d *idMap) addPolicy(localId localId, remoteId remoteId) {
	d.pMu.Lock()
	defer d.pMu.Unlock()
	d.polIds[localId] = remoteId
}

func (d *idMap) addGroup(localId localId, remoteId remoteId) {
	d.grMu.Lock()
	defer d.grMu.Unlock()
	d.grIds[localId] = remoteId
}

func (d *idMap) addPolicies(policies map[string]remoteId) {
	d.pMu.Lock()
	defer d.pMu.Unlock()
	for k, v := range policies {
		d.polIds[k] = v
	}
}

func (d *idMap) addMZones(mzones []ManagementZone) {
	d.mzMu.Lock()
	defer d.mzMu.Unlock()
	d.mzIds = append(d.mzIds, mzones...)
}

func (d *idMap) addGroups(groups map[string]remoteId) {
	d.grMu.Lock()
	defer d.grMu.Unlock()
	for k, v := range groups {
		d.grIds[k] = v
	}
}

func (d *idMap) getPolicyUUID(id localId) remoteId {
	d.pMu.RLock()
	defer d.pMu.RUnlock()
	return d.polIds[id]
}

func (d *idMap) getGroupUUID(id localId) remoteId {
	d.grMu.RLock()
	defer d.grMu.RUnlock()
	return d.grIds[id]
}

func (d *idMap) getMZoneUUID(envName, mzName string) remoteId {
	d.mzMu.RLock()
	defer d.mzMu.RUnlock()
	for _, z := range d.mzIds {
		if z.Parent == envName && z.Name == mzName {
			return z.Id
		}
	}
	return ""
}
