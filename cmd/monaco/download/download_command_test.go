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
	t.Run("Token is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url")
		assert.EqualError(t, err, "--token flag missing")
	})

	t.Run("Authorization via token", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "http://some.url",
			auth:           auth{token: "TOKEN"},
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

		err := m.download("--url http://some.url --token TOKEN")
		assert.NoError(t, err)
	})

	t.Run("Authorization via OAuth", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "http://some.url",
			auth: auth{
				token:        "TOKEN",
				clientID:     "CLIENT_ID",
				clientSecret: "CLIENT_SECRET",
			},
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

		err := m.download("--url http://some.url --token TOKEN --oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET")
		assert.NoError(t, err)
	})

	t.Run("Clint ID for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --token TOKEN --oauth-client-secret CLIENT_SECRET")
		assert.ErrorContains(t, err, "--oauth-client-id flag missing")
	})
	t.Run("Clint secret for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --token TOKEN --oauth-client-id CLIENT_ID")
		assert.ErrorContains(t, err, "--oauth-client-secret flag missing")
	})

	t.Run("no specific apis provided", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "http://some.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url http://some.url --token token --project test")
		assert.NoError(t, err)
	})
	t.Run("default project provided", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "http://some.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url http://some.url --token token")
		assert.NoError(t, err)
	})
	t.Run("skip download of settings", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --only-apis")
		assert.NoError(t, err)
	})
	t.Run("skip download of APIs", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --only-settings")
		assert.NoError(t, err)
	})
	t.Run("with specific apis (multiple flags)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --project test --api test --api test2")
		assert.NoError(t, err)
	})
	t.Run("with specific apis (single flag)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --project test --api test,test2")
		assert.NoError(t, err)
	})
	t.Run("specific apis (mixed flags)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --project test --api test,test2 --api test3")
		assert.NoError(t, err)
	})
	t.Run("specific settings (single flag)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --project test --settings-schema builtin:alerting.profile,builtin:problem.notifications")
		assert.NoError(t, err)
	})
	t.Run("specific settings (mixed flags)", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --project test --settings-schema builtin:alerting.profile,builtin:problem.notifications --settings-schema builtin:metric.metadata")
		assert.NoError(t, err)
	})
	t.Run("with outputfolder", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --output-folder myDownloads")
		assert.NoError(t, err)
	})
	t.Run("output-folder and force overwrite", func(t *testing.T) {
		expected := directDownloadCmdOptions{
			environmentUrl: "test.url",
			auth:           auth{token: "token"},
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

		err := m.download("--url test.url --token token --output-folder myDownloads --force")
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
		args                  []string
		flags                 []string
		errorContainsExpected []string
	}{
		{
			"manifest not provided but missing specific environment",
			[]string{},
			[]string{},
			[]string{"missing --environment/-e flag"},
		},
		{
			"manifest provided but missing specific environment",
			[]string{},
			[]string{"manifest:manifest.yaml"},
			[]string{"missing --environment/-e flag"},
		},
		// ENTITIES
		{
			"entities: no arguments provided to direct download",
			[]string{"entities", "direct"},
			[]string{},
			[]string{"url and token have to be provided as positional argument"},
		},
		{
			"entities: url is missing other required argument",
			[]string{"entities", "direct", "some.env.url.com"},
			[]string{},
			[]string{"url and token have to be provided as positional argument"},
		},
		{
			"entities: manifest provided but missing specific environment",
			[]string{"entities", "manifest", "manifest.yaml"},
			[]string{},
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"entities: manifest is missing but environment is provider",
			[]string{"entities", "manifest", "some_env"},
			[]string{},
			[]string{"manifest and environment name have to be provided as positional arguments"},
		},
		{
			"unknown flag",
			[]string{"--test"},
			[]string{},
			[]string{"--test"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(test.args)
			for _, fl := range test.flags {
				flag := strings.Split(fl, ":")
				_ = cmd.Flags().Set(flag[0], flag[1])
			}
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
		args  []string
		flags []string
		setup func(command *MockCommand)
	}{
		// CONFIGS
		{
			"manifest download no specific apis",
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "only-apis=true"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "only-settings=true"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "api=test", "api=test2"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "api=test,test2"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "api=test,test2", "api=test3"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "settings-schema=builtin:alerting.profile,builtin:problem.notifications"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "settings-schema=builtin:alerting.profile,builtin:problem.notifications", "settings-schema=builtin:metric.metadata"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "project=testproject"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "output-folder=myDownloads"},
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
			[]string{},
			[]string{"manifest=test.yaml", "environment=test_env", "output-folder=myDownloads", "force=true"},
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
			[]string{"entities", "direct", "test.url", "token", "--project", "test"},
			[]string{},
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
			[]string{"entities", "direct", "test.url", "token"},
			[]string{},
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
			[]string{"entities", "direct", "test.url", "token", "--output-folder", "myDownloads"},
			[]string{},
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
			[]string{"entities", "direct", "test.url", "token", "--output-folder", "myDownloads", "--force"},
			[]string{},
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
			[]string{"entities", "manifest", "test.yaml", "test_env"},
			[]string{},
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
			[]string{"entities", "manifest", "test.yaml", "test_env", "--project", "testproject"},
			[]string{},
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
			[]string{"entities", "manifest", "test.yaml", "test_env", "--output-folder", "myDownloads"},
			[]string{},
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
			[]string{"entities", "manifest", "test.yaml", "test_env", "--output-folder", "myDownloads", "--force"},
			[]string{},
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
			[]string{"entities", "manifest", "test.yaml", "test_env", "--specific-types", "HOST,SERVICE"},
			[]string{},
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
			cmd.SetArgs(test.args)
			for _, fl := range test.flags {
				flag := strings.Split(fl, "=")
				_ = cmd.Flags().Set(flag[0], flag[1])
			}
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			assert.NoError(t, err, "no error expected")
		})
	}
}

func TestEntitiesDownloadEnabled(t *testing.T) {
	commandMock := createDownloadCommandMock(t)
	cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{"entities"})
	cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
	err := cmd.Execute()
	assert.Error(t, err)

}

// createDownloadCommandMock creates the mock for the download command
// finish is called automatically by the testing system to ensure expected calls
func createDownloadCommandMock(t *testing.T) *MockCommand {
	mockCtrl := gomock.NewController(t)
	commandMock := NewMockCommand(mockCtrl)
	return commandMock
}
