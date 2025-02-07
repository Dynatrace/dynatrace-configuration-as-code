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
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetDownloadCommand(t *testing.T) {
	t.Run("url and token are mutually exclusive", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --manifest my-manifest.yaml")
		assert.EqualError(t, err, "'url' and 'manifest' are mutually exclusive")
	})

	t.Run("Download via manifest - manifest set explicitly", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "path/to/my-manifest.yaml",
			specificEnvironmentName: "my-environment1",
			projectName:             "project",
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--manifest path/to/my-manifest.yaml --environment my-environment1")

		assert.NoError(t, err)
	})

	t.Run("Download via manifest - manifest is not set (will take default value)", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "my-environment",
			projectName:             "project",
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

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
			environmentURL: "http://some.url",
			auth:           auth{token: "TOKEN"},
			projectName:    "project",
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

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
			projectName: "project",
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

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
			projectName:             "my-project",
			outputFolder:            "path/to/my-folder",
			forceOverwrite:          true,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--manifest path/my-manifest.yaml --environment my-environment --project my-project --output-folder path/to/my-folder --force true")

		assert.NoError(t, err)
	})

	t.Run("If not provided, default project name is 'project'", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "my_environment",
			projectName:             "project",
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment my_environment")
		assert.NoError(t, err)
	})

	t.Run("Api selection - set of wanted api", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "myEnvironment",
			projectName:             "project",
			specificAPIs:            []string{"test", "test2", "test3", "test4"},
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --api test --api test2 --api test3,test4")
		assert.NoError(t, err)
	})

	t.Run("Api selection - download all api", func(t *testing.T) {
		expected := downloadCmdOptions{
			environmentURL: "test.url",
			auth:           auth{token: "token"},
			projectName:    "project",
			onlyAPIs:       true,
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

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

		err = m.download("--environment myEnvironment --only-apis --only-settings")
		assert.Error(t, err)
	})

	t.Run("Settings schema selection - set of wanted settings schema", func(t *testing.T) {
		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "myEnvironment",
			projectName:             "project",
			specificSchemas:         []string{"settings:schema:1", "settings:schema:2", "settings:schema:3", "settings:schema:4"},
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --settings-schema settings:schema:1 --settings-schema settings:schema:2 --settings-schema settings:schema:3,settings:schema:4")
		assert.NoError(t, err)
	})

	t.Run("Settings schema selection - download all settings schema", func(t *testing.T) {
		expected := downloadCmdOptions{
			environmentURL: "test.url",
			auth:           auth{token: "token"},
			projectName:    "project",
			onlySettings:   true,
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

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

		err = m.download("--environment myEnvironment --only-apis --only-settings")
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
