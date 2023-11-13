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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testClient struct {
	userFunc   func(ctx context.Context, accountUUID, email string) error
	groupFunc  func(ctx context.Context, accountUUID, name string) error
	policyFunc func(ctx context.Context, levelType, levelID, name string) error
}

func (c *testClient) DeleteUser(ctx context.Context, accountUUID, email string) error {
	return c.userFunc(ctx, accountUUID, email)
}

func (c *testClient) DeleteGroup(ctx context.Context, accountUUID, name string) error {
	return c.groupFunc(ctx, accountUUID, name)
}

func (c *testClient) DeletePolicy(ctx context.Context, levelType, levelID, name string) error {
	return c.policyFunc(ctx, levelType, levelID, name)
}

func TestDeletesResources(t *testing.T) {
	userDeleteCalled := 0
	groupDeleteCalled := 0
	policyDeleteCalled := 0
	c := testClient{
		userFunc: func(ctx context.Context, accountUUID, email string) error {
			userDeleteCalled++
			return nil
		},
		groupFunc: func(ctx context.Context, accountUUID, name string) error {
			groupDeleteCalled++
			return nil
		},
		policyFunc: func(ctx context.Context, levelType, levelID, name string) error {
			policyDeleteCalled++
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
	err := delete.AccountResources(context.TODO(), &c, "1234", entriesToDelete)
	assert.NoError(t, err)
	assert.Equal(t, 2, userDeleteCalled)
	assert.Equal(t, 1, groupDeleteCalled)
	assert.Equal(t, 2, policyDeleteCalled)
}

func TestContinuesDeletionIfOneTypeFails(t *testing.T) {
	userDeleteCalled := 0
	policyDeleteCalled := 0
	c := testClient{
		userFunc: func(ctx context.Context, accountUUID, email string) error {
			userDeleteCalled++
			return nil
		},
		groupFunc: func(ctx context.Context, accountUUID, name string) error {
			return errors.New("fail")
		},
		policyFunc: func(ctx context.Context, levelType, levelID, name string) error {
			policyDeleteCalled++
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
	err := delete.AccountResources(context.TODO(), &c, "1234", entriesToDelete)
	assert.Error(t, err)
	assert.Equal(t, 2, userDeleteCalled)
	assert.Equal(t, 2, policyDeleteCalled)
}

func TestContinuesIfSingleEntriesFailToDelete(t *testing.T) {
	userDeleteCalled := 0
	groupDeleteCalled := 0
	policyDeleteCalled := 0
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
		policyFunc: func(ctx context.Context, levelType, levelID, name string) error {
			policyDeleteCalled++
			if policyDeleteCalled > 1 {
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
		},
	}
	err := delete.AccountResources(context.TODO(), &c, "1234", entriesToDelete)
	assert.Error(t, err)
	assert.Equal(t, 2, userDeleteCalled)
	assert.Equal(t, 2, groupDeleteCalled)
	assert.Equal(t, 3, policyDeleteCalled)
}
