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

package list

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"strings"
)

// ListParameterType specifies the type of the parameter used in config files
const ListParameterType = "list"

var ListParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeListParameter,
	Deserializer: parseListParameter,
}

// ListParameter represents a simple list of string values.
type ListParameter struct {
	Values []string
}

func New(values []string) *ListParameter {
	return &ListParameter{Values: values}
}

// this forces the compiler to check if ValueParameter is of type Parameter
var _ parameter.Parameter = (*ListParameter)(nil)

func (p *ListParameter) GetType() string {
	return ListParameterType
}

func (p *ListParameter) GetReferences() []parameter.ParameterReference {
	// TODO allow references?
	// the value parameter cannot have references, as it is a simple value
	return []parameter.ParameterReference{}
}

func (p *ListParameter) ResolveValue(_ parameter.ResolveContext) (interface{}, error) {

	listValues := make([]string, len(p.Values))
	for i, s := range p.Values {
		escaped, err := util.EscapeSpecialCharactersInValue(s, util.FullStringEscapeFunction)
		if err != nil {
			return nil, err
		}
		listValues[i] = fmt.Sprintf(`"%s"`, escaped)
	}
	list := fmt.Sprintf("[ %s ]", strings.Join(listValues, ","))
	return list, nil
}

func writeListParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	listParam, ok := context.Parameter.(*ListParameter)

	if !ok {
		return nil, parameter.NewParameterWriterError(context, "unexpected type. parameter is not of type `ValueParameter`")
	}

	result := make(map[string]interface{})

	result["values"] = listParam.Values

	return result, nil
}

func parseListParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	values, ok := context.Value["values"]
	if !ok {
		return nil, parameter.NewParameterParserError(context, "missing property `values`")
	}

	valueSlice, ok := values.([]interface{})
	if !ok {
		return nil, parameter.NewParameterParserError(context, "malformed property `values` - expected list")
	}

	stringSlice := make([]string, len(valueSlice))
	for i, v := range valueSlice {
		s, ok := v.(string)
		if !ok {
			return nil, parameter.NewParameterParserError(context,
				fmt.Sprintf("malformed value `%s` at index %d - expected list string", s, i),
			)
		}
		stringSlice[i] = s
	}

	return New(stringSlice), nil
}
