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
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dtClientEnv1 = &dtclient.DummyClient{}
var dtClientEnv2 = &dtclient.DummyClient{}

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
		dynatrace.EnvironmentClients{dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClientEnv1}},
	)

	require.Len(t, entries, 1)
	assert.Equal(t, entries[0].client, dtClientEnv1)
	assert.Equal(t, entries[0].configType, config.SettingsType{
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
		dynatrace.EnvironmentClients{dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClientEnv1}},
	)

	require.Len(t, entries, 2)
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv1, configType: config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv1, configType: config.ClassicApiType{
		Api: "management-zone",
	}})
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
		dynatrace.EnvironmentClients{dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClientEnv1}},
	)

	require.Len(t, entries, 1)
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv1, configType: config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
}

func Test_gatherPreloadConfigTypeEntries_OneEntryForEachEnvironmentInSameProject(t *testing.T) {
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
					},
					"env2": project.ConfigsPerType{
						"builtin:alerting.profile": {
							{
								Coordinate: coordinate.Coordinate{Project: "projectID", ConfigId: "alertingProfile1", Type: "builtin:alerting.profile"},
								Type: config.SettingsType{
									SchemaId: "builtin:alerting.profile",
								},
							},
						},
					},
				},
			},
		},
		dynatrace.EnvironmentClients{
			dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClientEnv1},
			dynatrace.EnvironmentInfo{Name: "env2"}: &client.ClientSet{DTClient: dtClientEnv2},
		},
	)

	require.Len(t, entries, 2)
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv1, configType: config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv2, configType: config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
}

func Test_gatherPreloadConfigTypeEntries_OneEntryForEachEnvironmentInDifferentProjects(t *testing.T) {
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
					"env2": project.ConfigsPerType{
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
		dynatrace.EnvironmentClients{
			dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClientEnv1},
			dynatrace.EnvironmentInfo{Name: "env2"}: &client.ClientSet{DTClient: dtClientEnv2},
		},
	)

	require.Len(t, entries, 2)
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv1, configType: config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv2, configType: config.SettingsType{
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
		dynatrace.EnvironmentClients{
			dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClientEnv1},
		},
	)

	require.Len(t, entries, 1)
	assert.Contains(t, entries, preloadConfigTypeEntry{client: dtClientEnv1, configType: config.SettingsType{
		SchemaId: "builtin:alerting.profile",
	}})
}

func Test_gatherPreloadConfigTypeEntries_NoEntryIfEnvironmentMissingClient(t *testing.T) {
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
					},
				},
			},
		},
		dynatrace.EnvironmentClients{dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: nil}},
	)

	assert.Len(t, entries, 0)
}

func Test_ScopedConfigsAreNotCached(t *testing.T) {
	dtClient := client.NewMockDynatraceClient(gomock.NewController(t)) //<- dont expect any call(s) on the mocked client
	type args struct {
		projects           []project.Project
		environmentClients dynatrace.EnvironmentClients
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
				environmentClients: dynatrace.EnvironmentClients{dynatrace.EnvironmentInfo{Name: "env1"}: &client.ClientSet{DTClient: dtClient}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadCaches(tt.args.projects, tt.args.environmentClients)
		})
	}
}
