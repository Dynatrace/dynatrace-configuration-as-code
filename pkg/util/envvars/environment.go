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

package envvars

import (
	"os"
	"sync"
)

type environment interface {
	Lookup(name string) (string, bool)
}

type osBasedEnvironment struct {
}

func (e *osBasedEnvironment) Lookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

type fakeEnvironment struct {
	data map[string]string
}

func (e *fakeEnvironment) Lookup(name string) (string, bool) {
	data, found := e.data[name]

	return data, found
}

var instance environment = &osBasedEnvironment{}
var mutex = &sync.RWMutex{}

func InstallFakeEnvironment(data map[string]string) {
	mutex.Lock()
	defer mutex.Unlock()

	instance = &fakeEnvironment{
		data,
	}
}

func InstallOsBased() {
	instance = &osBasedEnvironment{}
}

func Lookup(name string) (string, bool) {
	return instance.Lookup(name)
}
