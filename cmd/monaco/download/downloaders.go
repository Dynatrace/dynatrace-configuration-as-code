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
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
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

	var automationDownloader download.Downloader[v2.AutomationType] = dlautomation.NoopAutomationDownloader{}
	if clients.Automation() != nil {
		automationDownloader = dlautomation.NewDownloader(clients.Automation())
	}
	var settingsDownloader download.Downloader[v2.SettingsType] = settings.NewDownloader(clients.Settings())
	var classicDownloader download.Downloader[v2.ClassicApiType] = classic.NewDownloader(clients.Classic())
	return downloaders{settingsDownloader, classicDownloader, automationDownloader}, nil
}

func (d downloaders) Classic() download.Downloader[v2.ClassicApiType] {
	return getDownloader[v2.ClassicApiType](d)
}

func (d downloaders) Settings() download.Downloader[v2.SettingsType] {
	return getDownloader[v2.SettingsType](d)
}

func (d downloaders) Automation() download.Downloader[v2.AutomationType] {
	return getDownloader[v2.AutomationType](d)
}

func getDownloader[T v2.Type](d downloaders) download.Downloader[T] {
	for _, downloader := range d {
		if dl, ok := downloader.(download.Downloader[T]); ok {
			return dl
		}
	}
	panic("No downloader implementation found")
}
