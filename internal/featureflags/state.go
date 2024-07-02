/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package featureflags

import (
	"fmt"
	"strings"
)

func AnyModified() bool {
	for _, v := range Temporary {
		enabled, def := v.Value()
		if enabled != def {
			return false
		}
	}
	return true
}

func StateInfo() string {
	s := strings.Builder{}
	_, _ = fmt.Fprintf(&s, "Temporary Feature Flags:\n\n")
	for _, v := range Temporary {
		enabled, def := v.Value()
		_, _ = fmt.Fprintf(&s, "\t%v: %v (default:%v)\n", v.envName, enabled, def)
	}
	return s.String()
}
