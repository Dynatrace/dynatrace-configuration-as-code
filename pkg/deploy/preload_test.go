/**
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

package deploy

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

var dummyConfigClient = &dtclient.DummyConfigClient{}
var dummySettingsClient = &dtclient.DummySettingsClient{}
var clientsetEnv1 = &client.ClientSet{ConfigClient: dummyConfigClient, SettingsClient: dummySettingsClient}

//var clientsetEnv2 = &client.ClientSet{ConfigClient: dummyConfigClient, SettingsClient: dummySettingsClient}

func Test_gatherPreloadConfigTypeEntries_OneEntryPerConfigType(t *testing.T) {
	entries := gatherPreloadConfigTypeEntries(
		[]project.Project{
			{
				Id:      "projectID",
				GroupId: "groupID",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "alertingProfile2", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
						},
					},
				},
			},
		},
		"env1",
	)

	require.Len(t, entries, 1)
	assert.Equal(t, entries[0], config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	})
}

func Test_gatherPreloadConfigTypeEntries_OneEntryForEachConfigType(t *testing.T) {
	entries := gatherPreloadConfigTypeEntries(
		[]project.Project{
			{
				Id:      "projectID",
				GroupId: "groupID",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
						},
						"management-zone": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "managementZone1", Type: "management-zone"},
								Type: config.ClassicApiType{
									Api: "management-zone",
								}},
						},
					},
				},
			},
		},
		"env1",
	)

	require.Len(t, entries, 2)
	assert.ElementsMatch(t, entries, []config.Type{
		config.SettingsType{SchemaId: "builtin:alerting.profile"},
		config.ClassicApiType{Api: "management-zone"},
	})
}

func Test_gatherPreloadConfigTypeEntries_EntriesOnlyForSupportedConfigTypes(t *testing.T) {
	entries := gatherPreloadConfigTypeEntries(
		[]project.Project{
			{
				Id:      "projectID",
				GroupId: "groupID",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
						},
						"workflow": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "workflow1", Type: "workflow"},
								Type: config.AutomationType{
									Resource: config.Workflow,
								}},
						},
					},
				},
			},
		},
		"env1",
	)

	require.Len(t, entries, 1)
	assert.Equal(t, entries, []config.Type{config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
}

func Test_gatherPreloadConfigTypeEntries_OneEntryForSameEnvironmentInDifferentProjects(t *testing.T) {
	entries := gatherPreloadConfigTypeEntries(
		[]project.Project{
			{
				Id:      "projectID1",
				GroupId: "groupID",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID1", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
						},
					},
				},
			},
			{
				Id:      "projectID2",
				GroupId: "groupID",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID2", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
						},
					},
				},
			},
		},
		"env1",
	)

	assert.Equal(t, entries, []config.Type{config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
}

func Test_CacheSettingsFails(t *testing.T) {
	expectedErrMsg := "error happened"
	logOutput := strings.Builder{}
	log.PrepareLogging(t.Context(), afero.NewMemMapFs(), false, &logOutput, false, false)

	dtClient := client.NewMockSettingsClient(gomock.NewController(t))
	dtClient.EXPECT().Cache(gomock.Any(), gomock.Any()).Times(1).Return(errors.New(expectedErrMsg))
	projects := []project.Project{
		{
			Id:      "projectID",
			GroupId: "groupID",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env1": project.ConfigsPerType{
					"dashboard-share-settings": {
						{
							Coordinate: coordinate.Coordinate{
								Project:  "projectID",
								ConfigId: "dashboard-share-settings",
								Type:     "dashboard-share-settings",
							},
							Type: config.SettingsType{
								SchemaId:      "my-schema-id",
								SchemaVersion: "1.0.0",
							},
						},
					},
				},
			},
		},
	}
	clientSet := &client.ClientSet{SettingsClient: dtClient}
	preloadCaches(t.Context(), projects, clientSet, "env1")
	assert.Contains(t, logOutput.String(), expectedErrMsg)
}

func Test_CacheSettingsSucceeds(t *testing.T) {
	logOutput := strings.Builder{}
	log.PrepareLogging(t.Context(), afero.NewMemMapFs(), true, &logOutput, false, false)

	dtClient := client.NewMockSettingsClient(gomock.NewController(t))
	dtClient.EXPECT().Cache(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	projects := []project.Project{
		{
			Id:      "projectID",
			GroupId: "groupID",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env1": project.ConfigsPerType{
					"dashboard-share-settings": {
						{
							Coordinate: coordinate.Coordinate{
								Project:  "projectID",
								ConfigId: "dashboard-share-settings",
								Type:     "dashboard-share-settings",
							},
							Type: config.SettingsType{
								SchemaId:      "my-schema-id",
								SchemaVersion: "1.0.0",
							},
						},
					},
				},
			},
		},
	}
	clientSet := &client.ClientSet{SettingsClient: dtClient}
	preloadCaches(t.Context(), projects, clientSet, "env1")
	assert.NotContains(t, logOutput.String(), "warn")
	assert.Contains(t, logOutput.String(), "Cached")
}

func Test_CacheClassicSucceeds(t *testing.T) {
	logOutput := strings.Builder{}
	log.PrepareLogging(t.Context(), afero.NewMemMapFs(), true, &logOutput, false, false)

	dtClient := client.NewMockConfigClient(gomock.NewController(t))
	dtClient.EXPECT().Cache(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	projects := []project.Project{
		{
			Id:      "projectID",
			GroupId: "groupID",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env1": project.ConfigsPerType{
					api.Dashboard: {
						{
							Coordinate: coordinate.Coordinate{
								Project:  "projectID",
								ConfigId: "dashboard",
								Type:     "dashboard",
							},
							Type: config.ClassicApiType{
								Api: "dashboard",
							},
						},
					},
				},
			},
		},
	}
	clientSet := &client.ClientSet{ConfigClient: dtClient}
	preloadCaches(t.Context(), projects, clientSet, "env1")
	output := logOutput.String()
	assert.NotContains(t, output, "warn")
	assert.Contains(t, output, "Cached")
}

func Test_CacheClassicFails(t *testing.T) {
	logOutput := strings.Builder{}
	expectedErr := "my error"
	log.PrepareLogging(t.Context(), afero.NewMemMapFs(), true, &logOutput, false, false)

	dtClient := client.NewMockConfigClient(gomock.NewController(t))
	dtClient.EXPECT().Cache(gomock.Any(), gomock.Any()).Times(1).Return(errors.New(expectedErr))
	projects := []project.Project{
		{
			Id:      "projectID",
			GroupId: "groupID",
			Configs: project.ConfigsPerTypePerEnvironments{
				"env1": project.ConfigsPerType{
					api.Dashboard: {
						{
							Coordinate: coordinate.Coordinate{
								Project:  "projectID",
								ConfigId: "dashboard",
								Type:     "dashboard",
							},
							Type: config.ClassicApiType{
								Api: "dashboard",
							},
						},
					},
				},
			},
		},
	}
	clientSet := &client.ClientSet{ConfigClient: dtClient}
	preloadCaches(t.Context(), projects, clientSet, "env1")
	output := logOutput.String()
	assert.Contains(t, output, expectedErr)
}

func Test_ScopedConfigsAreNotCached(t *testing.T) {
	dtClient := client.NewMockConfigClient(gomock.NewController(t)) //<- dont expect any call(s) on the mocked client
	type args struct {
		projects    []project.Project
		clientSet   *client.ClientSet
		environment string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			args: args{
				projects: []project.Project{
					{
						Id:      "projectID",
						GroupId: "groupID",
						Configs: project.ConfigsPerTypePerEnvironments{
							"env1": project.ConfigsPerType{
								"dashboard-share-settings": {
									{
										Coordinate: coordinate.Coordinate{
											Project:  "projectID",
											ConfigId: "dashboard-share-settings",
											Type:     "dashboard-share-settings"},
										Type: config.ClassicApiType{
											Api: "dashboard-share-settings", //<- scoped config
										},
									},
								},
							},
						},
					},
				},
				clientSet:   &client.ClientSet{ConfigClient: dtClient},
				environment: "env1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadCaches(t.Context(), tt.args.projects, tt.args.clientSet, tt.args.environment)
		})
	}
}

func Test_gatherPreloadConfigTypeEntries_WithSkipParam(t *testing.T) {
	entries := gatherPreloadConfigTypeEntries(
		[]project.Project{
			{
				Id:      "projectID",
				GroupId: "groupID",
				Configs: project.ConfigsPerTypePerEnvironments{
					"env1": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
								Skip: true,
							},
						},
						"dashboard-share-settings": {
							{
								Coordinate: coordinate.Coordinate{
									Project:  "projectID",
									ConfigId: "dashboard-share-settings",
									Type:     "dashboard-share-settings"},
								Type: config.ClassicApiType{
									Api: "dashboard-share-settings",
								},
							},
						},
						"management-zone": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "managementZone1", Type: "management-zone"},
								Type: config.ClassicApiType{
									Api: "management-zone",
								},
								Skip: false,
							},
						},
					},
				},
			},
		},
		"env1",
	)

	require.ElementsMatch(t, entries, []config.Type{config.ClassicApiType{Api: "dashboard-share-settings"}, config.ClassicApiType{Api: "management-zone"}})
}
