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
)

// JsonValidationError is an error which contains more information about
// where the error appeared in the json file which was validated.
// It contains the fileName, line number, character number, and line
// content as additional information. Furthermore, it contains the original
// error (the cause) which happened during the json unmarshalling.
type JsonValidationError struct {

	// FileName is the file name (full qualified) where the error happened
	// This field is always filled.
	FileName string
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

func (e JsonValidationError) Error() string {
	return fmt.Sprintf("file %s is not a valid json: Error: %s", e.FileName, e.Cause.Error())
}

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

func (e *JsonValidationError) PrettyPrintError() {

	if e.ContainsLineInformation() {

		lengthOfLineNum := len(strconv.Itoa(e.LineNumber))
		whiteSpace := strings.Repeat(" ", lengthOfLineNum)
		whiteSpaceOffset := strings.Repeat(" ", e.CharacterNumberInLine-1)
		lineContent := strings.Replace(e.LineContent, "\t", " ", -1)
		previousLineContent := strings.Replace(e.PreviousLineContent, "\t", " ", -1)

		Log.Error("\t"+errorTemplate, e.FileName, e.LineNumber, e.CharacterNumberInLine,
			whiteSpace, previousLineContent,
			e.LineNumber, lineContent,
			whiteSpace, whiteSpaceOffset,
			whiteSpace, e.Cause.Error())
	}
}

// ValidateJson validates whether the json file is correct, by using the internal validation done
// when unmarshalling to a an object. As none of our jsons can actually be unmarshalled
// to a string, we catch that error, but report any other error as fatal. We then return the parsed
// json object.
func ValidateAndParseJson(jsonString string, filename string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &result)

	if jsonError, ok := err.(*json.SyntaxError); ok {
		return nil, mapError(jsonString, filename, int(jsonError.Offset), err)
	}
	if _, ok := err.(*json.UnmarshalTypeError); ok {
		// TODO actually check against the model (github issue #104)
		return result, nil
	}
	return result, nil
}

func ValidateJson(json string, filename string) error {
	_, err := ValidateAndParseJson(json, filename)

	return err
}

// mapError maps the json parsing error to a JsonValidationError which contains
// the line number, character number, and line in which the error happened
func mapError(input string, filename string, offset int, err error) (mappedError JsonValidationError) {

	if offset > len(input) || offset < 0 {
		return newEmptyErr(filename, err)
	}

	var characterCountToEndOfPrevLine = 0
	previousLineContent := ""
	lines := strings.Split(input, "\n")

	for i, line := range lines {
		if offset <= characterCountToEndOfPrevLine+len(line) {

			return JsonValidationError{
				FileName:              filename,
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

	return newEmptyErr(filename, err)
}

// newEmptyErr constructs an empty error without line number, character number,
// and line in which the error happened
func newEmptyErr(filename string, err error) JsonValidationError {
	return JsonValidationError{
		FileName:              filename,
		LineNumber:            -1,
		CharacterNumberInLine: -1,
		LineContent:           "",
		PreviousLineContent:   "",
		Cause:                 err,
	}
}
