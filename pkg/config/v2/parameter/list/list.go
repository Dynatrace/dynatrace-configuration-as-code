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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
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
	Values []value.ValueParameter // TODO(CA-1517): allow for parameter.Parameter
}

func New(values []value.ValueParameter) *ListParameter {
	return &ListParameter{Values: values}
}

// this forces the compiler to check if ValueParameter is of type Parameter
var _ parameter.Parameter = (*ListParameter)(nil)

func (p *ListParameter) GetType() string {
	return ListParameterType
}

func (p *ListParameter) GetReferences() []parameter.ParameterReference {
	// TODO(CA-1517): implement handling of references in list values
	// the value parameter cannot have references, as it is a simple value
	return []parameter.ParameterReference{}
}

func (p *ListParameter) ResolveValue(c parameter.ResolveContext) (interface{}, error) {

	listValues := make([]string, len(p.Values))
	for i, v := range p.Values {
		resolved, err := v.ResolveValue(c)
		if err != nil {
			return nil, err
		}
		listValues[i] = fmt.Sprintf(`"%s"`, resolved)
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

	result["values"] = toWritableValues(context, listParam.Values)

	return result, nil
}

// TODO(CA-1517): call the underlying parameter's Deserialzer/write methods
// toWriteableValues turns the underlying ValueParameters into a simple string list to write them in their short form
func toWritableValues(context parameter.ParameterWriterContext, values []value.ValueParameter) []interface{} {
	writableValues := make([]interface{}, len(values))
	for i, v := range values {
		v := v // to avoid implicit memory aliasing (gosec G601)
		if s, ok := v.Value.(string); ok {
			writableValues[i] = s
			continue
		}

		subCtxt := context
		subCtxt.Parameter = &v
		writableVal, _ := value.ValueParameterSerde.Serializer(subCtxt)

		writableVal["type"] = v.GetType()

		writableValues[i] = writableVal
	}
	return writableValues
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

	parameterSlice := make([]value.ValueParameter, len(valueSlice))
	for i, v := range valueSlice {

		if s, ok := v.(string); ok {
			parameterSlice[i] = value.ValueParameter{Value: s}
		} else {
			p, err := parseSubParameter(v, context)
			if err != nil {
				return nil, parameter.NewParameterParserError(context,
					fmt.Sprintf("malformed value `%v` at index %d: %v", v, i, err),
				)
			}
			parameterSlice[i] = p
		}
	}

	return New(parameterSlice), nil
}

func parseSubParameter(paramValue interface{}, context parameter.ParameterParserContext) (value.ValueParameter, error) {
	mapVal, ok := paramValue.(map[interface{}]interface{})
	if !ok {
		return value.ValueParameter{}, fmt.Errorf("malformed list entry `%v`", paramValue)
	}
	subValue := maps.ToStringMap(mapVal)
	subContext := parameter.ParameterParserContext{
		Coordinate:    context.Coordinate,
		Group:         context.Group,
		Environment:   context.Environment,
		ParameterName: context.ParameterName,
		Value:         subValue,
	}
	p, err := value.ValueParameterSerde.Deserializer(subContext)
	if err != nil {
		return value.ValueParameter{}, err
	}
	return *p.(*value.ValueParameter), nil
}
