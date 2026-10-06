/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package grouppermissions

import (
	"fmt"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
)

// GetPermissionUpdates returns which permissions should be created and deleted
// Note: Update is not supported by the API
func GetPermissionUpdates(desired, existing []accountmanagement.PermissionsDto) (createPermissions, deletePermissions []accountmanagement.PermissionsDto) {
	existingMap := toMap(existing)
	desiredMap := toMap(desired)

	for _, p := range desired {
		if _, ok := existingMap[toIdentifier(p)]; !ok {
			// Is locally but not on the API side => create
			createPermissions = append(createPermissions, p)
		}
	}

	for _, p := range existing {
		if _, ok := desiredMap[toIdentifier(p)]; !ok {
			// Is on the API side but not local => delete
			deletePermissions = append(deletePermissions, p)
		}
	}
	return
}

func toMap(permissions []accountmanagement.PermissionsDto) map[string]struct{} {
	result := make(map[string]struct{})
	for _, p := range permissions {
		result[toIdentifier(p)] = struct{}{}
	}
	return result
}

func toIdentifier(permission accountmanagement.PermissionsDto) string {
	return fmt.Sprintf("%s#-#%s#-#%s", permission.PermissionName, permission.Scope, permission.ScopeType)
}
