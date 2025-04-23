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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

func Test_prepareAPIs(t *testing.T) {
	t.Run(`handling "--only*" flags return nil for APIs`, func(t *testing.T) {
		tests := []struct {
			name  string
			given downloadConfigsOptions
		}{
			{
				name:  "onlySettings",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySettingsFlag: true}},
			},
			{
				name:  "onlyDocuments",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyDocumentsFlag: true}},
			},
			{
				name:  "onlyOpenpipeline",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyOpenPipelineFlag: true}},
			},
			{
				name:  "onlyAutomation",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyAutomationFlag: true}},
			},
			{
				name:  "onlyFilterSegments",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySegmentsFlag: true}},
			},
			{
				name:  "onlyFilterBuckets",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyBucketsFlag: true}},
			},
			{
				name:  "onlyFilterSloV2",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySloV2Flag: true}},
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
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyAutomationFlag: true}},
			},
			{
				name:  "onlySettings",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySettingsFlag: true}},
			},
			{
				name:  "onlyDocuments",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyDocumentsFlag: true}},
			},
			{
				name:  "onlyBuckets",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyBucketsFlag: true}},
			},
			{
				name:  "onlyOpenpipeline",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyOpenPipelineFlag: true}},
			},
			{
				name:  "onlySegment",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySegmentsFlag: true}},
			},
			{
				name:  "onlyAPIs",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyApisFlag: true}},
			},
			{
				name:  "specificAPIs is marked as 'skip'",
				given: downloadConfigsOptions{specificAPIs: []string{"extension"}, onlyOptions: OnlyOptions{OnlyApisFlag: true}},
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

	t.Run("require to set all of listed FF", func(t *testing.T) {
		testApi := api.API{
			ID:           "test-endpoint",
			RequireAllFF: []featureflags.FeatureFlag{featureflags.ExtractScopeAsParameter},
		}
		type given struct {
			apis api.APIs
			ff   []featureflags.FeatureFlag
		}
		tests := []struct {
			name     string
			given    given
			expected api.APIs
		}{
			{
				name: "with set FF",
				given: given{
					apis: api.APIs{testApi.ID: testApi},
					ff:   []featureflags.FeatureFlag{featureflags.ExtractScopeAsParameter},
				},
				expected: api.APIs{testApi.ID: testApi},
			},
			{
				name: "without set FF",
				given: given{
					apis: api.APIs{testApi.ID: testApi},
				},
				expected: api.APIs{},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {

				for _, ff := range tc.given.ff {
					t.Setenv(ff.EnvName(), "true")
				}

				actualAPIs := prepareAPIs(tc.given.apis, downloadConfigsOptions{})

				assert.Equal(t, tc.expected, actualAPIs)
			})
		}
	})

	t.Run("do not skip anything when `MONACO_FEAT_DOWNLOAD_FILTER` are disabled", func(t *testing.T) {
		t.Setenv(featureflags.DownloadFilter.EnvName(), "false") // by default, it is true
		tests := []struct {
			name  string
			given downloadConfigsOptions
		}{
			{
				name:  "onlyAPIs",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyApisFlag: true}},
			},
			{
				name:  "specificAPIs is marked as 'skip'",
				given: downloadConfigsOptions{specificAPIs: []string{"extension"}, onlyOptions: OnlyOptions{OnlyApisFlag: true}},
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
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyApisFlag: true}},
			},
			{
				name:  "specificAPIs is marked as 'skip'",
				given: downloadConfigsOptions{specificAPIs: []string{"extension"}, onlyOptions: OnlyOptions{OnlyApisFlag: true}},
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
				name:       "onlySegment",
				given:      downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySegmentsFlag: true}},
				deprecated: false,
			},
			{
				name:       "onlyAutomation",
				given:      downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyAutomationFlag: true}},
				deprecated: false,
			},
			{
				name:       "onlySettings",
				given:      downloadConfigsOptions{onlyOptions: OnlyOptions{OnlySettingsFlag: true}},
				deprecated: false,
			},
			{
				name:  "onlyDocuments",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyDocumentsFlag: true}},
			},
			{
				name:  "onlyBuckets",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyBucketsFlag: true}},
			},
			{
				name:  "onlyOpenpipeline",
				given: downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyOpenPipelineFlag: true}},
			},
			{
				name:       "onlyAPIs",
				given:      downloadConfigsOptions{onlyOptions: OnlyOptions{OnlyOpenPipelineFlag: true}},
				deprecated: false,
			},
			{
				name:       "specificAPI marked with 'deprecatedBy' is not filtered out",
				given:      downloadConfigsOptions{specificAPIs: []string{"auto-tag"}, onlyOptions: OnlyOptions{OnlyApisFlag: true}},
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
