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

package log

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
)

var (
	requestLogFile  *os.File
	responseLogFile *os.File
)

func setupRequestLog() error {
	if logFilePath, found := os.LookupEnv("MONACO_REQUEST_LOG"); found {
		logFilePath, err := filepath.Abs(logFilePath)

		if err != nil {
			return err
		}

		Debug("request log activated at %s", logFilePath)
		handle, err := prepareLogFile(logFilePath)

		if err != nil {
			return err
		}

		requestLogFile = handle
	} else {
		Debug("request log not activated")
	}

	return nil
}

func setupResponseLog() error {
	if logFilePath, found := os.LookupEnv("MONACO_RESPONSE_LOG"); found {
		logFilePath, err := filepath.Abs(logFilePath)

		if err != nil {
			return err
		}

		Debug("response log activated at %s", logFilePath)
		handle, err := prepareLogFile(logFilePath)

		if err != nil {
			return err
		}

		responseLogFile = handle
	} else {
		Debug("response log not activated")
	}

	return nil
}

func prepareLogFile(file string) (*os.File, error) {
	return os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
}

func IsRequestLoggingActive() bool {
	return requestLogFile != nil
}

func IsResponseLoggingActive() bool {
	return responseLogFile != nil
}

func LogRequest(id string, request *http.Request) error {
	if !IsRequestLoggingActive() {
		return nil
	}

	var dumpBody = shouldDumpBody(request.Header)

	dump, err := httputil.DumpRequestOut(request, dumpBody)

	if err != nil {
		return err
	}

	stringDump := string(dump)

	_, err = requestLogFile.WriteString(fmt.Sprintf(`Request-ID: %s
%s
=========================
`, id, stringDump))

	if err != nil {
		return err
	}

	return requestLogFile.Sync()
}

func LogResponse(id string, response *http.Response, body string) error {
	if !IsResponseLoggingActive() {
		return nil
	}

	dump, err := httputil.DumpResponse(response, false)

	if err != nil {
		return err
	}

	if id != "" {
		_, err = responseLogFile.WriteString(fmt.Sprintf("Request-ID: %s\n", id))

		if err != nil {
			return err
		}
	}

	stringDump := string(dump)

	_, err = responseLogFile.WriteString(fmt.Sprintf(`%s
%s

=========================
`, stringDump, body))

	if err != nil {
		return err
	}

	return responseLogFile.Sync()
}

var dumpCasePrefixes = []string{
	"text/",
	"application/xml",
	"application/json",
}

func shouldDumpBody(headers http.Header) bool {
	contentType := headers.Get("Content-Type")

	return shouldDumpBodyForContentType(contentType)
}

func shouldDumpBodyForContentType(contentType string) bool {
	for _, prefix := range dumpCasePrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}

	return false
}
