//go:build integration
// +build integration

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package main

import (
	"testing"
)

//TestMigrateEnvironment validates if the configurations can be downloaded from environment 1 and apply to environment 2
//It has 3 stages:
//Preparation: Uploads a set of configurations to environment 1 and return the virtual filesystem
//Execution:
//1. Download the configurations to the virtual filesystem
//2. Delete the configurations that were uploaded during the preparation stage to environment 1
//Validation: Uploads the downloaded configs to environment 2 and checks for status code 0 as result
//Cleanup: Deletes the configurations that were uploaded during validation

func TestMigrateEnvironment(t *testing.T) {

}
