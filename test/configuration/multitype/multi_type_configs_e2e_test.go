//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package multitype

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/v2"
)

const multiTypeProjectFolder = "testdata/integration-multi-type-configs/"
const multiTypeManifest = multiTypeProjectFolder + "manifest.yaml"

func TestMultiTypeConfigsDeployment(t *testing.T) {

	v2.Run(t, multiTypeProjectFolder,
		v2.Options{
			v2.WithManifestPath(multiTypeManifest),
			v2.WithSuffix("MultiType"),
		},
		func(fs afero.Fs, _ v2.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", multiTypeManifest))
			assert.NoError(t, err)

			integrationtest.AssertAllConfigsAvailability(t, fs, multiTypeManifest, []string{}, "", true)
		})
}

func TestMultiTypeConfigsValidation(t *testing.T) {
	err := monaco.Run(t, monaco.NewTestFs(), fmt.Sprintf("monaco deploy %s --dry-run --verbose", multiTypeManifest))
	assert.NoError(t, err)
}
