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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
)

// JsonValidationError is an error which contains more information about
// where the error appeared in the json file which was validated.
// It contains the fileName, line number, character number, and line
// content as additional information. Furthermore, it contains the original
// error (the cause) which happened during the json unmarshalling.
type JsonValidationError struct {

	// FileName is the file name (full qualified) where the error happened
	// This field is always filled.
	Location Location
	// LineNumber contains the line number (starting by one) where the error happened
	// If we don't have the information, this is -1.
	LineNumber int
	// CharacterNumberInLine contains the character number (starting by one) where
	// the error happened. If we don't have the information, this is -1.
	CharacterNumberInLine int
	// LineContent contains the full line content of where the error happened
	// If we don't have the information, this is an empty string.
	LineContent string
	// PreviousLineContent contains the full line content of the line before LineContent
	// If we don't have the information, this is an empty string.
	PreviousLineContent string
	// Cause is the original error which happened during the json unmarshalling.
	Cause error
}

var (
	_ PrettyPrintableError = (*JsonValidationError)(nil)
)

func (e *JsonValidationError) Error() string {
	return fmt.Sprintf("rendered template `%s` is not a valid json: Error: %s",
		e.Location.TemplateFilePath, e.Cause.Error())
}

var (
	_ error = (*JsonValidationError)(nil)
)

// ContainsLineInformation indicates whether additional line information is present in
// the error.
func (e *JsonValidationError) ContainsLineInformation() bool {
	return e.LineNumber > 0 && e.CharacterNumberInLine > 0 && e.LineContent != ""
}

const errorTemplate = `File did not contain valid json:
 --> %s:%d:%d
 %s | %s
 %d | %s
 %s | %s^^^
 %s - Cause: %s
`

func (e *JsonValidationError) PrettyError() string {

	if e.ContainsLineInformation() {
		lengthOfLineNum := len(strconv.Itoa(e.LineNumber))
		whiteSpace := strings.Repeat(" ", lengthOfLineNum)
		whiteSpaceOffset := strings.Repeat(" ", e.CharacterNumberInLine-1)
		lineContent := strings.Replace(e.LineContent, "\t", " ", -1)
		previousLineContent := strings.Replace(e.PreviousLineContent, "\t", " ", -1)

		return fmt.Sprintf(errorTemplate,
			e.Location.TemplateFilePath, e.LineNumber, e.CharacterNumberInLine,
			whiteSpace, previousLineContent,
			e.LineNumber, lineContent,
			whiteSpace, whiteSpaceOffset,
			whiteSpace, e.Cause.Error())
	}

	return e.Error()
}

type Location struct {
	Coordinate       coordinate.Coordinate
	Group            string
	Environment      string
	TemplateFilePath string
}

func ValidateJson(data string, location Location) error {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)

	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *json.SyntaxError:
		return mapError(data, location, int(e.Offset), err)
	case *json.UnmarshalTypeError:
		// TODO actually check against the model (github issue #104)
		return nil
	}

	return nil
}

// mapError maps the json parsing error to a JsonValidationError which contains
// the line number, character number, and line in which the error happened
func mapError(input string, location Location, offset int, err error) error {
	if offset > len(input) || offset < 0 {
		return newEmptyErr(location, err)
	}

	var characterCountToEndOfPrevLine = 0
	previousLineContent := ""
	lines := strings.Split(input, "\n")

	for i, line := range lines {
		if offset <= characterCountToEndOfPrevLine+len(line) {

			return &JsonValidationError{
				Location:              location,
				LineNumber:            i + 1, // humans tend to count from 1
				CharacterNumberInLine: offset - characterCountToEndOfPrevLine,
				LineContent:           line,
				PreviousLineContent:   previousLineContent,
				Cause:                 err,
			}
		}
		characterCountToEndOfPrevLine += len(line) + 1 // +1 for newline
		previousLineContent = line
	}

	return newEmptyErr(location, err)
}

// newEmptyErr constructs an empty error without line number, character number,
// and line in which the error happened
func newEmptyErr(location Location, err error) error {
	return &JsonValidationError{
		Location:              location,
		LineNumber:            -1,
		CharacterNumberInLine: -1,
		LineContent:           "",
		PreviousLineContent:   "",
		Cause:                 err,
	}
}
