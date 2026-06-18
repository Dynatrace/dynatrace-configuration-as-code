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

package classic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

func TestDownload_DisabledSloReusesListPayload(t *testing.T) {
	sloAPI := api.NewAPIs()[api.Slo]

	customFields := map[string]any{
		"id":      "slo-id",
		"name":    "my disabled slo",
		"enabled": false,
	}

	c := client.NewMockConfigClient(gomock.NewController(t))
	// Get must not be called for a disabled SLO - its payload is re-used from the List response
	c.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	v := value{value: dtclient.Value{Id: "slo-id", Name: "my disabled slo", CustomFields: customFields}}

	got, err := download(t.Context(), c, sloAPI, v)

	require.NoError(t, err)
	assert.Equal(t, []map[string]any{customFields}, got)
}

func TestDownload_EnabledSloIsFetchedViaGet(t *testing.T) {
	sloAPI := api.NewAPIs()[api.Slo]

	c := client.NewMockConfigClient(gomock.NewController(t))
	c.EXPECT().Get(gomock.Any(), gomock.Any(), "slo-id").
		Return([]byte(`{"id":"slo-id","name":"my enabled slo","enabled":true}`), nil)

	// CustomFields is nil for an enabled SLO, so it must be fetched via Get
	v := value{value: dtclient.Value{Id: "slo-id", Name: "my enabled slo"}}

	got, err := download(t.Context(), c, sloAPI, v)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "slo-id", got[0]["id"])
	assert.Equal(t, true, got[0]["enabled"])
}

func TestDownload_CustomFieldsAreIgnoredForNonSloApi(t *testing.T) {
	// Even when CustomFields is set, a non-SLO API must always be fetched via Get
	dashboardAPI := api.API{ID: "dashboard", URLPath: "/api/config/v1/dashboards"}

	c := client.NewMockConfigClient(gomock.NewController(t))
	c.EXPECT().Get(gomock.Any(), gomock.Any(), "config-id").
		Return([]byte(`{"id":"config-id","name":"my dashboard"}`), nil)

	v := value{value: dtclient.Value{
		Id:           "config-id",
		Name:         "my dashboard",
		CustomFields: map[string]any{"id": "config-id", "enabled": false},
	}}

	got, err := download(t.Context(), c, dashboardAPI, v)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "config-id", got[0]["id"])
}
