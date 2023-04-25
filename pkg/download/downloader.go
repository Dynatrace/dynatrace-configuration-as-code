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
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	projectv2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

// Downloader represents a component that is responsible for downloading configuration for a given project from Dynatrace
type Downloader[T v2.Type] interface {

	// Download downloads configurations from a Dynatrace environment.
	// If only projectName is given, it will download all configuration.
	// If additionally specific configuration names/types are given, then it will only download those
	Download(projectName string, specificConfigs ...T) (projectv2.ConfigsPerType, error)
}
