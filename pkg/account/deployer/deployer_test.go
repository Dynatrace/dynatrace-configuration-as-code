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
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
)

func testResources(t *testing.T) *account.Resources {
	res, err := loader.Load(afero.NewOsFs(), "testdata/accdata.yaml")
	assert.NoError(t, err)
	return res
}

func mockClient(t *testing.T) *Mockclient {
	mockedClient := NewMockclient(gomock.NewController(t))
	mockedClient.EXPECT().getAccountInfo().AnyTimes().Return(account.AccountInfo{Name: "my-account", AccountUUID: "1334-1223-1112-1111"})
	return mockedClient
}

func TestDeployer(t *testing.T) {
	t.Run("Deployer - Getting global policies fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(nil, errors.New("ERR - GET GLOBAL POLICIES"))
		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Getting management zones fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(nil, errors.New("ERR - GET GLOBAL POLICIES"))
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return(nil, errors.New("ERR - GET MANAGEMENT ZONES"))
		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Upserting policy fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("ERR - UPSERT POLICY"))
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("3158497c-7fc7-44bc-ab15-c3ab8fea8560", nil)
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("5b9aaf94-26d0-4464-a469-3d8563612554", nil)

		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Upserting group fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("e59db51f-2ce1-4489-82ba-f1f00a93a85e", nil)
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("ERR - UPSERT GROUP"))
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("5b9aaf94-26d0-4464-a469-3d8563612554", nil)

		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Updating Group <-> Policy Bindings fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)

		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("e59db51f-2ce1-4489-82ba-f1f00a93a85e", nil)
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("3158497c-7fc7-44bc-ab15-c3ab8fea8560", nil)
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("5b9aaf94-26d0-4464-a469-3d8563612554", nil)
		mockedClient.EXPECT().updateGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updatePermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updateAccountPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("ERR - POLICY BINDINGS"))

		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Upserting Group Permissions fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)

		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("e59db51f-2ce1-4489-82ba-f1f00a93a85e", nil)
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("3158497c-7fc7-44bc-ab15-c3ab8fea8560", nil)
		mockedClient.EXPECT().updateAccountPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updateEnvironmentPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("5b9aaf94-26d0-4464-a469-3d8563612554", nil)
		mockedClient.EXPECT().updateGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updatePermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("ERR - GROUP PERMISSIONS"))

		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Upserting User fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("e59db51f-2ce1-4489-82ba-f1f00a93a85e", nil)
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("3158497c-7fc7-44bc-ab15-c3ab8fea8560", nil)
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("", errors.New("ERR - UPSERT USER"))

		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - Updating Group Bindings fails", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("e59db51f-2ce1-4489-82ba-f1f00a93a85e", nil)
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("31»58497c-7fc7-44bc-ab15-c3ab8fea8560", nil)
		mockedClient.EXPECT().updateAccountPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updateEnvironmentPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updatePermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("5b9aaf94-26d0-4464-a469-3d8563612554", nil)
		mockedClient.EXPECT().updateGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("ERR - GROUP BINDINGS"))
		err := instance.Deploy(t.Context(), testResources(t))
		assert.Error(t, err)
	})

	t.Run("Deployer - OK", func(t *testing.T) {
		mockedClient := mockClient(t)
		instance := NewAccountDeployer(mockedClient)
		mockedClient.EXPECT().getAllGroups(gomock.Any()).Return(map[string]remoteId{}, nil)
		mockedClient.EXPECT().getGlobalPolicies(gomock.Any()).Return(map[string]remoteId{"builtin-policy-1": "6a269841-ac77-47ca-9e39-3663ddd9bf9b"}, nil)
		mockedClient.EXPECT().getManagementZones(gomock.Any()).Return([]accountmanagement.ManagementZoneResourceDto{{Parent: "env12345", Name: "Mzone", Id: "-3664092122630505211"}}, nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("8f14c703-aa31-4d33-b888-edd553aea02c", nil)
		mockedClient.EXPECT().upsertPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("e59db51f-2ce1-4489-82ba-f1f00a93a85e", nil)
		mockedClient.EXPECT().upsertGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return("3158497c-7fc7-44bc-ab15-c3ab8fea8560", nil)
		mockedClient.EXPECT().updateAccountPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updateEnvironmentPolicyBindings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().updatePermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockedClient.EXPECT().upsertUser(gomock.Any(), gomock.Any()).Return("5b9aaf94-26d0-4464-a469-3d8563612554", nil)
		mockedClient.EXPECT().updateGroupBindings(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		err := instance.Deploy(t.Context(), testResources(t))
		assert.NoError(t, err)
	})
}
