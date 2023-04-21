/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package download

import (
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownloadersClassic(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
	}
	classicDownloader := downloaders.Classic()
	assert.IsType(t, &classic.Downloader{}, classicDownloader)
}

func TestDownloadersSettings(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
	}
	settingsDownloader := downloaders.Settings()
	assert.IsType(t, &settings.Downloader{}, settingsDownloader)
}

func TestGetDownloader(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
	}

	classicDownloader := getDownloader[v2.ClassicApiType](downloaders)
	assert.IsType(t, &classic.Downloader{}, classicDownloader)
	settingsDownloader := getDownloader[v2.SettingsType](downloaders)
	assert.IsType(t, &settings.Downloader{}, settingsDownloader)
}

func TestGetDownloaderPanic(t *testing.T) {
	downloaders := downloaders{}
	assert.Panics(t, func() {
		getDownloader[v2.ClassicApiType](downloaders)
	})
}
