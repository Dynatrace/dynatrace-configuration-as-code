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

package manifest

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
)

var (
	errNameMissing   = errors.New("name is missing")
	errAccUidMissing = errors.New("accountUUID is missing")
)

type invalidUUIDError struct {
	uuid string
	err  error
}

func (e invalidUUIDError) Error() string {
	return fmt.Sprintf("invalid uuid %q: %s", e.uuid, e.err)
}

func (e invalidUUIDError) Unwrap() error {
	return e.err
}

func convertSingleAccount(c *LoaderContext, a account) (Account, error) {

	if a.AccountUUID == "" {
		return Account{}, errAccUidMissing
	}

	accountId, err := uuid.Parse(a.AccountUUID)
	if err != nil {
		return Account{}, invalidUUIDError{a.AccountUUID, err}
	}

	oAuthDef, err := parseOAuth(c, a.OAuth)
	if err != nil {
		return Account{}, fmt.Errorf("oAuth is invalid: %w", err)
	}

	var urlDef *URLDefinition
	if a.ApiUrl != nil {
		if u, err := parseURLDefinition(c, *a.ApiUrl); err != nil {
			return Account{}, fmt.Errorf("apiUrl: %w", err)
		} else {
			urlDef = &u
		}
	}

	acc := Account{
		Name:        a.Name,
		AccountUUID: accountId,
		ApiUrl:      urlDef,
		OAuth:       oAuthDef,
	}

	return acc, nil
}

// convertAccounts converts the persistence definition to the in-memory definition
func convertAccounts(c *LoaderContext, accounts []account) (map[string]Account, error) {

	result := make(map[string]Account, len(accounts))

	for i, a := range accounts {
		if a.Name == "" {
			return nil, fmt.Errorf("failed to parse account on position %d: %w", i, errNameMissing)
		}

		acc, err := convertSingleAccount(c, a)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account %q: %w", a.Name, err)
		}

		result[acc.Name] = acc
	}

	return result, nil
}
