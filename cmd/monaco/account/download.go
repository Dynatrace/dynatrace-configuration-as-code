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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/accounts"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader"
	presistance "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/writer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	manifestwriter "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/writer"
)

func downloadAll(ctx context.Context, fs afero.Fs, opts *downloadOpts) error {
	if opts.outputFolder == "" {
		opts.outputFolder = fmt.Sprintf("download_account_%s", timeutils.TimeAnchor().Format(log.LogFileTimestampPrefixFormat))
	}

	var accs map[string]manifest.Account
	var err error
	if opts.accountUUID == "" {
		accs, err = loadAccountsFromManifest(fs, opts)
		if err != nil {
			return err
		}
	} else {
		accs, err = createAccount(opts)
		if err != nil {
			return err
		}
	}

	accountClients, err := dynatrace.CreateAccountClients(ctx, accs)
	if err != nil {
		return fmt.Errorf("failed to create account clients: %w", err)
	}

	var failedDownloads []account.AccountInfo
	for acc, accClient := range accountClients {
		err := downloadAndPersist(ctx, fs, opts, acc, accClient)
		if err != nil {
			log.ErrorContext(ctx, "Failed to download account resources for account %q: %s", acc, err)
			failedDownloads = append(failedDownloads, acc)
		}
	}

	// all environments failed to download
	if len(failedDownloads) == len(accountClients) {
		return fmt.Errorf("failed to download any resources from accounts %q - not creating download folder", maps.Keys(accs))
	}

	if err := writeManifest(fs, opts, accs); err != nil {
		log.ErrorContext(ctx, "failed to persist manifest: %s", err)
	}

	if len(failedDownloads) > 0 {
		var es []string
		for _, t := range failedDownloads {
			es = append(es, t.String())
		}
		return fmt.Errorf("failed to download resources from %q", es)
	}

	return nil
}

func createAccount(opts *downloadOpts) (map[string]manifest.Account, error) {
	uuid, err := uuid.Parse(opts.accountUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse accountUUID: %w", err)
	}
	clientID, err := readAuthSecretFromEnv(opts.clientID)
	if err != nil {
		return nil, err
	}
	clientSecret, err := readAuthSecretFromEnv(opts.clientSecret)
	if err != nil {
		return nil, err
	}
	retVal := make(map[string]manifest.Account, 1)
	retVal["account"] = manifest.Account{
		Name:        fmt.Sprintf("account_%s", uuid),
		AccountUUID: uuid,
		OAuth: manifest.OAuth{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	}
	return retVal, nil
}

func loadAccountsFromManifest(fs afero.Fs, opts *downloadOpts) (map[string]manifest.Account, error) {
	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: opts.manifestName,
		Opts:         manifestloader.Options{RequireAccounts: true},
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return nil, errors.New("error while loading manifest")
	}

	if len(opts.accountList) > 0 {
		var retVal map[string]manifest.Account
		for _, a := range opts.accountList {
			if n, ok := m.Accounts[a]; !ok {
				return nil, fmt.Errorf("unknown enviroment %q", n.Name)
			}

			retVal = make(map[string]manifest.Account)
			retVal[a] = m.Accounts[a]
		}
		return retVal, nil
	}

	return m.Accounts, nil
}

func downloadAndPersist(ctx context.Context, fs afero.Fs, opts *downloadOpts, accInfo account.AccountInfo, accClient *accounts.Client) error {
	downloader := downloader.New(&accInfo, accClient)

	ctx = context.WithValue(ctx, log.CtxKeyAccount{}, accInfo.Name)
	resources, err := downloader.DownloadResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to download resources: %w", err)
	}

	c := presistance.Context{
		Fs:            fs,
		OutputFolder:  opts.outputFolder,
		ProjectFolder: filepath.Join(opts.projectName, accInfo.Name),
	}
	err = presistance.Write(c, *resources)
	if err != nil {
		return fmt.Errorf("failed to persist resources: %w", err)
	}

	return nil
}

func readAuthSecretFromEnv(envVar string) (manifest.AuthSecret, error) {
	var content string
	if envVar == "" {
		return manifest.AuthSecret{}, fmt.Errorf("unknown environment variable name")
	} else if content = os.Getenv(envVar); content == "" {
		return manifest.AuthSecret{}, fmt.Errorf("the content of the environment variable %q is not set", envVar)
	}
	return manifest.AuthSecret{Name: envVar, Value: secret.MaskedString(content)}, nil
}

func writeManifest(fs afero.Fs, opts *downloadOpts, accs map[string]manifest.Account) error {
	manifestPath := filepath.Join(opts.outputFolder, "manifest.yaml")
	man := manifest.Manifest{
		Projects: manifest.ProjectDefinitionByProjectID{
			opts.projectName: manifest.ProjectDefinition{Name: opts.projectName},
		},
		Accounts: accs,
	}

	return manifestwriter.Write(fs, manifestPath, man)
}
