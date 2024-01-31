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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download"
	dlautomation "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/bucket"
)

type downloaders []interface{}

func (d downloaders) Classic() download.Downloader[config.ClassicApiType] {
	return getDownloader[config.ClassicApiType](d)
}

func (d downloaders) Automation() download.Downloader[config.AutomationType] {
	return getDownloader[config.AutomationType](d)
}

func (d downloaders) Bucket() download.Downloader[config.BucketType] {
	return getDownloader[config.BucketType](d)
}

func makeDownloaders(options downloadConfigsOptions) (downloaders, error) {
	clients, err := dynatrace.CreateClientSet(options.environmentURL, options.auth)
	if err != nil {
		return nil, err
	}

	var automationDownloader download.Downloader[config.AutomationType] = dlautomation.NoopAutomationDownloader{}
	if clients.Automation() != nil {
		automationDownloader = dlautomation.NewDownloader(clients.Automation())
	}
	var bucketDownloader download.Downloader[config.BucketType] = bucket.NewDownloader(clients.Bucket())

	return downloaders{automationDownloader, bucketDownloader}, nil
}

func prepareAPIs(apis api.APIs, opts downloadConfigsOptions) api.APIs {
	switch {
	case opts.onlyAutomation:
		return nil
	case opts.onlySettings:
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
	return featureflags.DownloadFilter().Enabled() && featureflags.DownloadFilterClassicConfigs().Enabled()
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

func getDownloader[T config.Type](d downloaders) download.Downloader[T] {
	for _, downloader := range d {
		if dl, ok := downloader.(download.Downloader[T]); ok {
			return dl
		}
	}
	panic("No downloader implementation found")
}
