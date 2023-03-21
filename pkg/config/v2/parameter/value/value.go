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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
)

// ValueParameterType specifies the type of the parameter used in config files
const ValueParameterType = "value"

var ValueParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeValueParameter,
	Deserializer: parseValueParameter,
}

// ValueParameter represents a simple value. the value has to be
// resolve at config load time.
type ValueParameter struct {
	Value interface{}
}

func New(value interface{}) *ValueParameter {
	return &ValueParameter{Value: value}
}

// this forces the compiler to check if ValueParameter is of type Parameter
var _ parameter.Parameter = (*ValueParameter)(nil)

func (p *ValueParameter) GetType() string {
	return ValueParameterType
}

func (p *ValueParameter) GetReferences() []parameter.ParameterReference {
	// the value parameter cannot have references, as it is a simple value
	return []parameter.ParameterReference{}
}

func (p *ValueParameter) ResolveValue(_ parameter.ResolveContext) (interface{}, error) {
	return template.EscapeSpecialCharactersInValue(p.Value, template.FullStringEscapeFunction)
}

// parseValueParameter parses a given context into an instance of ValueParameter.
// the only required property is `value`.
func parseValueParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	if val, ok := context.Value["value"]; ok {
		return New(val), nil
	}

	return nil, parameter.NewParameterParserError(context, "missing property `value`")
}

func writeValueParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	valueParam, ok := context.Parameter.(*ValueParameter)

	if !ok {
		return nil, parameter.NewParameterWriterError(context, "unexpected type. parameter is not of type `ValueParameter`")
	}

	return map[string]interface{}{
		"value": valueParam.Value,
	}, nil
}
