//go:build account_integration

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

	stringutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
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

	defer func() {
		t.Log("Starting cleanup")
		err := monaco.Run(t, baseFs, "monaco account delete --manifest manifest.yaml --file delete.yaml")
		require.NoError(t, err)
	}()

	deploy1st := deploy(project, baseFs)
	download1st := download(project, afero.NewCopyOnWriteFs(baseFs, afero.NewMemMapFs()))

	allDeployedItemsDownloaded := true
	for _, u := range deploy1st.Users {
		if !assert.Contains(t, download1st.Users, u.Email.Value()) {
			allDeployedItemsDownloaded = false
		}
	}
	for _, deployedServiceUser := range deploy1st.ServiceUsers {
		_, found := assertElementInSlice(t, download1st.ServiceUsers, func(su account.ServiceUser) bool { return su.Name == deployedServiceUser.Name })
		if !found {
			allDeployedItemsDownloaded = false
		}
	}
	for _, p := range deploy1st.Policies {
		if !assert.Contains(t, download1st.Policies, toID(p.Name)) { // when downloading, ID is generated from name
			allDeployedItemsDownloaded = false
		}
	}
	for _, g := range deploy1st.Groups {
		if !assert.Contains(t, download1st.Groups, toID(g.Name)) { // when downloading, ID is generated from name
			allDeployedItemsDownloaded = false
		}
	}

	require.True(t, allDeployedItemsDownloaded, "Not all deployed items were downloaded")

	deploy2nd := deploy(project, baseFs)
	download2nd := download(project, afero.NewCopyOnWriteFs(baseFs, afero.NewMemMapFs()))
	assert.Equal(t, deploy2nd, deploy1st)

	redownloadedItemsAreIdentical := true
	for _, u := range deploy1st.Users {
		if !assert.Equal(t, download1st.Users[u.Email.Value()], download2nd.Users[u.Email.Value()]) {
			redownloadedItemsAreIdentical = false
		}
	}
	for _, deployedServiceUser := range deploy1st.ServiceUsers {
		e1, found1 := assertElementInSlice(t, download1st.ServiceUsers, func(su account.ServiceUser) bool { return su.Name == deployedServiceUser.Name })
		e2, found2 := assertElementInSlice(t, download2nd.ServiceUsers, func(su account.ServiceUser) bool { return su.Name == deployedServiceUser.Name })
		if !found1 || !found2 || !assert.Equal(t, *e1, *e2) {
			redownloadedItemsAreIdentical = false
		}
	}
	for _, p := range deploy1st.Policies {
		p.ID = toID(p.Name)
		if !assert.Equal(t, download1st.Policies[p.ID], download2nd.Policies[p.ID]) {
			redownloadedItemsAreIdentical = false
		}
	}
	for _, g := range deploy1st.Groups {
		g.ID = toID(g.Name)
		if !assert.Equal(t, download1st.Groups[g.ID], download2nd.Groups[g.ID]) {
			redownloadedItemsAreIdentical = false
		}
	}

	require.True(t, redownloadedItemsAreIdentical, "Not all redownloaded items were identical")
}
