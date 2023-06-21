//go:build unit

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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownloadersClassic(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
		&automation.Downloader{},
	}
	classicDownloader := downloaders.Classic()
	assert.IsType(t, &classic.Downloader{}, classicDownloader)
}

func TestDownloadersSettings(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
		&automation.Downloader{},
	}
	settingsDownloader := downloaders.Settings()
	assert.IsType(t, &settings.Downloader{}, settingsDownloader)
}

func TestDownloadersAutomation(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
		&automation.Downloader{},
	}
	automationDownloader := downloaders.Automation()
	assert.IsType(t, &automation.Downloader{}, automationDownloader)
}
func TestGetDownloader(t *testing.T) {
	downloaders := downloaders{
		&classic.Downloader{},
		&settings.Downloader{},
		&automation.Downloader{},
	}

	classicDownloader := getDownloader[v2.ClassicApiType](downloaders)
	assert.IsType(t, &classic.Downloader{}, classicDownloader)
	settingsDownloader := getDownloader[v2.SettingsType](downloaders)
	assert.IsType(t, &settings.Downloader{}, settingsDownloader)
	automationDownloader := getDownloader[v2.AutomationType](downloaders)
	assert.IsType(t, &automation.Downloader{}, automationDownloader)
}

func TestGetDownloaderPanic(t *testing.T) {
	downloaders := downloaders{}
	assert.Panics(t, func() {
		getDownloader[v2.ClassicApiType](downloaders)
	})
}

func Test_prepareAPIs(t *testing.T) {
	t.Run("no endpoint marked as 'skip' is present", func(t *testing.T) {
		tests := []struct {
			name  string
			given downloadConfigsOptions
		}{
			{
				name:  "onlyAutomation",
				given: downloadConfigsOptions{onlyAutomation: true},
			},
			{
				name:  "onlySettings",
				given: downloadConfigsOptions{onlySettings: true},
			},
			{
				name:  "onlyAPIs",
				given: downloadConfigsOptions{onlyAPIs: true},
			},
			{
				name:  "specificAPIs is marked as 'skip'",
				given: downloadConfigsOptions{specificAPIs: []string{"extension"}},
			},
			{
				name: "without special cases",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				apis := prepareAPIs(tc.given)
				for _, e := range apis {
					assert.False(t, e.SkipDownload)
				}
			})
		}
	})

	t.Run("filtration of deprecated", func(t *testing.T) {
		tests := []struct {
			name       string
			given      downloadConfigsOptions
			deprecated bool
		}{
			{
				name:       "onlyAutomation",
				given:      downloadConfigsOptions{onlyAutomation: true},
				deprecated: false,
			},
			{
				name:       "onlySettings",
				given:      downloadConfigsOptions{onlySettings: true},
				deprecated: false,
			},
			{
				name:       "onlyAPIs",
				given:      downloadConfigsOptions{onlyAPIs: true},
				deprecated: false,
			},
			{
				name:       "specificAPIs is marked with 'deprecatedBy'",
				given:      downloadConfigsOptions{specificAPIs: []string{"auto-tag"}},
				deprecated: true,
			},
			{
				name:       "without special cases",
				deprecated: false,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				apis := prepareAPIs(tc.given)
				exists := false

				for _, e := range apis {
					if !tc.deprecated {
						assert.Equal(t, "", e.DeprecatedBy)
					}

					if e.DeprecatedBy != "" {
						exists = true
					}
				}
				if tc.deprecated {
					assert.True(t, exists)
				}
			})
		}
	})
}
