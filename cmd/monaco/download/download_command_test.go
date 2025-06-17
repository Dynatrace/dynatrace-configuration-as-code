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
	"maps"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
)

func TestGetDownloadCommand(t *testing.T) {
	defaultOnlyOptions := OnlyOptions{
		OnlySettingsFlag:     false,
		OnlyApisFlag:         false,
		OnlySegmentsFlag:     false,
		OnlySloV2Flag:        false,
		OnlyOpenPipelineFlag: false,
		OnlyDocumentsFlag:    false,
		OnlyBucketsFlag:      false,
	}

	t.Run("URL and manifest are mutually exclusive", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --manifest manifest.yaml")
		assert.EqualError(t, err, "'url' and 'manifest' are mutually exclusive")
	})

	t.Run("Download using manifest - manifest file set explicitly", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "path/to/my-manifest.yaml",
			specificEnvironmentName: "my-environment1",
			projectName:             "project",
			onlyOptions:             defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--manifest path/to/my-manifest.yaml --environment my-environment1")

		assert.NoError(t, err)
	})

	t.Run("Download using manifest - default manifest file used if not set", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "my-environment",
			projectName:             "project",
			onlyOptions:             defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment my-environment")

		assert.NoError(t, err)
	})

	t.Run("Download using manifest - environment missing", func(t *testing.T) {
		err := newMonaco(t).download("")
		assert.EqualError(t, err, "to download with manifest, 'environment' needs to be specified")
	})

	t.Run("Download using manifest - API token cannot be specified", func(t *testing.T) {
		err := newMonaco(t).download("--token API_TOKEN")
		assert.EqualError(t, err, "'token', 'oauth-client-id', and 'oauth-client-secret' can only be used with 'url', while 'manifest' must NOT be set ")
	})

	t.Run("Download using manifest - OAuth client ID cannot be specified", func(t *testing.T) {
		err := newMonaco(t).download("--oauth-client-id CLIENT_ID")
		assert.EqualError(t, err, "'token', 'oauth-client-id', and 'oauth-client-secret' can only be used with 'url', while 'manifest' must NOT be set ")
	})

	t.Run("Download using manifest - OAuth client secret cannot be specified", func(t *testing.T) {
		err := newMonaco(t).download("--oauth-client-secret CLIENT_SECRET")
		assert.EqualError(t, err, "'token', 'oauth-client-id', and 'oauth-client-secret' can only be used with 'url', while 'manifest' must NOT be set ")
	})

	t.Run("Download using manifest - platform token cannot be specified", func(t *testing.T) {
		t.Setenv(featureflags.PlatformToken.EnvName(), "true")
		err := newMonaco(t).download("--platform-token PLATFORM_TOKEN")
		assert.EqualError(t, err, "'token', 'oauth-client-id', 'oauth-client-secret', and 'platform-token' can only be used with 'url', while 'manifest' must NOT be set ")
	})

	t.Run("Direct download - just token", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL: "http://some.url",
			auth:           auth{apiToken: "TOKEN"},
			projectName:    "project",
			onlyOptions:    defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --token TOKEN")

		assert.NoError(t, err)
	})

	t.Run("Direct download - just OAuth", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL: "http://some.url",
			auth: auth{
				clientID:     "CLIENT_ID",
				clientSecret: "CLIENT_SECRET",
			},
			projectName: "project",
			onlyOptions: defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET")
		assert.NoError(t, err)
	})

	t.Run("Direct download - token and OAuth", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL: "http://some.url",
			auth: auth{
				apiToken:     "TOKEN",
				clientID:     "CLIENT_ID",
				clientSecret: "CLIENT_SECRET",
			},
			projectName: "project",
			onlyOptions: defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --token TOKEN --oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET")
		assert.NoError(t, err)
	})

	t.Run("Direct download - just platform token", func(t *testing.T) {
		t.Setenv(featureflags.PlatformToken.EnvName(), "true")
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL: "http://some.url",
			auth: auth{
				platformToken: "PLATFORM_TOKEN",
			},
			projectName: "project",
			onlyOptions: defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --platform-token PLATFORM_TOKEN")
		assert.NoError(t, err)
	})

	t.Run("Direct download - API token and platform token", func(t *testing.T) {
		t.Setenv(featureflags.PlatformToken.EnvName(), "true")
		m := newMonaco(t)

		expected := downloadCmdOptions{
			environmentURL: "http://some.url",
			auth: auth{
				apiToken:      "API_TOKEN",
				platformToken: "PLATFORM_TOKEN",
			},
			projectName: "project",
			onlyOptions: defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url http://some.url --token API_TOKEN --platform-token PLATFORM_TOKEN")
		assert.NoError(t, err)
	})

	t.Run("Direct download - missing token or OAuth credentials", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url")
		assert.EqualError(t, err, "if 'url' is set, 'token' or 'oauth-client-id' and 'oauth-client-secret' must also be set")
	})

	t.Run("Direct download - missing token, OAuth credentials or platform token", func(t *testing.T) {
		t.Setenv(featureflags.PlatformToken.EnvName(), "true")
		err := newMonaco(t).download("--url http://some.url")
		assert.EqualError(t, err, "if 'url' is set, 'token', 'oauth-client-id' and 'oauth-client-secret', or 'platform-token' must also be set")
	})

	t.Run("Direct download - client ID for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --oauth-client-secret CLIENT_SECRET")
		assert.EqualError(t, err, "'oauth-client-id' and 'oauth-client-secret' must always be set together")
	})

	t.Run("Direct download - client secret for OAuth authorization is missing", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --oauth-client-id CLIENT_ID")
		assert.EqualError(t, err, "'oauth-client-id' and 'oauth-client-secret' must always be set together")
	})

	t.Run("Direct download - OAuth and platform token cant be used together", func(t *testing.T) {
		t.Setenv(featureflags.PlatformToken.EnvName(), "true")
		err := newMonaco(t).download("--url http://some.url --oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET --platform-token PLATFORM_TOKEN")
		assert.EqualError(t, err, "OAuth credentials and a platform token can't be used together")
	})

	t.Run("Direct download - environment specified", func(t *testing.T) {
		err := newMonaco(t).download("--url http://some.url --token API_TOKEN --environment environment")
		assert.EqualError(t, err, "'environment' is specific to manifest-based download and incompatible with direct download from 'url'")
	})

	t.Run("All non-conflicting flags", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "path/my-manifest.yaml",
			specificEnvironmentName: "my-environment",
			projectName:             "my-project",
			outputFolder:            "path/to/my-folder",
			forceOverwrite:          true,
			onlyOptions:             defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--manifest path/my-manifest.yaml --environment my-environment --project my-project --output-folder path/to/my-folder --force true")

		assert.NoError(t, err)
	})

	t.Run("Default project name is used if not set", func(t *testing.T) {
		m := newMonaco(t)

		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "my_environment",
			projectName:             "project",
			onlyOptions:             defaultOnlyOptions,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment my_environment")
		assert.NoError(t, err)
	})

	t.Run("Download multiple config types", func(t *testing.T) {
		m := newMonaco(t)
		onlyOptions := maps.Clone(defaultOnlyOptions)
		onlyOptions[OnlyApisFlag] = true
		onlyOptions[OnlySettingsFlag] = true
		onlyOptions[OnlySegmentsFlag] = true
		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "myEnvironment",
			projectName:             "project",
			onlyOptions:             onlyOptions,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --only-apis --only-settings --only-segments")
		assert.NoError(t, err)
	})

	t.Run("API selection - set of wanted APIs", func(t *testing.T) {
		m := newMonaco(t)
		onlyOptions := maps.Clone(defaultOnlyOptions)
		onlyOptions[OnlyApisFlag] = true
		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "myEnvironment",
			projectName:             "project",
			specificAPIs:            []string{"test", "test2", "test3", "test4"},
			onlyOptions:             onlyOptions,
		}
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --api test --api test2 --api test3,test4")
		assert.NoError(t, err)
	})

	t.Run("API selection - download all APIs", func(t *testing.T) {
		onlyOptions := maps.Clone(defaultOnlyOptions)
		onlyOptions[OnlyApisFlag] = true
		expected := downloadCmdOptions{
			environmentURL: "test.url",
			auth:           auth{apiToken: "token"},
			projectName:    "project",
			onlyOptions:    onlyOptions,
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url test.url --token token --only-apis")
		assert.NoError(t, err)
	})

	t.Run("API selection - mutually exclusive combination", func(t *testing.T) {
		m := newMonaco(t)

		err := m.download("--environment myEnvironment --api test,test2 --only-apis")
		assert.Error(t, err)
	})

	t.Run("Settings schema selection - set of wanted settings schema", func(t *testing.T) {
		onlyOptions := maps.Clone(defaultOnlyOptions)
		onlyOptions[OnlySettingsFlag] = true
		expected := downloadCmdOptions{
			manifestFile:            "manifest.yaml",
			specificEnvironmentName: "myEnvironment",
			projectName:             "project",
			specificSchemas:         []string{"settings:schema:1", "settings:schema:2", "settings:schema:3", "settings:schema:4"},
			onlyOptions:             onlyOptions,
		}
		m := newMonaco(t)
		m.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--environment myEnvironment --settings-schema settings:schema:1 --settings-schema settings:schema:2 --settings-schema settings:schema:3,settings:schema:4")
		assert.NoError(t, err)
	})

	t.Run("Settings schema selection - download all settings schema", func(t *testing.T) {
		onlyOptions := maps.Clone(defaultOnlyOptions)
		onlyOptions[OnlySettingsFlag] = true
		expected := downloadCmdOptions{
			environmentURL: "test.url",
			auth:           auth{apiToken: "token"},
			projectName:    "project",
			onlyOptions:    onlyOptions,
		}

		m := newMonaco(t)
		m.EXPECT().DownloadConfigs(gomock.Any(), gomock.Any(), expected).Return(nil)

		err := m.download("--url test.url --token token --only-settings")
		assert.NoError(t, err)
	})

	t.Run("Settings schema selection - mutually exclusive combination", func(t *testing.T) {
		m := newMonaco(t)

		err := m.download("--environment myEnvironment --settings-schema schema:1,schema:2 --only-settings")
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
