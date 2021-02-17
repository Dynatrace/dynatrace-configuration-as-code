/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

import (
	"errors"
	"os"
)

// PrintError should pretty-print the error using a more user-friendly format
func PrintError(err error) {
	if ppError, ok := err.(JsonValidationError); ok {
		ppError.PrettyPrintError()
	} else {
		Log.Error("\t%s", err)
	}
}

func PrintErrors(errors []error) {
	for _, err := range errors {
		PrintError(err)
	}
}

func FailOnError(err error, msg string) {
	if err != nil {
		Log.Fatal(msg + ": " + err.Error())
		os.Exit(1)
	}
}

func CheckError(err error, msg string) bool {
	if err != nil {
		Log.Error(msg + ": " + err.Error())
		return true
	}
	return false
}

func CheckProperty(properties map[string]string, property string) (string, error) {

	prop, ok := properties[property]
	if !ok {
		return "", errors.New("Property " + property + " was not available")
	}
	return prop, nil
}
