// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

// Template is the main interface of a configuration payload that may contain template references (using Go Templates)
// The main implementation is FileBasedTemplate, which loads its Content from a file on disk. An InMemoryTemplate exists
// as well and is used in cases where Template data is not related to a file - e.g. during download or convert.
type Template interface {
	// ID of the template
	Id() string

	// Content returns the string content of the template, returns error if content is not accessible
	Content() (string, error)

	// UpdateContent sets the content of the template to the new provided one, returns error if update failed
	UpdateContent(newContent string) error
}
