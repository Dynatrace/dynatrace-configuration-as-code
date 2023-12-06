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

package account

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	presistance "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/writer"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"os"
)

func downloadAll(fs afero.Fs, opts downloadOpts) error {
	if opts.outputFolder == "" {
		opts.outputFolder = "project/accounts" //TODO: make output folder unique (Where to prevent overwriting - write/load module or here?)
	}

	var accs map[string]manifest.Account
	if opts.accountUUID == "" {
		m, errs := manifestloader.Load(&manifestloader.Context{
			Fs:           fs,
			ManifestPath: opts.manifestName,
		})
		if len(errs) > 0 {
			errutils.PrintErrors(errs)
			return errors.New("error while loading manifest")
		}

		if len(opts.accountName) > 0 {
			for _, a := range opts.accountName {
				if n, ok := m.Accounts[a]; !ok {
					return fmt.Errorf("unknown enviroment %q", n.Name)
				}
			}
			for _, a := range opts.accountName {
				accs = make(map[string]manifest.Account)
				accs[a] = m.Accounts[a]
			}
		} else {
			accs = m.Accounts
		}
	} else {
		uuid, err := uuid.Parse(opts.accountUUID)
		if err != nil {
			return fmt.Errorf("failed to parese accountUUID: %w", err)
		}
		clientID, err := readEnvVariable(opts.clientID)
		if err != nil {
			return err
		}
		clientSecret, err := readEnvVariable(opts.clientSecret)
		if err != nil {
			return err
		}
		accs = make(map[string]manifest.Account, 1)
		accs["account"] = manifest.Account{
			Name:        "account",
			AccountUUID: uuid,
			OAuth: manifest.OAuth{
				ClientID:     clientID,
				ClientSecret: clientSecret,
			},
		}
	}

	accountClients, err := dynatrace.CreateAccountClients(accs)
	if err != nil {
		return fmt.Errorf("failed to create account clients: %w", err)
	}

	var failedDownloads []account.AccountInfo
	for acc, accClient := range accountClients {
		err := download(fs, opts, acc, accClient)
		if err != nil {
			log.Error("Configuration download for account %q failed! Cause: %s", acc, err)
			failedDownloads = append(failedDownloads, acc)
		}
	}

	if len(failedDownloads) > 0 {
		var es []string
		for _, t := range failedDownloads {
			log.Debug("Failed to download account %q (UUID: %q)", t.Name, t.AccountUUID)
			es = append(es, t.String())
		}
		return fmt.Errorf("failed to download enviromets %q", es)
	}

	return nil
}

func download(fs afero.Fs, opts downloadOpts, accInfo account.AccountInfo, accClient *accounts.Client) error {
	downloader := downloader.New(&accInfo, accClient)

	resources, err := downloader.DownloadConfiguration()
	if err != nil {
		return err
	}

	c := presistance.Context{
		Fs:            fs,
		OutputFolder:  opts.outputFolder,
		ProjectFolder: accInfo.String(),
	}
	err = presistance.Write(c, *resources)
	if err != nil {
		return err
	}

	fmt.Println(resources)
	return nil
}

func readEnvVariable(envVar string) (manifest.AuthSecret, error) {
	var content string
	if envVar == "" {
		return manifest.AuthSecret{}, fmt.Errorf("unknown environment variable name")
	} else if content = os.Getenv(envVar); content == "" {
		return manifest.AuthSecret{}, fmt.Errorf("the content of the environment variable %q is not set", envVar)
	}
	return manifest.AuthSecret{Name: envVar, Value: secret.MaskedString(content)}, nil
}
