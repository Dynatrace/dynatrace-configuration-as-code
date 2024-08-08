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

package loader

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/mutlierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/spf13/afero"
)

type ParamTypeParsers = map[string]parameter.ParameterSerDe

func parseParameters(fs afero.Fs, parsers ParamTypeParsers, in map[string]interface{}) (map[string]parameter.Parameter, error) {

	parameters := make(map[string]parameter.Parameter)
	var errs []error

	for name, param := range in {
		if _, found := parameters[name]; found {
			continue
		}

		result, err := parseParameter(fs, parsers, name, param)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		parameters[name] = result
	}

	if errs != nil {
		return nil, mutlierror.New(errs...)
	}

	return parameters, nil
}

func parseParameter(fs afero.Fs, parsers ParamTypeParsers, name string, param interface{}) (parameter.Parameter, error) {

	if val, ok := param.(map[interface{}]interface{}); ok {
		parameterType := toString(val["type"])

		if parameterType == reference.ReferenceParameterType || parameterType == compound.CompoundParameterType {
			return nil, fmt.Errorf("invalid parameter type `%s` for global parameter %q", parameterType, name)
		}

		serDe, found := parsers[parameterType]

		if !found {
			return nil, fmt.Errorf("unknown parameter type `%s` for global parameter %q", parameterType, name)
		}

		return serDe.Deserializer(parameter.ParameterParserContext{
			Fs:            fs,
			ParameterName: name,
			Value:         maps.ToStringMap(val),
		})
	}

	return valueParam.New(param), nil
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
