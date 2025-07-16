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

package loader

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
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

func parseSingleAccount(c *Context, a persistence.Account) (manifest.Account, error) {

	accountUUID, err := parseAccountUUID(c, a.AccountUUID)
	if err != nil {
		return manifest.Account{}, err
	}

	oAuthDef, err := parseOAuth(c, &a.OAuth)
	if err != nil {
		return manifest.Account{}, fmt.Errorf("oAuth is invalid: %w", err)
	}

	var urlDef *manifest.URLDefinition
	if a.ApiUrl != nil {
		if u, err := parseURLDefinition(c, *a.ApiUrl); err != nil {
			return manifest.Account{}, fmt.Errorf("apiUrl: %w", err)
		} else {
			urlDef = &u
		}
	}

	acc := manifest.Account{
		Name:        a.Name,
		AccountUUID: accountUUID,
		ApiUrl:      urlDef,
		OAuth:       *oAuthDef,
	}

	return acc, nil
}

func parseAccountUUID(c *Context, u persistence.TypedValue) (uuid.UUID, error) {
	if u.Value == "" {
		return uuid.UUID{}, errAccUidMissing
	}

	if u.Type == "" || u.Type == persistence.TypeValue { // shorthand or explicit type: value
		return parseUUID(u.Value)
	}

	if u.Type != persistence.TypeEnvironment {
		return uuid.UUID{}, fmt.Errorf("unexpected type: %q (expected one of %q, %q)", u.Type, persistence.TypeValue, persistence.TypeEnvironment)
	}

	if c.Opts.DoNotResolveEnvVars {
		log.Debug("Skipped resolving environment variable %s based on loader options", u.Value)
		return uuid.UUID{}, nil
	}

	val, found := os.LookupEnv(u.Value)
	if !found {
		return uuid.UUID{}, fmt.Errorf("environment variable %q could not be found", u.Value)
	}
	if val == "" {
		return uuid.UUID{}, fmt.Errorf("environment variable %q is defined but has no value", u.Value)
	}
	return parseUUID(val)
}

func parseUUID(value string) (uuid.UUID, error) {
	u, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, invalidUUIDError{value, err}
	}

	return u, nil
}

// parseAccounts converts the persistence definition to the in-memory definition
func parseAccounts(c *Context, accounts []persistence.Account) (map[string]manifest.Account, error) {

	result := make(map[string]manifest.Account)

	for i, a := range accounts {
		if c.Account != "" && c.Account != a.Name {
			// skipped
			continue
		}
		if a.Name == "" {
			return nil, fmt.Errorf("failed to parse account on position %d: %w", i, errNameMissing)
		}

		acc, err := parseSingleAccount(c, a)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account %q: %w", a.Name, err)
		}

		result[acc.Name] = acc
	}

	if len(result) == 0 && c.Account != "" {
		return nil, fmt.Errorf("required account '%s' was not found", c.Account)
	}

	return result, nil
}
