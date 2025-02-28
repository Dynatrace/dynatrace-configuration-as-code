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
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	tmpl "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

const FileParameterType = "file"

var FileParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeFileValueParameter,
	Deserializer: parseFileValueParameter,
}

type FileParameter struct {
	Fs                   afero.Fs
	Path                 string
	Escape               bool
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
	parameterTmpl, err := tmpl.NewFileTemplate(f.Fs, f.Path)
	if err != nil {
		return nil, parameter.NewParameterResolveValueError(context, err.Error())
	}

	resolvedParameterValues, err := f.getReferencedParameterValues(context)
	if err != nil {
		return nil, err
	}

	strContent, err := tmpl.Render(parameterTmpl, resolvedParameterValues)
	if err != nil {
		return nil, parameter.NewParameterResolveValueError(context, err.Error())
	}
	if f.Escape {
		return template.EscapeSpecialCharactersInValue(strContent, template.FullStringEscapeFunction)
	}

	return strContent, nil
}

// getReferencedParameterValues gets the resolved values of parameters defined in the `references` section of the file parameter. If an unknown parameter is referenced, an error is returned.
// These are the only properties that may be used within the template.
func (f *FileParameter) getReferencedParameterValues(context parameter.ResolveContext) (map[string]any, error) {
	resolvedParameterValues := make(map[string]any)
	for _, param := range f.referencedParameters {
		value, ok := context.ResolvedParameterValues[param.Property]
		if !ok {
			return nil, fmt.Errorf("unknown parameter '%s'", param.Property)
		}
		resolvedParameterValues[param.Property] = value
	}
	return resolvedParameterValues, nil
}

func parseFileValueParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	if context.Fs == nil {
		return nil, parameter.NewParameterParserError(context, "missing filesystem handle to load parameter")
	}

	p, ok := context.Value["path"]
	if !ok {
		return nil, parameter.NewParameterParserError(context, "missing property `path`")
	}

	path := strings.ToString(p)
	path = filepath.FromSlash(path)

	path = filepath.Join(context.Folder, path)
	escape := true
	if escapedValue, ok := context.Value["escape"]; ok {
		escapeBool, ok := escapedValue.(bool)
		if !ok {
			return nil, parameter.NewParameterParserError(context, "property `escape` must be a boolean")
		}
		escape = escapeBool
	}

	references, ok := context.Value["references"]
	if !ok {
		return &FileParameter{Fs: context.Fs, Path: strings.ToString(path), Escape: escape}, nil
	}

	referencedParameterSlice, ok := references.([]interface{})
	if !ok {
		return nil, parameter.NewParameterParserError(context, "malformed value `references`")
	}

	referencedParameters, err := parameter.ToParameterReferences(referencedParameterSlice, context.Coordinate)
	if err != nil {
		return nil, parameter.NewParameterParserError(context, fmt.Sprintf("invalid parameter references: %v", err))
	}

	return &FileParameter{Fs: context.Fs, Path: strings.ToString(path), Escape: escape, referencedParameters: referencedParameters}, nil

}

func writeFileValueParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	fileParam, ok := context.Parameter.(*FileParameter)

	if !ok {
		return nil, parameter.NewParameterWriterError(context, "unexpected type. parameter is not of type `FileParameter`")
	}

	result := make(map[string]interface{})

	result["path"] = fileParam.Path
	result["escape"] = fileParam.Escape

	return result, nil
}
