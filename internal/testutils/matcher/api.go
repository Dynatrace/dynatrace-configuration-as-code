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

package matcher

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"go.uber.org/mock/gomock"
)

// EqAPI returns the gomock matcher that matches two API.
//
// API is equal if both API.ID and API.URLPath are equal
func EqAPI(x api.API) gomock.Matcher {
	type matcher interface {
		Matches(x any) bool
	}

	return struct {
		matcher
		gomock.GotFormatter
		fmt.Stringer
	}{
		matcher: eqAPI(x),
		GotFormatter: gomock.GotFormatterFunc(func(i any) string {
			if a, ok := i.(api.API); ok {
				return fmt.Sprintf("%q{url: %q} (%T)", a, a.URLPath, a)
			}
			return fmt.Sprintf("%#v (%T)", i, i)
		}),
		Stringer: gomock.StringerFunc(func() string {
			return fmt.Sprintf("%q{url: %q} (%T)", x, x.URLPath, x)
		}),
	}
}

type eqAPI api.API

func (m eqAPI) Matches(x any) bool {
	if a, ok := x.(api.API); ok {
		return a.ID == m.ID && a.URLPath == m.URLPath
	}
	return false
}
