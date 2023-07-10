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

package trafficlogs

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"sync"
)

const TrafficLogFilePrefixFormat = log.TrafficLogFilePrefixFormat

type FileBasedLogger struct {
	fs               afero.Fs
	requestFilePath  string
	responseFilePath string
	requestLogFile   afero.File
	responseLogFile  afero.File
	lock             sync.Mutex
}

func NewFileBased() *FileBasedLogger {
	tl := &FileBasedLogger{
		fs:               afero.NewOsFs(),
		requestFilePath:  path.Join(log.LogDirectory, timeutils.TimeAnchor().Format(TrafficLogFilePrefixFormat)+"-"+"req.log"),
		responseFilePath: path.Join(log.LogDirectory, timeutils.TimeAnchor().Format(TrafficLogFilePrefixFormat)+"-"+"resp.log"),
	}

	return tl
}

func (l *FileBasedLogger) Log(req *http.Request, reqBody string, resp *http.Response, respBody string) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	requestId := ""
	requestId = uuid.NewString()
	if err := l.openRequestLogFile(); err != nil {
		return fmt.Errorf("unable to open file for logging requests: %w", err)
	}

	if err := l.logRequest(requestId, req, reqBody); err != nil {
		l.logError(requestId, "request", err)
	}

	if err := l.openResponseLogFile(); err != nil {
		return fmt.Errorf("unable to open file for logging responses: %w", err)
	}

	if err := l.logResponse(requestId, resp, respBody); err != nil {
		l.logError(requestId, "response", err)
	}

	return nil
}

func (l *FileBasedLogger) Close() {
	l.requestLogFile.Close()
	l.responseLogFile.Close()
}

func (l *FileBasedLogger) logRequest(id string, request *http.Request, body string) error {
	// delete auth header
	req := request.Clone(request.Context())
	req.Header.Del("Authorization")

	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}
	_, err = l.requestLogFile.WriteString(fmt.Sprintf("Request-ID: %s\n%s\n%s\n=========================\n", id, string(dump), body))
	if err != nil {
		return err
	}

	return l.requestLogFile.Sync()
}

func (l *FileBasedLogger) logResponse(id string, response *http.Response, body string) error {
	dump, err := httputil.DumpResponse(response, false)
	if err != nil {
		return err
	}

	if id != "" {
		_, err = l.responseLogFile.WriteString(fmt.Sprintf("Request-ID: %s\n", id))
		if err != nil {
			return err
		}
	}

	_, err = l.responseLogFile.WriteString(fmt.Sprintf("%s\n%s\n\n=========================\n", string(dump), body))
	if err != nil {
		return err
	}

	return l.responseLogFile.Sync()
}

func (l *FileBasedLogger) openRequestLogFile() error {
	if l.requestLogFile == nil {
		requestLogFile, err := l.fs.OpenFile(l.requestFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		l.requestLogFile = requestLogFile
	}
	return nil
}

func (l *FileBasedLogger) openResponseLogFile() error {
	if l.responseLogFile == nil {
		responseLogFile, err := l.fs.OpenFile(l.responseFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		l.responseLogFile = responseLogFile
	}
	return nil
}

func (l *FileBasedLogger) logError(requestId, logType string, err error) {
	logMessage := fmt.Sprintf("error while writing %s log", logType)
	if requestId != "" {
		logMessage += fmt.Sprintf(" for id `%s`", requestId)
	}

	log.WithFields(field.Error(err)).Warn(logMessage+": %v", err)
}
