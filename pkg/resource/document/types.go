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

package document

import (
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/documents"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

// supportedDocumentTypes are all document types supported by Monaco.
// due to the current test setup, the types must be downloaded in order. This should be changed eventually.
var supportedDocumentTypes = []documents.DocumentType{
	documents.Dashboard,
	documents.Notebook,
	documents.Launchpad,
}

// documentTypeToKind maps document types to config document kinds
var documentTypeToKind = map[string]config.DocumentKind{
	documents.Dashboard: config.DashboardKind,
	documents.Notebook:  config.NotebookKind,
	documents.Launchpad: config.LaunchpadKind,
}

// documentKindToType maps config document kinds to document types
var documentKindToType = map[config.DocumentKind]documents.DocumentType{
	config.DashboardKind: documents.Dashboard,
	config.NotebookKind:  documents.Notebook,
	config.LaunchpadKind: documents.Launchpad,
}
