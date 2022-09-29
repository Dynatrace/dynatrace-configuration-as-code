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

package value

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

// LegacyValueParameter behaves as a ValueParameter for parameters loaded from v1 configuration
// While ValueParameters are fully string escaped when Resolved,
// LegacyValueParameters only escape newlines, to not break v1 use-cases such as list definitions.
type LegacyValueParameter struct {
	ValueParameter
}

// this forces the compiler to check if LegacyValueParameter is of type Parameter
var _ parameter.Parameter = (*LegacyValueParameter)(nil)

func (p *LegacyValueParameter) ResolveValue(_ parameter.ResolveContext) (interface{}, error) {
	return util.EscapeSpecialCharactersInValue(p.Value, util.SimpleStringEscapeFunction)
}
