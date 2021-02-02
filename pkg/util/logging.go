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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jcelliott/lumber"
)

// Log is the shared Lumber Logger logging to console and after calling SetupLogging also to file
var Log lumber.Logger = lumber.NewConsoleLogger(lumber.INFO)

var requestLogFile *os.File

// SetupLogging is used to initialize the shared file Logger once the necessary setup config is available
func SetupLogging(verbose bool) error {
	multiLog := lumber.NewMultiLogger()
	consoleLog := lumber.NewConsoleLogger(lumber.INFO)
	if verbose {
		consoleLog.Level(lumber.DEBUG)
	}
	multiLog.AddLoggers(consoleLog)

	if _, err := os.Stat(".logs"); os.IsNotExist(err) {
		err = os.Mkdir(".logs", 0777)

		if err != nil {
			FailOnError(err, "could not create directory")
		}
	}

	logName := ".logs" + string(os.PathSeparator) + time.Now().Format("20060102-150405") + ".log"
	fileLog, err := lumber.NewAppendLogger(logName)

	if err != nil {
		return err
	}

	fileLog.Level(lumber.DEBUG)
	multiLog.AddLoggers(fileLog)
	Log = multiLog

	if file, found := os.LookupEnv("MONACO_REQUEST_LOG"); found {
		logFilePath, err := filepath.Abs(file)

		if err != nil {
			return err
		}

		Log.Debug("request log activated at %s", logFilePath)
		return setupRequestLogging(logFilePath)
	} else {
		Log.Debug("request log not activated")
	}

	return err
}

func setupRequestLogging(file string) error {
	handle, err := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	requestLogFile = handle

	return nil
}

func IsRequestLoggingActive() bool {
	return requestLogFile != nil
}

func LogRequest(url string, method string, content string) error {
	if content == "" {
		requestLogFile.WriteString(fmt.Sprintf(`url: %s
method: %s
==========
`, url, method))
	} else {
		requestLogFile.WriteString(fmt.Sprintf(`url: %s
method: %s
content:
%s
==========
`, url, method, content))
	}

	return requestLogFile.Sync()
}
