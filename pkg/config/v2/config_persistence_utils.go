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

package v2

import "fmt"
import "errors"

func (t configType) IsSound(knownApis map[string]struct{}) (bool, error) {
	isClassicSound, classicErrs := t.isClassicSound(knownApis)
	isSettingsSound, settingsErrs := t.Settings.isSettings2Sound()

	switch {
	case t.isSettingsPresent() && t.isClassicPresent():
		return false, errors.New("wrong configuration of type property")
	case isClassicSound != isSettingsSound:
		return true, nil
	case !t.isSettingsPresent() && !t.isClassicPresent():
		return false, errors.New("type configuration is missing")
	case t.isSettingsPresent():
		return false, settingsErrs
	case t.isClassicPresent():
		return false, classicErrs
	default:
		return false, errors.New("wrong configuration of type property")
	}
}

func (t configType) isSettingsPresent() bool {
	return t.Settings != settingsType{}
}
func (t settingsType) isSettings2Sound() (bool, error) {
	var s []string
	if t.Schema == "" {
		s = append(s, "type.schema")
	}
	if t.Scope == "" {
		s = append(s, "type.scope")
	}
	if s == nil {
		return true, nil
	}
	return false, fmt.Errorf("next property missing: %v", s)
}

func (t configType) isClassicPresent() bool {
	return t.Api != ""
}
func (t configType) isClassicSound(knownApis map[string]struct{}) (bool, error) {
	if !t.isClassicPresent() {
		return false, errors.New("missing 'type.api' property")
	} else if _, found := knownApis[t.Api]; !found {
		return false, errors.New("unknown API: " + t.Api)
	}
	return true, nil
}

func (t configType) GetApiType() string {
	switch {
	case t.isSettingsPresent():
		return t.Settings.Schema
	case t.isClassicPresent():
		return t.Api
	default:
		return ""
	}
}
