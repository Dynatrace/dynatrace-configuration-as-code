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

package delete_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
)

type testClient struct {
	AccountUUID           string
	userFunc              func(ctx context.Context, accountUUID, email string) error
	serviceUserFunc       func(ctx context.Context, accountUUID, name string) error
	groupFunc             func(ctx context.Context, accountUUID, name string) error
	accountPolicyFunc     func(ctx context.Context, name string) error
	environmentPolicyFunc func(ctx context.Context, environmentID, name string) error
}

var _ delete.Client = (*testClient)(nil)

func (c *testClient) DeleteUser(ctx context.Context, email string) error {
	return c.userFunc(ctx, c.AccountUUID, email)
}

func (c *testClient) DeleteServiceUser(ctx context.Context, name string) error {
	return c.serviceUserFunc(ctx, c.AccountUUID, name)
}

func (c *testClient) DeleteGroup(ctx context.Context, name string) error {
	return c.groupFunc(ctx, c.AccountUUID, name)
}

func (c *testClient) DeleteAccountPolicy(ctx context.Context, name string) error {
	return c.accountPolicyFunc(ctx, name)
}

func (c *testClient) DeleteEnvironmentPolicy(ctx context.Context, environmentID, name string) error {
	return c.environmentPolicyFunc(ctx, environmentID, name)
}

func TestDeletesResources(t *testing.T) {
	userDeleteCalled := 0
	groupDeleteCalled := 0
	accountPolicyDeleteCalled := 0
	environmentPolicyDeleteCalled := 0
	c := testClient{
		userFunc: func(ctx context.Context, accountUUID, email string) error {
			userDeleteCalled++
			return nil
		},
		groupFunc: func(ctx context.Context, accountUUID, name string) error {
			groupDeleteCalled++
			return nil
		},
		accountPolicyFunc: func(ctx context.Context, name string) error {
			accountPolicyDeleteCalled++
			return nil
		},
		environmentPolicyFunc: func(ctx context.Context, envID, name string) error {
			environmentPolicyDeleteCalled++
			return nil
		},
	}
	entriesToDelete := delete.Resources{
		Users: []delete.User{
			{
				Email: "test@user.com",
			},
			{
				Email: "another@user.com",
			},
		},
		Groups: []delete.Group{
			{
				Name: "test-group",
			},
		},
		AccountPolicies: []delete.AccountPolicy{
			{
				Name: "test-policy",
			},
		},
		EnvironmentPolicies: []delete.EnvironmentPolicy{
			{
				Name:        "test-policy",
				Environment: "abc1234567",
			},
		},
	}
	acc := delete.Account{
		Name:      "name",
		UUID:      "1234",
		APIClient: &c,
	}
	err := delete.DeleteAccountResources(t.Context(), acc, entriesToDelete)
	assert.NoError(t, err)
	assert.Equal(t, 2, userDeleteCalled)
	assert.Equal(t, 1, groupDeleteCalled)
	assert.Equal(t, 1, accountPolicyDeleteCalled)
	assert.Equal(t, 1, environmentPolicyDeleteCalled)
}

func TestContinuesDeletionIfOneTypeFails(t *testing.T) {
	userDeleteCalled := 0
	accountPolicyDeleteCalled := 0
	environmentPolicyDeleteCalled := 0
	c := testClient{
		userFunc: func(ctx context.Context, accountUUID, email string) error {
			userDeleteCalled++
			return nil
		},
		groupFunc: func(ctx context.Context, accountUUID, name string) error {
			return errors.New("fail")
		},
		accountPolicyFunc: func(ctx context.Context, name string) error {
			accountPolicyDeleteCalled++
			return nil
		},
		environmentPolicyFunc: func(ctx context.Context, envID, name string) error {
			environmentPolicyDeleteCalled++
			return nil
		},
	}
	entriesToDelete := delete.Resources{
		Users: []delete.User{
			{
				Email: "test@user.com",
			},
			{
				Email: "another@user.com",
			},
		},
		Groups: []delete.Group{
			{
				Name: "test-group",
			},
		},
		AccountPolicies: []delete.AccountPolicy{
			{
				Name: "test-policy",
			},
		},
		EnvironmentPolicies: []delete.EnvironmentPolicy{
			{
				Name:        "test-policy",
				Environment: "abc1234567",
			},
		},
	}
	acc := delete.Account{
		Name:      "name",
		UUID:      "1234",
		APIClient: &c,
	}
	err := delete.DeleteAccountResources(t.Context(), acc, entriesToDelete)
	assert.Error(t, err)
	assert.Equal(t, 2, userDeleteCalled)
	assert.Equal(t, 1, accountPolicyDeleteCalled)
	assert.Equal(t, 1, environmentPolicyDeleteCalled)
}

func TestContinuesIfSingleEntriesFailToDelete(t *testing.T) {
	userDeleteCalled := 0
	groupDeleteCalled := 0
	accountPolicyDeleteCalled := 0
	environmentPolicyDeleteCalled := 0
	c := testClient{
		userFunc: func(ctx context.Context, accountUUID, email string) error {
			userDeleteCalled++
			if userDeleteCalled > 1 {
				return errors.New("fail")
			}
			return nil
		},
		groupFunc: func(ctx context.Context, accountUUID, name string) error {
			groupDeleteCalled++
			if groupDeleteCalled > 1 {
				return errors.New("fail")
			}
			return nil
		},
		accountPolicyFunc: func(ctx context.Context, name string) error {
			accountPolicyDeleteCalled++
			if accountPolicyDeleteCalled > 1 {
				return errors.New("fail")
			}
			return nil
		},
		environmentPolicyFunc: func(ctx context.Context, envID, name string) error {
			environmentPolicyDeleteCalled++
			if environmentPolicyDeleteCalled > 1 {
				return errors.New("fail")
			}
			return nil
		},
	}
	entriesToDelete := delete.Resources{
		Users: []delete.User{
			{
				Email: "test@user.com",
			},
			{
				Email: "another@user.com",
			},
		},
		Groups: []delete.Group{
			{
				Name: "test-group",
			},
			{
				Name: "other-group",
			},
		},
		AccountPolicies: []delete.AccountPolicy{
			{
				Name: "test-policy",
			},
			{
				Name: "test-policy-2",
			},
		},
		EnvironmentPolicies: []delete.EnvironmentPolicy{
			{
				Name:        "test-policy",
				Environment: "abc1234567",
			},
			{
				Name:        "test-policy",
				Environment: "def7654321",
			},
		},
	}
	acc := delete.Account{
		Name:      "name",
		UUID:      "1234",
		APIClient: &c,
	}
	err := delete.DeleteAccountResources(t.Context(), acc, entriesToDelete)
	assert.Error(t, err)
	assert.Equal(t, 2, userDeleteCalled)
	assert.Equal(t, 2, groupDeleteCalled)
	assert.Equal(t, 2, accountPolicyDeleteCalled)
	assert.Equal(t, 2, environmentPolicyDeleteCalled)
}
