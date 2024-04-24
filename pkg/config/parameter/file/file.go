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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/spf13/afero"
	"path/filepath"
)

const FileParameterType = "file"

var FileParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeFileValueParameter,
	Deserializer: parseFileValueParameter,
}

type FileParameter struct {
	WorkingDir string
	Folder     string
	Path       string
}

func (f *FileParameter) GetType() string {
	return FileParameterType
}

func (f *FileParameter) GetReferences() []parameter.ParameterReference {
	// file parameters cannot have references
	return []parameter.ParameterReference{}
}

func (f *FileParameter) ResolveValue(context parameter.ResolveContext) (interface{}, error) {
	content, err := afero.ReadFile(afero.NewOsFs(), filepath.Join(f.WorkingDir, f.Folder, f.Path))
	if err != nil {
		return nil, parameter.NewParameterResolveValueError(context, fmt.Sprintf("unable to read from file %s: %v", f.Path, err))
	}
	return template.EscapeSpecialCharactersInValue(string(content), template.FullStringEscapeFunction)

}

func parseFileValueParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	if path, ok := context.Value["path"]; ok {
		return &FileParameter{WorkingDir: context.WorkingDir, Folder: context.Folder, Path: strings.ToString((path))}, nil
	}
	return nil, parameter.NewParameterParserError(context, "missing property `path`")
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
