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

package compound

import (
	"bytes"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/strings"
	template2 "github.com/dynatrace/dynatrace-configuration-as-code/internal/template"
	templ "text/template" // nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
)

// CompoundParameterType specifies the type of the parameter used in config files
const CompoundParameterType = "compound"

var CompoundParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeCompoundParameter,
	Deserializer: parseCompoundParameter,
}

type CompoundParameter struct {
	format               *templ.Template
	rawFormatString      string
	referencedParameters []parameter.ParameterReference
}

func New(name string, format string, referencedParameters []parameter.ParameterReference) (*CompoundParameter, error) {
	formatTempl, err := template.ParseTemplate(name, format)
	if err != nil {
		return &CompoundParameter{}, err
	}

	return &CompoundParameter{
		format:               formatTempl,
		rawFormatString:      format,
		referencedParameters: referencedParameters,
	}, nil
}

// this forces the compiler to check if CompoundParameter is of type Parameter
var _ parameter.Parameter = (*CompoundParameter)(nil)

func (p *CompoundParameter) GetType() string {
	return CompoundParameterType
}

func (p *CompoundParameter) GetReferences() []parameter.ParameterReference {
	return p.referencedParameters
}

func (p *CompoundParameter) ResolveValue(context parameter.ResolveContext) (interface{}, error) {
	compoundData := make(map[string]interface{})

	for _, param := range p.referencedParameters {
		compoundData[param.Property] = context.ResolvedParameterValues[param.Property]
	}

	out := bytes.Buffer{}
	err := p.format.Execute(&out, compoundData)

	if err != nil {
		return nil, fmt.Errorf("error resolving compound value: %w", err)
	}

	str := out.String()
	return template2.EscapeSpecialCharactersInValue(str, template2.FullStringEscapeFunction)

}

// parseCompoundParameter parses a given context into an instance of CompoundParameter.
// This requires a string `format` and a slice of strings `references`, where `format`
// is a template string and `references` are all the used references in `format` refering
// to other parameters within the config.
func parseCompoundParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	format, ok := context.Value["format"]
	if !ok {
		return nil, parameter.NewParameterParserError(context, "missing property `format`")
	}

	references, ok := context.Value["references"]
	if !ok {
		return nil, parameter.NewParameterParserError(context, "missing property `references`")
	}

	referencedParameterSlice, ok := references.([]interface{})
	if !ok {
		return nil, parameter.NewParameterParserError(context, "malformed value `references`")
	}

	referencedParameters, err := toParameterReferences(referencedParameterSlice, context.Coordinate)
	if err != nil {
		return nil, parameter.NewParameterParserError(context, fmt.Sprintf("invalid parameter references: %v", err))
	}

	return New(context.ParameterName, strings.ToString(format), referencedParameters)
}

func writeCompoundParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	compoundParam, ok := context.Parameter.(*CompoundParameter)

	if !ok {
		return nil, parameter.NewParameterWriterError(context, "unexpected type. parameter is not of type `CompoundParameter`")
	}

	result := make(map[string]interface{})

	if compoundParam.rawFormatString == "" {
		return nil, parameter.NewParameterWriterError(context, "missing property `format`")
	}
	result["format"] = compoundParam.rawFormatString

	if len(compoundParam.referencedParameters) == 0 {
		return nil, parameter.NewParameterWriterError(context, "missing property `references`")
	}
	references := make([]interface{}, len(compoundParam.referencedParameters))

	for i, reference := range compoundParam.referencedParameters {
		references[i] = reference.Property
	}
	result["references"] = references

	return result, nil
}

func toParameterReferences(params []interface{}, coord coordinate.Coordinate) (paramRefs []parameter.ParameterReference, err error) {
	for _, param := range params {
		switch param.(type) {
		case []interface{}, map[interface{}]interface{}:
			return nil, fmt.Errorf("error creating parameter reference: %v is not a string", param)
		}

		paramRefs = append(paramRefs, parameter.ParameterReference{
			Config:   coord,
			Property: strings.ToString(param),
		})
	}
	return paramRefs, nil
}
