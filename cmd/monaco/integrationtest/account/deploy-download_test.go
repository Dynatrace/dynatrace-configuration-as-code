//go:build integration

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package account

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	stringutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/loader"
)

func TestIdempotenceOfDeployment(t *testing.T) {

	deploy := func(project string, fs afero.Fs) *account.Resources {
		err := monaco.Run(t, fs, fmt.Sprintf("monaco account deploy --project %s --verbose", project))

		require.NoError(t, err)

		r, err := loader.Load(fs, project)
		require.NoError(t, err)

		return r
	}
	download := func(project string, fs afero.Fs) *account.Resources {
		err := monaco.Run(t, fs, fmt.Sprintf("monaco account download --project %s --output-folder output --verbose", project))
		require.NoError(t, err)

		r, err := loader.Load(fs, fmt.Sprintf("%s/%s/%s", "output", project, "test-account"))
		require.NoError(t, err)

		return r
	}
	toID := stringutils.Sanitize
	project := "add_user"

	createMZone(t)
	baseFs := afero.NewCopyOnWriteFs(afero.NewBasePathFs(afero.NewOsFs(), "resources/deploy-download"), afero.NewMemMapFs())

	randomString := strconv.Itoa(rand.Int())
	randomizeConfiguration(t, baseFs, project, randomString)
	randomizeConfiguration(t, baseFs, "delete.yaml", randomString)
	baseFs = afero.NewReadOnlyFs(baseFs)

	deploy1st := deploy(project, baseFs)
	download1st := download(project, afero.NewCopyOnWriteFs(baseFs, afero.NewMemMapFs()))

	for _, u := range deploy1st.Users {
		assert.Contains(t, download1st.Users, u.Email.Value())
	}
	for _, p := range deploy1st.Policies {
		assert.Contains(t, download1st.Policies, toID(p.Name)) // when downloading, ID is generated from name
	}
	for _, g := range deploy1st.Groups {
		assert.Contains(t, download1st.Groups, toID(g.Name)) // when downloading, ID is generated from name
	}

	deploy2nd := deploy(project, baseFs)
	download2nd := download(project, afero.NewCopyOnWriteFs(baseFs, afero.NewMemMapFs()))
	assert.Equal(t, deploy2nd, deploy1st)

	for _, u := range deploy1st.Users {
		assert.Equal(t, download1st.Users[u.Email.Value()], download2nd.Users[u.Email.Value()])
	}
	for _, p := range deploy1st.Policies {
		p.ID = toID(p.Name)
		assert.Equal(t, download1st.Policies[p.ID], download2nd.Policies[p.ID])
	}
	for _, g := range deploy1st.Groups {
		g.ID = toID(g.Name)
		assert.Equal(t, deploy1st.Groups[g.ID], deploy2nd.Groups[g.ID])
	}

	err := monaco.Run(t, baseFs, "monaco account delete --manifest manifest.yaml --file delete.yaml")
	require.NoError(t, err)
}
