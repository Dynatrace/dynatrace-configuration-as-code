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

package test

import "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"

const DummyParameterType = "dummy"

type DummyParameter struct {
	Value      interface{}
	Err        error
	References []parameter.ParameterReference
}

func (d *DummyParameter) GetType() string {
	return DummyParameterType
}

func (d *DummyParameter) GetReferences() []parameter.ParameterReference {
	return d.References
}

func (d *DummyParameter) ResolveValue(context parameter.ResolveContext) (interface{}, error) {
	if d.Err != nil {
		return nil, d.Err
	}

	return d.Value, nil
}

var _ parameter.Parameter = (*DummyParameter)(nil)
