//go:build unit
// +build unit

/**
 * @license
 * Copyright 2022 Dynatrace LLC
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

package util

import "testing"

func TestIsEnvVariable(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			"env reference detected correctly",
			"{{ .Env.ENV_VAR }}",
			true,
		},
		{
			"env reference in a longer string",
			"here's some other text, and this: {{ .Env.ENV_VAR }} is an environment variable",
			true,
		},
		{
			"env reference with spaces detected correctly",
			"   {{         .Env.ENV_VAR      }}    ",
			true,
		},
		{
			"several env references detected correctly",
			"{{ .Env.ENV_VAR }} {{ .Env.ENV_VAR2 }}",
			true,
		},
		{
			"non-env var reference does not match",
			"{{ .SomeOtherValue }}",
			false,
		},
		{
			"random string does not match",
			"just a random string { }",
			false,
		},
		{
			"random string with reference brackets does not match",
			"just a random string {{ }}",
			false,
		},
		{
			"random string containing Env does not match",
			"just a random string about an Environment",
			false,
		},
		{
			"reference without curly brackets does not match",
			".Env.VARIABLE",
			false,
		},
		{
			"random string containing .Env. does not match",
			"you use .Env. to signify and environment variable",
			false,
		},
		{
			"url value does not match",
			"https://www.dynatrace.com",
			false,
		},
		{
			"url value containing env does not match",
			"https://www.some.env.dynatrace.com",
			false,
		},
		{
			"url value containing capital Env does not match",
			"https://www.Some.Env.Dynatrace.com",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEnvVariable(tt.input); got != tt.want {
				t.Errorf("IsEnvVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_trimToEnvVariableName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"simple env reference trimmed correctly",
			"{{ .Env.ENV_VAR }}",
			"ENV_VAR",
		},
		{
			"empty string returned unchanged",
			"    ",
			"    ",
		},
		{
			"env reference with spaces trimmed correctly",
			"   {{         .Env.ENV_VAR      }}    ",
			"ENV_VAR",
		},
		{
			"env reference in a longer string parsed correctly",
			"here's some other text, and this: {{ .Env.ENV_VAR }} is an environment variable",
			"ENV_VAR",
		},
		{
			"several env references returns first",
			"{{ .Env.ENV_VAR }} {{ .Env.ENV_VAR2 }}",
			"ENV_VAR",
		},
		{
			"non-env var reference returned unchanged",
			"{{ .SomeOtherValue }}",
			"{{ .SomeOtherValue }}",
		},
		{
			"wrong string returned unchanged",
			"just a random string { }",
			"just a random string { }",
		},
		{
			"random string with reference brackets returned unchanged",
			"just a random string {{ }}",
			"just a random string {{ }}",
		},
		{
			"random string containing Env returned unchanged",
			"just a random string about an Environment",
			"just a random string about an Environment",
		},
		{
			"reference without curly brackets does not match",
			".Env.VARIABLE",
			".Env.VARIABLE",
		},
		{
			"random string containing .Env. returned unchanged",
			"you use .Env. to signify and environment variable",
			"you use .Env. to signify and environment variable",
		},
		{
			"url value returned unchanged",
			"https://www.dynatrace.com",
			"https://www.dynatrace.com",
		},
		{
			"url value containing env returned unchanged",
			"https://www.some.env.dynatrace.com",
			"https://www.some.env.dynatrace.com",
		},
		{
			"url value containing capital Env returned unchanged",
			"https://www.Some.Env.Dynatrace.com",
			"https://www.Some.Env.Dynatrace.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimToEnvVariableName(tt.input); got != tt.want {
				t.Errorf("trimToEnvVariableName() = %v, want %v", got, tt.want)
			}
		})
	}
}
