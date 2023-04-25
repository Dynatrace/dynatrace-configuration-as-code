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
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download"
	dlautomation "github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
)

type downloaders []interface{}

func makeDownloaders(options downloadConfigsOptions) (downloaders, error) {
	dtClient, err := dynatrace.CreateClient(options.environmentURL, options.auth, false,
		dtclient.WithClientRequestLimiter(concurrency.NewLimiter(environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey))))
	if err != nil {
		return nil, err
	}

	autClient := automation.NewClient(options.environmentURL, client.NewOAuthClient(context.TODO(), client.OauthCredentials{
		ClientID:     options.auth.OAuth.ClientID.Value,
		ClientSecret: options.auth.OAuth.ClientSecret.Value,
		TokenURL:     options.auth.OAuth.GetTokenEndpointValue(),
	}))

	return downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient), dlautomation.NewDownloader(autClient)}, nil
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
