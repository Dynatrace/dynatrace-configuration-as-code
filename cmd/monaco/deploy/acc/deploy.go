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

package acc

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/spf13/afero"
)

// deployAcc triggers the deployment of account management resources for a given monaco project
func deployAcc(fs afero.Fs, projectName string) error { //nolint:unused
	accResources, err := account.Load(fs, projectName)
	if err != nil {
		return err
	}

	err = account.Validate(accResources)
	if err != nil {
		return err
	}

	//convert acc resources to internal representation to be deployable and pass to pkg/deploy/acc::Deploy()

	return nil
}
