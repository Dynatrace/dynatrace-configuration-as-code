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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/automation"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDownloadersAutomation(t *testing.T) {
	downloaders := downloaders{
		&automation.Downloader{},
	}
	automationDownloader := downloaders.Automation()
	assert.IsType(t, &automation.Downloader{}, automationDownloader)
}
func TestGetDownloader(t *testing.T) {
	downloaders := downloaders{
		&automation.Downloader{},
	}
	automationDownloader := getDownloader[config.AutomationType](downloaders)
	assert.IsType(t, &automation.Downloader{}, automationDownloader)
}

func TestGetDownloaderPanic(t *testing.T) {
	downloaders := downloaders{}
	assert.Panics(t, func() {
		getDownloader[config.ClassicApiType](downloaders)
	})
}

func Test_prepareAPIs(t *testing.T) {
	t.Run(`handling "--only*" flags`, func(t *testing.T) {
		tests := []struct {
			name  string
			given downloadConfigsOptions
		}{
			{
				name:  "onlySettings",
				given: downloadConfigsOptions{onlySettings: true},
			},
			{
				name:  "onlyAutomation",
				given: downloadConfigsOptions{onlyAutomation: true},
			},
			{
				name:  "specificSchemas is pressent",
				given: downloadConfigsOptions{specificSchemas: []string{"anything"}},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				actual := prepareAPIs(api.NewAPIs(), tc.given)
				assert.Nil(t, actual)
			})
		}
	})

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
				apis := prepareAPIs(api.NewAPIs(), tc.given)
				for _, e := range apis {
					assert.False(t, e.SkipDownload)
				}
			})
		}
	})

	t.Run("do not skip anything when `MONACO_FEAT_DOWNLOAD_FILTER` are disabled", func(t *testing.T) {
		t.Setenv("MONACO_FEAT_DOWNLOAD_FILTER", "false")
		tests := []struct {
			name  string
			given downloadConfigsOptions
		}{
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
				apis := prepareAPIs(api.NewAPIs(), tc.given)

				a := false
				for _, e := range apis {
					if e.SkipDownload {
						a = true
					}
				}
				assert.True(t, a)
			})
		}
	})

	t.Run("do not skip anything when `MONACO_FEAT_DOWNLOAD_FILTER_CLASSIC_CONFIGS` are disabled", func(t *testing.T) {
		t.Setenv("MONACO_FEAT_DOWNLOAD_FILTER_CLASSIC_CONFIGS", "false")
		tests := []struct {
			name  string
			given downloadConfigsOptions
		}{
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
				apis := prepareAPIs(api.NewAPIs(), tc.given)

				a := false
				for _, e := range apis {
					if e.SkipDownload {
						a = true
					}
				}
				assert.True(t, a)
			})
		}
	})

	t.Run("handling of deprecated endpoints", func(t *testing.T) {
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
				name:       "specificAPI marked with 'deprecatedBy' is not filtered out",
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
				apis := prepareAPIs(api.NewAPIs(), tc.given)
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
