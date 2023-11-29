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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/downloader"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	presistance "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/writer"
	"github.com/spf13/afero"
)

func downloadAll(fs afero.Fs, opts downloadOpts) error {

	opts.outputFolder = ".output"

	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: opts.manifestName,
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return errors.New("error while loading manifest") //FIXME: provide error message
	}

	// filter account
	accs := m.Accounts

	accountClients, err := dynatrace.CreateAccountClients(accs)
	if err != nil {
		return fmt.Errorf("failed to create account clients: %w", err)
	}

	//supportedPermissions, err := deployer.DefaultPermissionProvider() //FIXME: ignores apiUrl from manifest
	//if err != nil {
	//	return fmt.Errorf("failed to fetch supportedPermissions: %w", err)
	//}

	for acc, accClient := range accountClients {
		download(fs, opts, acc, accClient)
	}

	return nil
}

func download(fs afero.Fs, opts downloadOpts, accInfo account.AccountInfo, accClient *accounts.Client) error {
	downloader := downloader.New(&accInfo, accClient)
	uu, err := downloader.Users()
	if err != nil {
		return err
	}

	_, err = downloader.Groups()
	if err != nil {
		return err
	}

	users := make(map[account.UserId]account.User)
	for i := range uu {
		users[uu[i].Email] = uu[i]
	}

	resources := account.Resources{
		Users: users,
	}

	c := presistance.Context{
		Fs:            fs,
		OutputFolder:  opts.outputFolder,
		ProjectFolder: accInfo.String(),
	}
	err = presistance.Write(c, resources)
	if err != nil {
		return err
	}

	fmt.Println(resources)
	return nil
}
