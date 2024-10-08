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

package file

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	tmpl "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/spf13/afero"
)

const FileParameterType = "file"

var FileParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeFileValueParameter,
	Deserializer: parseFileValueParameter,
}

type FileParameter struct {
	Fs                   afero.Fs
	Path                 string
	referencedParameters []parameter.ParameterReference
}

func (f *FileParameter) GetType() string {
	return FileParameterType
}

func (f *FileParameter) GetReferences() []parameter.ParameterReference {
	if f.referencedParameters == nil {
		return []parameter.ParameterReference{}
	}
	return f.referencedParameters
}

func (f *FileParameter) ResolveValue(context parameter.ResolveContext) (interface{}, error) {
	parameterTmpl, err := tmpl.NewFileTemplate(f.Fs, cache.NoopCache[tmpl.FileBasedTemplate]{}, f.Path)
	if err != nil {
		return nil, parameter.NewParameterResolveValueError(context, err.Error())
	}

	strContent, err := tmpl.Render(parameterTmpl, context.ResolvedParameterValues)
	if err != nil {
		return nil, parameter.NewParameterResolveValueError(context, err.Error())
	}

	return template.EscapeSpecialCharactersInValue(strContent, template.FullStringEscapeFunction)
}

func parseFileValueParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	if context.Fs == nil {
		return nil, parameter.NewParameterParserError(context, "missing filesystem handle to load parameter")
	}

	path, ok := context.Value["path"]
	if !ok {
		return nil, parameter.NewParameterParserError(context, "missing property `path`")
	}

	references, ok := context.Value["references"]
	if !ok {
		return &FileParameter{Fs: context.Fs, Path: strings.ToString((path))}, nil
	}

	referencedParameterSlice, ok := references.([]interface{})
	if !ok {
		return nil, parameter.NewParameterParserError(context, "malformed value `references`")
	}

	referencedParameters, err := parameter.ToParameterReferences(referencedParameterSlice, context.Coordinate)
	if err != nil {
		return nil, parameter.NewParameterParserError(context, fmt.Sprintf("invalid parameter references: %v", err))
	}

	return &FileParameter{Fs: context.Fs, Path: strings.ToString((path)), referencedParameters: referencedParameters}, nil

}

func writeFileValueParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	fileParam, ok := context.Parameter.(*FileParameter)

	if !ok {
		return nil, parameter.NewParameterWriterError(context, "unexpected type. parameter is not of type `FileParameter`")
	}

	result := make(map[string]interface{})

	result["path"] = fileParam.Path

	return result, nil
}
