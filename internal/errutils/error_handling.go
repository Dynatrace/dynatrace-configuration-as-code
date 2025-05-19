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

package errutils

import (
	"errors"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
)

type PrettyPrintableError interface {
	PrettyError() string
}

func ErrorString(err error) string {
	if err == nil {
		return "<nil>"
	}

	var prettyPrintError PrettyPrintableError

	if errors.As(err, &prettyPrintError) {
		return prettyPrintError.PrettyError()
	} else {
		return err.Error()
	}
}

// PrintError should pretty-print the error using a more user-friendly format
func PrintError(err error) {
	var prettyPrintError PrettyPrintableError

	if errors.As(err, &prettyPrintError) {
		log.WithFields(field.Error(err)).Error("%s", prettyPrintError.PrettyError())
	} else if err != nil {
		log.WithFields(field.Error(err)).Error("%s", err)
	}
}

func PrintErrors(errors []error) {
	for _, err := range errors {
		PrintError(err)
	}
}

func CheckError(err error, msg string) bool {
	if err != nil {
		log.WithFields(field.Error(err)).Error("%s: %s", msg, err)
		return true
	}
	return false
}

// PrintWarning prints the error as a warning.
// The error is pretty-printed if the error implements the PrettyPrintableError interface
func PrintWarning(err error) {
	var prettyPrintError PrettyPrintableError

	if errors.As(err, &prettyPrintError) {
		log.WithFields(field.Error(err)).Warn("%s", prettyPrintError.PrettyError())
	} else if err != nil {
		log.WithFields(field.Error(err)).Warn("%s", err)
	}
}

func PrintWarnings(errors []error) {
	for _, err := range errors {
		PrintWarning(err)
	}
}
