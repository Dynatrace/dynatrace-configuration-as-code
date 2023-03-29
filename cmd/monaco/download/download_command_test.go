// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build unit

package download

import (
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func TestGetDownloadCommand_directDownload(t *testing.T) {
	t.Run("Authorization via token", func(t *testing.T) {
		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any()).Return(nil)

		err := m.download("direct http://some.url --token TOKEN")
		assert.NoError(t, err)
	})
	t.Run("Token is missing", func(t *testing.T) {
		err := newMonaco(t).download("direct http://some.url")

		assert.EqualError(t, err, "token credentials not provided")
	})
	t.Run("Authorization via OAuth", func(t *testing.T) {
		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any()).Return(nil)

		err := m.download("direct http://some.url --token TOKEN --oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET --oauth-token-endpoint TOKEN_ENDOINT")
		assert.NoError(t, err)
	})
	t.Run("Clint ID for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("direct http://some.url --token TOKEN --oauth-client-secret CLIENT_SECRET --oauth-token-endpoint TOKEN_ENDOINT")
		assert.ErrorContains(t, err, "OAuth clientID credentials not provided")
	})
	t.Run("Clint secret for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("direct http://some.url --token TOKEN --oauth-client-id CLIENT_ID --oauth-token-endpoint TOKEN_ENDOINT")
		assert.ErrorContains(t, err, "OAuth client secret credentials not provided")
	})
	t.Run("no argument provided", func(t *testing.T) {
		err := newMonaco(t).download("direct --token TOKEN --oauth-client-id CLIENT_ID --oauth-token-endpoint TOKEN_ENDOINT")
		assert.ErrorContains(t, err, "url have to be provided as positional argument")
	})
	t.Run("no specific apis rovided", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "http://some.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "test",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{},
				specificSchemas: []string{},
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct http://some.url --token token --project test")
		assert.NoError(t, err)
	})
	t.Run("default project provided", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "http://some.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "project",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{},
				specificSchemas: []string{},
			},
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct http://some.url --token token")
		assert.NoError(t, err)
	})
	t.Run("skip download of settings", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "project",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{},
				specificSchemas: []string{},
				onlyAPIs:        true,
				onlySettings:    false,
			},
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --only-apis")
		assert.NoError(t, err)
	})
	t.Run("skip download of APIs", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "project",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{},
				specificSchemas: []string{},
				onlyAPIs:        false,
				onlySettings:    true,
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --only-settings")
		assert.NoError(t, err)
	})
	t.Run("with specific apis (multiple flags)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "test",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{"test", "test2"},
				specificSchemas: []string{},
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --project test --api test --api test2")
		assert.NoError(t, err)
	})
	t.Run("with specific apis (single flag)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "test",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{"test", "test2"},
				specificSchemas: []string{},
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --project test --api test,test2")
		assert.NoError(t, err)
	})
	t.Run("specific apis (mixed flags)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "test",
					outputFolder:   "",
					forceOverwrite: false,
				},
				specificAPIs:    []string{"test", "test2", "test3"},
				specificSchemas: []string{},
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --project test --api test,test2 --api test3")
		assert.NoError(t, err)
	})
	t.Run("specific settings (single flag)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName: "test",
				},
				specificAPIs:    []string{},
				specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications"},
			},
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --project test --settings-schema builtin:alerting.profile,builtin:problem.notifications")
		assert.NoError(t, err)
	})
	t.Run("specific settings (mixed flags)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName: "test",
				},
				specificAPIs:    []string{},
				specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications", "builtin:metric.metadata"},
			},
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --project test --settings-schema builtin:alerting.profile,builtin:problem.notifications --settings-schema builtin:metric.metadata")
		assert.NoError(t, err)
	})
	t.Run("with outputfolder", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "project",
					outputFolder:   "myDownloads",
					forceOverwrite: false,
				},
				specificAPIs:    []string{},
				specificSchemas: []string{},
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --output-folder myDownloads")
		assert.NoError(t, err)
	})
	t.Run("output-folder and force overwrite", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			envVarName:     "token",
			downloadCmdOptions: downloadCmdOptions{
				sharedDownloadCmdOptions: sharedDownloadCmdOptions{
					projectName:    "project",
					outputFolder:   "myDownloads",
					forceOverwrite: true,
				},
				specificAPIs:    []string{},
				specificSchemas: []string{},
			},
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("direct test.url --token token --output-folder myDownloads --force")
		assert.NoError(t, err)
	})
}

type monaco struct {
	*MockCommand
}

func newMonaco(t *testing.T) *monaco {
	return &monaco{NewMockCommand(gomock.NewController(t))}
}

func (monaco monaco) download(bashCmd string) error {
	cmd := GetDownloadCommand(afero.NewOsFs(), monaco.MockCommand)
	cmd.SetArgs(strings.Split(bashCmd, " "))
	cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message

	return cmd.Execute()
}

// TestInvalidCliCommands is a very basic test testing that invalid commands error.
// It is not the goal to test the exact message that cobra generates, except if we supply the message.
// Otherwise, we would run into issues upon upgrading.
// On th other hand, we could use the exact message to review the exact messages customers will see.
func TestInvalidCliCommands(t *testing.T) {
	t.Setenv("MONACO_FEAT_ENTITIES", "1")

	tests := []struct {
		name                  string
		args                  string
		errorContainsExpected []string
	}{
		{
			"no arguments provided",
			"",
			[]string{"sub-command is required"},
		},
		{
			"manifest provided but missing specific environment",
			"manifest manifest.yaml",
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"manifest is missing but environment is provider",
			"manifest some_env",
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		// ENTITIES
		{
			"entities: no arguments provided to direct download",
			"entities direct",
			[]string{"url and token have to be provided as positional argument"},
		},
		{
			"entities: url is missing other required argument",
			"entities direct some.env.url.com",
			[]string{"url and token have to be provided as positional argument"},
		},
		{
			"entities: manifest provided but missing specific environment",
			"entities manifest manifest.yaml",
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"entities: manifest is missing but environment is provider",
			"entities manifest some_env",
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"unknown flag",
			"--test",
			[]string{"--test"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			// for all test cases there should be at least an error
			assert.Error(t, err)

			// for most cases we can test the message in more detail
			for _, expected := range test.errorContainsExpected {
				assert.ErrorContains(t, err, expected)
			}

			// for testing not to forget adding expectations
			assert.NotEmpty(t, test.errorContainsExpected, "no error conditions specified")
		})
	}
}

func TestValidCommands(t *testing.T) {
	t.Setenv("MONACO_FEAT_ENTITIES", "1")

	tests := []struct {
		name  string
		args  string
		setup func(command *MockCommand)
	}{
		// CONFIGS
		{
			"manifest download no specific apis",
			"manifest test.yaml test_env",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download - skip download of settings ",
			"manifest test.yaml test_env --only-apis",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{},
						specificSchemas: []string{},
						onlyAPIs:        true,
						onlySettings:    false,
					},
				})
			},
		},
		{
			"manifest download - skip download of APIs ",
			"manifest test.yaml test_env --only-settings",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{},
						specificSchemas: []string{},
						onlyAPIs:        false,
						onlySettings:    true,
					},
				})
			},
		},
		{
			"manifest download with specific apis (multiple flags)",
			"manifest test.yaml test_env --api test --api test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{"test", "test2"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with specific apis (single flag)",
			"manifest test.yaml test_env --api test,test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{"test", "test2"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with specific apis (mixed flags)",
			"manifest test.yaml test_env --api test,test2 --api test3",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{"test", "test2", "test3"},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with specific apis (single flag)",
			"manifest test.yaml test_env --settings-schema builtin:alerting.profile,builtin:problem.notifications",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName: "project",
						},
						specificAPIs:    []string{},
						specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications"},
					},
				})
			},
		},
		{
			"manifest download with specific apis (mixed flags)",
			"manifest test.yaml test_env --settings-schema builtin:alerting.profile,builtin:problem.notifications --settings-schema builtin:metric.metadata",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName: "project",
						},
						specificAPIs:    []string{},
						specificSchemas: []string{"builtin:alerting.profile", "builtin:problem.notifications", "builtin:metric.metadata"},
					},
				})
			},
		},
		{
			"manifest download with project",
			"manifest test.yaml test_env --project testproject",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "testproject",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with outputfolder",
			"manifest test.yaml test_env --output-folder myDownloads",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "myDownloads",
							forceOverwrite: false,
						},
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		{
			"manifest download with output-folder and force overwrite",
			"manifest test.yaml test_env --output-folder myDownloads --force",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), manifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					downloadCmdOptions: downloadCmdOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "myDownloads",
							forceOverwrite: true,
						},
						specificAPIs:    []string{},
						specificSchemas: []string{},
					},
				})
			},
		},
		// ENTITIES
		{
			"entities direct download",
			"entities direct test.url token --project test",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntities(gomock.Any(), entitiesDirectDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "test",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities direct download with default project",
			"entities direct test.url token",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntities(gomock.Any(), entitiesDirectDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities direct download with outputfolder",
			"entities direct test.url token --output-folder myDownloads",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntities(gomock.Any(), entitiesDirectDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "myDownloads",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities direct download with output-folder and force overwrite",
			"entities direct test.url token --output-folder myDownloads --force",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntities(gomock.Any(), entitiesDirectDownloadOptions{
					environmentUrl: "test.url",
					envVarName:     "token",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "myDownloads",
							forceOverwrite: true,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities manifest download",
			"entities manifest test.yaml test_env",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntitiesBasedOnManifest(gomock.Any(), entitiesManifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities manifest download with project",
			"entities manifest test.yaml test_env --project testproject",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntitiesBasedOnManifest(gomock.Any(), entitiesManifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "testproject",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities manifest download with outputfolder",
			"entities manifest test.yaml test_env --output-folder myDownloads",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntitiesBasedOnManifest(gomock.Any(), entitiesManifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "myDownloads",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities manifest download with output-folder and force overwrite",
			"entities manifest test.yaml test_env --output-folder myDownloads --force",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntitiesBasedOnManifest(gomock.Any(), entitiesManifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "myDownloads",
							forceOverwrite: true,
						},
						specificEntitiesTypes: []string{},
					},
				})
			},
		},
		{
			"entities manifest download with specific types",
			"entities manifest test.yaml test_env --specific-types HOST,SERVICE",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntitiesBasedOnManifest(gomock.Any(), entitiesManifestDownloadOptions{
					manifestFile:            "test.yaml",
					specificEnvironmentName: "test_env",
					entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
						sharedDownloadCmdOptions: sharedDownloadCmdOptions{
							projectName:    "project",
							outputFolder:   "",
							forceOverwrite: false,
						},
						specificEntitiesTypes: []string{"HOST", "SERVICE"},
					},
				})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)
			test.setup(commandMock)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			assert.NoError(t, err, "no error expected")
		})
	}
}

// TestInvalidCliCommands is a very basic test testing that invalid commands error.
// It is not the goal to test the exact message that cobra generates, except if we supply the message.
// Otherwise, we would run into issues upon upgrading.
// On th other hand, we could use the exact message to review the exact messages customers will see.
func TestDisabledCommands(t *testing.T) {
	t.Setenv("MONACO_FEAT_ENTITIES", "")

	tests := []struct {
		name                  string
		args                  string
		errorContainsExpected []string
	}{
		{
			"entities but is env. var is disabled",
			"entities",
			[]string{"unknown command \"entities\" for \"download\""},
		},
		{
			"entities manifest download with project but is env. var is disabled",
			"entities manifest test.yaml test_env --project testproject",
			[]string{"unknown command \"entities\" for \"download\""},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			// for all test cases there should be at least an error
			assert.Error(t, err)

			// for most cases we can test the message in more detail
			for _, expected := range test.errorContainsExpected {
				assert.ErrorContains(t, err, expected)
			}

			// for testing not to forget adding expectations
			assert.NotEmpty(t, test.errorContainsExpected, "no error conditions specified")
		})
	}
}

// createDownloadCommandMock creates the mock for the download command
// finish is called automatically by the testing system to ensure expected calls
func createDownloadCommandMock(t *testing.T) *MockCommand {
	mockCtrl := gomock.NewController(t)
	commandMock := NewMockCommand(mockCtrl)
	return commandMock
}
