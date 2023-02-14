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

/*
Package download provides all functionality required to download configuration from a Dynatrace tenant as Monaco configuration files.

Basically, the download happens in 3 steps.

 1. Download all specified APIs (or all APIs if non are specified) from Dynatrace into our in-memory representation.
 2. Resolve dependencies between the components
 3. Write the in-memory representation to disk

# Downloading

Entry point: [pkg/github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/downloader.DownloadAllConfigs]

Downloading happens in the downloader-subpackage.

# Dependency resolution

Entry point: [ResolveDependencies]

Our current approach for dependency resolution is very basic.
We collect all ids off all the configs we downloaded, and search all templates for any occurances of those ids.
In case of an occurrence, the occurrence is replaced by a generic variable, and added as a reference.

# Persistence

Entry point: [WriteToDisk]

When persisting, all configs that were downloaded are stored in a project folder inside either a specified outputFolder,
or a default folder named 'download_{TIMESTAMP}'.

If any existing configs are located there, they are overwritten.

In addition to downloaded configurations, a manifest file is created which can be used to deploy the downloaded configs.
If a manifest.yaml already exists in the outputFolder a timestamp is appended and a new manifest created to ensure a
config 'update' does not destroy existing manifests.

The result of WriteToDisk will be a full configuration project and manifest with which that project can be deployed,
written to the Filesystem.
*/
package download
