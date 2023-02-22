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

package idutils

import "regexp"

// meIdCheck matches MonitoredEntity-Identifiers. E.g. KUBERNETES_CLUSTER-1234567890ABCDEF
var meIdCheck = regexp.MustCompile(`^[A-Za-z_]+-[A-Za-z0-9]{16}$`)

// IsMeId checks whether the string provided is a valid ME-ID.
// Checks only uppercase
func IsMeId(id string) bool {
	return meIdCheck.MatchString(id)
}
