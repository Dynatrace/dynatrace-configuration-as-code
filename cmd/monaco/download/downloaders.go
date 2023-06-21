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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download"
	dlautomation "github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
)

type downloaders []interface{}

func makeDownloaders(options downloadConfigsOptions) (downloaders, error) {
	clients, err := dynatrace.CreateClientSet(options.environmentURL, options.auth)
	if err != nil {
		return nil, err
	}

	var automationDownloader download.Downloader[config.AutomationType] = dlautomation.NoopAutomationDownloader{}
	if clients.Automation() != nil {
		automationDownloader = dlautomation.NewDownloader(clients.Automation())
	}
	var settingsDownloader download.Downloader[config.SettingsType] = settings.NewDownloader(clients.Settings())
	var classicDownloader download.Downloader[config.ClassicApiType] = classicDownloader(clients.Classic(), options)
	return downloaders{settingsDownloader, classicDownloader, automationDownloader}, nil
}

func classicDownloader(client dtclient.Client, opts downloadConfigsOptions) *classic.Downloader {
	apis := prepareAPIs(opts)
	return classic.NewDownloader(client, apis)
}

func prepareAPIs(opts downloadConfigsOptions) api.APIs {
	switch {
	case opts.onlyAutomation:
		return nil
	case opts.onlySettings:
		return nil
	case opts.onlyAPIs:
		return api.NewAPIs().Filter(skipDownloadFilter, removeDeprecated(withWarn()))
	case len(opts.specificAPIs) > 0:
		return api.NewAPIs().Filter(api.RetainByName(opts.specificAPIs), skipDownloadFilter, warnDeprecated())
	case len(opts.specificSchemas) == 0:
		return api.NewAPIs().Filter(skipDownloadFilter, removeDeprecated())
	default:
		return nil
	}
}

func skipDownloadFilter(api api.API) bool {
	if api.SkipDownload {
		log.Info("API can not be downloaded and needs manual creation: '%v'.", api.ID)
		return true
	}
	return false
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
			log.Warn("classic endpoint %q is deprecated by %q and will not be downloaded", api.ID, api.DeprecatedBy)
		}
	}
}

func (d downloaders) Classic() download.Downloader[config.ClassicApiType] {
	return getDownloader[config.ClassicApiType](d)
}

func warnDeprecated() api.Filter {
	return func(api api.API) bool {
		if api.DeprecatedBy != "" {
			log.Warn("classic endpoint %q is deprecated by %q", api.ID, api.DeprecatedBy)
		}
		return false
	}
}

func (d downloaders) Settings() download.Downloader[config.SettingsType] {
	return getDownloader[config.SettingsType](d)
}

func (d downloaders) Automation() download.Downloader[config.AutomationType] {
	return getDownloader[config.AutomationType](d)
}

func getDownloader[T config.Type](d downloaders) download.Downloader[T] {
	for _, downloader := range d {
		if dl, ok := downloader.(download.Downloader[T]); ok {
			return dl
		}
	}
	panic("No downloader implementation found")
}
