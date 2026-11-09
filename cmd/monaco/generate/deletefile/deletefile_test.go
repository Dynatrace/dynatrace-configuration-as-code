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

package deletefile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

func newCredentialVaultConfig(nameParam parameter.Parameter) config.Config {
	params := config.Parameters{}
	if nameParam != nil {
		params[config.NameParameter] = nameParam
	}

	return config.Config{
		Coordinate: coordinate.Coordinate{Project: "project", Type: string(config.CredentialVaultID), ConfigId: "credential"},
		Type:       config.CredentialVault{},
		Parameters: params,
	}
}

func TestCreateDeleteEntry_CredentialVault_IdentifyByName(t *testing.T) {
	entry, err := createDeleteEntry(newCredentialVaultConfig(value.New("My credential")), api.NewAPIs(), project.Project{})
	require.NoError(t, err)

	// project and id must not be set, as the delete file loader rejects them for name-identified types
	assert.Equal(t, persistence.DeleteEntry{
		Type:       string(config.CredentialVaultID),
		ConfigName: "My credential",
	}, entry)
}

func TestCreatDeleteEntry_CredentialVault_Errors(t *testing.T) {
	invalidNames := []struct {
		name      string
		nameParam parameter.Parameter
	}{
		{name: "missing name", nameParam: nil},
		{name: "empty name", nameParam: value.New("")},
		{name: "non-string name", nameParam: value.New(42)},
		{name: "reference name", nameParam: reference.New("project", "credential-vault-v2", "other", "name")},
	}

	for _, tt := range invalidNames {
		t.Run(tt.name+" fails", func(t *testing.T) {
			_, err := createDeleteEntry(newCredentialVaultConfig(tt.nameParam), api.NewAPIs(), project.Project{})
			assert.Error(t, err)
		})
	}
}
