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

func TestGetDownloadCommand(t *testing.T) {
	t.Run("url and token are mutually exclusive", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --manifest my-manifest.yaml")
		assert.EqualError(t, err, "'url' and 'manifest' are mutually exclusive")
	})

	t.Run("Download via manifest - manifest set explicitly", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:             "path/to/my-manifest.yaml",
			specificEnvironmentName:  "my-environment1",
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), expected).Return(nil)

		err := m.download("--manifest path/to/my-manifest.yaml --environment my-environment1")

		assert.NoError(t, err)
	})

	t.Run("Download via manifest - manifest is not set (will take default value)", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:             "manifest.yaml",
			specificEnvironmentName:  "my-environment",
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), expected).Return(nil)

		err := m.download("--environment my-environment")

		assert.NoError(t, err)
	})

	t.Run("Download via manifest.yaml - environment missing", func(t *testing.T) {
		err := newMonaco(t).download("")
		assert.EqualError(t, err, "to download with manifest, 'environment' needs to be specified")
	})

	t.Run("Download w/o manifest.yaml - authorization via token", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL:           "http://some.url",
			auth:                     auth{token: "TOKEN"},
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --token TOKEN")

		assert.NoError(t, err)
	})

	t.Run("Download w/o manifest.yaml - authorization via OAuth", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL: "http://some.url",
			auth: auth{
				token:        "TOKEN",
				clientID:     "CLIENT_ID",
				clientSecret: "CLIENT_SECRET",
			},
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --token TOKEN --oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET")
		assert.NoError(t, err)
	})

	t.Run("Download w/o manifest.yaml - token missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url")
		assert.EqualError(t, err, "if 'url' is set, 'token' also must be set")
	})

	t.Run("Download w/o manifest.yaml - clint ID for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --token TOKEN --oauth-client-secret CLIENT_SECRET")
		assert.EqualError(t, err, "'oauth-client-id' and 'oauth-client-secret' must always be set together")
	})

	t.Run("Download w/o manifest.yaml - clint secret for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --token TOKEN --oauth-client-id CLIENT_ID")
		assert.EqualError(t, err, "'oauth-client-id' and 'oauth-client-secret' must always be set together")
	})

	t.Run("All non conflicting flags", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "path/my-manifest.yaml",
			specificEnvironmentName: "my-environment",
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{
				projectName:    "my-project",
				outputFolder:   "path/to/my-folder",
				forceOverwrite: true,
			},
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), expected).Return(nil)

		err := m.download("--manifest path/my-manifest.yaml --environment my-environment --project my-project --output-folder path/to/my-folder --force true")

		assert.NoError(t, err)
	})

	t.Run("If not provided, default project name is 'project'", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:             "manifest.yaml",
			specificEnvironmentName:  "my_environment",
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), expected).Return(nil)

		err := m.download("--environment my_environment")
		assert.NoError(t, err)
	})

	t.Run("Api selection - set of wanted api", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:             "manifest.yaml",
			specificEnvironmentName:  "myEnvironment",
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
			specificAPIs:             []string{"test", "test2", "test3", "test4"},
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --api test --api test2 --api test3,test4")
		assert.NoError(t, err)
	})

	t.Run("Api selection - download all api", func(t *testing.T) {
		expected := downloadCmdOptions{
			environmentURL:           "test.url",
			auth:                     auth{token: "token"},
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
			onlyAPIs:                 true,
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("--url test.url --token token --only-apis")
		assert.NoError(t, err)
	})

	t.Run("Api selection - mutually exclusive combination", func(t *testing.T) {
		m := newMonaco(t)
		var err error

		err = m.download("--environment myEnvironment --api test,test2 --only-apis")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --api test,test2 --only-settings")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --api test,test2 --only-automation")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --only-apis --only-settings")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --only-apis --only-automation")
		assert.Error(t, err)
	})

	t.Run("Settings schema selection - set of wanted settings schema", func(t *testing.T) {
		expected := downloadCmdOptions{
			manifestFile:             "manifest.yaml",
			specificEnvironmentName:  "myEnvironment",
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
			specificSchemas:          []string{"settings:schema:1", "settings:schema:2", "settings:schema:3", "settings:schema:4"},
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --settings-schema settings:schema:1 --settings-schema settings:schema:2 --settings-schema settings:schema:3,settings:schema:4")
		assert.NoError(t, err)
	})

	t.Run("Settings schema selection - download all settings schema", func(t *testing.T) {
		expected := downloadCmdOptions{
			environmentURL:           "test.url",
			auth:                     auth{token: "token"},
			sharedDownloadCmdOptions: sharedDownloadCmdOptions{projectName: "project"},
			onlySettings:             true,
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), expected).Return(nil)

		err := m.download("--url test.url --token token --only-settings")
		assert.NoError(t, err)
	})

	t.Run("Settings schema selection - mutually exclusive combination", func(t *testing.T) {
		m := newMonaco(t)
		var err error

		err = m.download("--environment myEnvironment --settings-schema schema:1,schema:2 --only-apis")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --settings-schema schema:1,schema:2 --only-settings")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --settings-schema schema:1,schema:2 --only-automation")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --only-apis --only-settings --only-automation")
		assert.Error(t, err)

		err = m.download("--environment myEnvironment --only-settings --only-automation")
		assert.Error(t, err)

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

/////////////////

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
		// ENTITIES
		{
			"entities direct download",
			[]string{"entities", "direct", "test.url", "token", "--project", "test"},
			[]string{},
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadEntities(gomock.Any(), entitiesDirectDownloadOptions{
					environmentURL: "test.url",
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
					environmentURL: "test.url",
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
					environmentURL: "test.url",
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
					environmentURL: "test.url",
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
