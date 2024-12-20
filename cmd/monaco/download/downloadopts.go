/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

type downloadConfigsOptions struct {
	downloadOptionsShared
	specificAPIs     []string
	specificSchemas  []string
	onlyAPIs         bool
	onlySettings     bool
	onlyAutomation   bool
	onlyDocuments    bool
	onlyOpenPipeline bool
	onlySegment      bool
}

func (opts downloadConfigsOptions) valid() []error {
	var retVal []error
	knownEndpoints := api.NewAPIs().Filter(api.RemoveDisabled)
	for _, e := range opts.specificAPIs {
		if !knownEndpoints.Contains(e) {
			retVal = append(retVal, fmt.Errorf("unknown (or unsupported) classic endpoint with name %q provided via \"--api\" flag. A list of supported classic endpoints is in the documentation", e))
		}
	}

	return retVal
}

func prepareAPIs(apis api.APIs, opts downloadConfigsOptions) api.APIs {
	apis = apis.Filter(api.RemoveDisabled)
	switch {
	case opts.onlyOpenPipeline:
		return nil
	case opts.onlyDocuments:
		return nil
	case opts.onlyAutomation:
		return nil
	case opts.onlySettings:
		return nil
	case opts.onlySegment:
		return nil
	case opts.onlyAPIs:
		return apis.Filter(removeSkipDownload, removeDeprecated(withWarn()))
	case len(opts.specificAPIs) > 0:
		return apis.Filter(api.RetainByName(opts.specificAPIs), removeSkipDownload, warnDeprecated())
	case len(opts.specificSchemas) == 0:
		return apis.Filter(removeSkipDownload, removeDeprecated())
	default:
		return nil
	}
}

func removeSkipDownload(api api.API) bool {
	if shouldApplyFilter() {
		if api.SkipDownload {
			log.Info("API can not be downloaded and needs manual creation: '%v'.", api.ID)
			return true
		}
	}
	return false
}

func shouldApplyFilter() bool {
	return featureflags.DownloadFilter.Enabled() && featureflags.DownloadFilterClassicConfigs.Enabled()
}

func removeDeprecated(log ...func(api api.API)) api.Filter {
	return func(api api.API) bool {
		if api.DeprecatedBy != "" {
			if len(log) > 0 {
				log[0](api)
			}
			return true
		}
		return false
	}
}

func withWarn() func(api api.API) {
	return func(api api.API) {
		if api.DeprecatedBy != "" {
			log.Warn("classic config endpoint %q is deprecated by %q and will not be downloaded", api.ID, api.DeprecatedBy)
		}
	}
}

func warnDeprecated() api.Filter {
	return func(api api.API) bool {
		if api.DeprecatedBy != "" {
			log.Warn("classic config endpoint %q is deprecated by %q", api.ID, api.DeprecatedBy)
		}
		return false
	}
}
