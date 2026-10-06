//go:build unit

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

package grouppermissions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer/grouppermissions"
)

func permission(name, scope, scopeType string) accountmanagement.PermissionsDto {
	return accountmanagement.PermissionsDto{PermissionName: name, Scope: scope, ScopeType: scopeType}
}

func TestGetPermissionUpdates(t *testing.T) {
	viewerTenant := permission("tenant-viewer", "abc12345", "tenant")
	adminTenant := permission("tenant-admin", "abc12345", "tenant")
	viewerOtherTenant := permission("tenant-viewer", "xyz98765", "tenant")
	accountViewer := permission("account-viewer", "a1b2c3d4-uuid", "account")

	tests := []struct {
		name           string
		desired        []accountmanagement.PermissionsDto
		existing       []accountmanagement.PermissionsDto
		expectedCreate []accountmanagement.PermissionsDto
		expectedDelete []accountmanagement.PermissionsDto
	}{
		{
			name: "both nil",
		},
		{
			name:     "both empty",
			desired:  []accountmanagement.PermissionsDto{},
			existing: []accountmanagement.PermissionsDto{},
		},
		{
			name:           "nothing exists - all desired are created",
			desired:        []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
			existing:       nil,
			expectedCreate: []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
		},
		{
			name:           "nothing desired - all existing are deleted",
			desired:        nil,
			existing:       []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
			expectedDelete: []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
		},
		{
			name:     "identical sets - no changes",
			desired:  []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
			existing: []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
		},
		{
			name:     "identical sets in different order - no changes",
			desired:  []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
			existing: []accountmanagement.PermissionsDto{accountViewer, viewerTenant},
		},
		{
			name:           "partial overlap - missing are created, surplus are deleted",
			desired:        []accountmanagement.PermissionsDto{viewerTenant, adminTenant},
			existing:       []accountmanagement.PermissionsDto{viewerTenant, accountViewer},
			expectedCreate: []accountmanagement.PermissionsDto{adminTenant},
			expectedDelete: []accountmanagement.PermissionsDto{accountViewer},
		},
		{
			name:           "different permission name with same scope is a different permission",
			desired:        []accountmanagement.PermissionsDto{adminTenant},
			existing:       []accountmanagement.PermissionsDto{viewerTenant},
			expectedCreate: []accountmanagement.PermissionsDto{adminTenant},
			expectedDelete: []accountmanagement.PermissionsDto{viewerTenant},
		},
		{
			name:           "different scope with same permission name is a different permission",
			desired:        []accountmanagement.PermissionsDto{viewerOtherTenant},
			existing:       []accountmanagement.PermissionsDto{viewerTenant},
			expectedCreate: []accountmanagement.PermissionsDto{viewerOtherTenant},
			expectedDelete: []accountmanagement.PermissionsDto{viewerTenant},
		},
		{
			name:           "different scope type with same name and scope is a different permission",
			desired:        []accountmanagement.PermissionsDto{permission("viewer", "abc12345", "tenant")},
			existing:       []accountmanagement.PermissionsDto{permission("viewer", "abc12345", "management-zone")},
			expectedCreate: []accountmanagement.PermissionsDto{permission("viewer", "abc12345", "tenant")},
			expectedDelete: []accountmanagement.PermissionsDto{permission("viewer", "abc12345", "management-zone")},
		},
		{
			name:           "result preserves the input order",
			desired:        []accountmanagement.PermissionsDto{accountViewer, adminTenant, viewerOtherTenant},
			existing:       []accountmanagement.PermissionsDto{viewerTenant},
			expectedCreate: []accountmanagement.PermissionsDto{accountViewer, adminTenant, viewerOtherTenant},
			expectedDelete: []accountmanagement.PermissionsDto{viewerTenant},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toCreate, toDelete := grouppermissions.GetPermissionUpdates(tt.desired, tt.existing)

			assert.Equal(t, tt.expectedCreate, toCreate)
			assert.Equal(t, tt.expectedDelete, toDelete)
		})
	}
}

// Fields returned by the API (e.g. timestamps) must not affect identity, otherwise every
// existing permission would be deleted and recreated on each deployment.
func TestGetPermissionUpdates_IgnoresMetadataFields(t *testing.T) {
	createdAt := "2021-05-01T15:11:00Z"
	updatedAt := "2022-06-02T10:00:00Z"

	desired := []accountmanagement.PermissionsDto{permission("tenant-viewer", "abc12345", "tenant")}
	existing := []accountmanagement.PermissionsDto{{
		PermissionName:       "tenant-viewer",
		Scope:                "abc12345",
		ScopeType:            "tenant",
		CreatedAt:            &createdAt,
		UpdatedAt:            &updatedAt,
		AdditionalProperties: map[string]any{"foo": "bar"},
	}}

	toCreate, toDelete := grouppermissions.GetPermissionUpdates(desired, existing)

	assert.Empty(t, toCreate)
	assert.Empty(t, toDelete)
}

func TestGetPermissionUpdates_ReturnsOriginalExistingObjectForDeletion(t *testing.T) {
	createdAt := "2021-05-01T15:11:00Z"
	existingPermission := accountmanagement.PermissionsDto{
		PermissionName: "tenant-viewer",
		Scope:          "abc12345",
		ScopeType:      "tenant",
		CreatedAt:      &createdAt,
	}

	_, toDelete := grouppermissions.GetPermissionUpdates(nil, []accountmanagement.PermissionsDto{existingPermission})

	assert.Equal(t, []accountmanagement.PermissionsDto{existingPermission}, toDelete)
}
