/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

type OnlyFlags = string

const (
	OnlyApis         OnlyFlags = "only-apis"
	OnlySettings     OnlyFlags = "only-settings"
	OnlyAutomation   OnlyFlags = "only-automation"
	OnlyDocuments    OnlyFlags = "only-documents"
	OnlyBuckets      OnlyFlags = "only-buckets"
	OnlyOpenPipeline OnlyFlags = "only-openpipeline"
	OnlySloV2        OnlyFlags = "only-slo-v2"
	OnlySegments     OnlyFlags = "only-segments"
)

type OnlyOptions map[OnlyFlags]bool

// OnlyCount returns the amount of enabled "only" flags
func (o OnlyOptions) OnlyCount() int {
	count := 0
	for _, value := range o {
		if value {
			count++
		}
	}
	return count
}

// ShouldDownload returns true if the provided "only" flag is enabled or if no flag is set at all
func (o OnlyOptions) ShouldDownload(f OnlyFlags) bool {
	if o.OnlyCount() == 0 {
		return true
	}
	enabled, exists := o[f]
	return exists && enabled
}

// IsSingleOption returns true if the provided "only" flag is the only one being enabled
func (o OnlyOptions) IsSingleOption(f OnlyFlags) bool {
	return o.OnlyCount() == 1 && o.ShouldDownload(f)
}
