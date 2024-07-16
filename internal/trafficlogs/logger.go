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
	"bytes"
	"fmt"
	lib "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"strings"
	"sync"
)

const TrafficLogFilePrefixFormat = log.LogFileTimestampPrefixFormat

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
		requestFilePath:  RequestFilePath(),
		responseFilePath: ResponseFilePath(),
	}

	return tl
}

// RequestFilePath returns the full path of an HTTP request log file for the current execution time - if no traffic logs are written (yet) no file may exist at this path.
func RequestFilePath() string {
	return path.Join(log.LogDirectory, timeutils.TimeAnchor().Format(TrafficLogFilePrefixFormat)+"-"+"req.log")
}

// ResponseFilePath returns the full path of an HTTP response log file for the current execution time - if no traffic logs are written (yet) no file may exist at this path.
func ResponseFilePath() string {
	return path.Join(log.LogDirectory, timeutils.TimeAnchor().Format(TrafficLogFilePrefixFormat)+"-"+"resp.log")
}

// LogToFiles takes a record containing request and response information and tries to write it into the files
// created by this logger.
func (l *FileBasedLogger) LogToFiles(record lib.RequestResponse) {
	if req, ok := record.IsRequest(); ok {
		if err := l.logRequest(record.ID, req, req.Body); err != nil {
			l.logError(record.ID, "request", err)
		}
	}
	if resp, ok := record.IsResponse(); ok {
		if err := l.logResponse(record.ID, resp, resp.Body); err != nil {
			l.logError(record.ID, "response", err)
		}
	}
}

// Log takes request and response data and tries to write them into files created by this logger.
// Note: this method is used by the "old" rest.Client and not the one from configuration-as-code-core
func (l *FileBasedLogger) Log(req *http.Request, reqBody string, resp *http.Response, respBody string) error {

	requestId := ""
	requestId = uuid.NewString()

	if err := l.logRequest(requestId, req, io.NopCloser(strings.NewReader(reqBody))); err != nil {
		l.logError(requestId, "request", err)
	}

	if err := l.logResponse(requestId, resp, io.NopCloser(strings.NewReader(respBody))); err != nil {
		l.logError(requestId, "response", err)
	}

	return nil
}

func (l *FileBasedLogger) Close() {
	l.requestLogFile.Close()
	l.responseLogFile.Close()
}

func (l *FileBasedLogger) logRequest(id string, request *http.Request, body io.ReadCloser) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if err := l.openRequestLogFile(); err != nil {
		return fmt.Errorf("unable to open file for logging requests: %w", err)
	}

	// delete auth header
	req := request.Clone(request.Context())
	req.Header.Del("Authorization")

	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}

	// write id
	_, err = l.requestLogFile.WriteString(fmt.Sprintf("Request-ID: %s\n", id))
	if err != nil {
		return err
	}

	// write dump
	if _, err = l.requestLogFile.WriteString(fmt.Sprintf("%s", string(dump))); err != nil {
		return err
	}

	// write body
	if body != nil {
		defer body.Close()
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		maskedData := secret.Mask(data)
		if _, err = io.Copy(l.requestLogFile, bytes.NewReader(maskedData)); err != nil {
			return err
		}
	}

	// write end indicator
	if _, err = l.requestLogFile.WriteString("\n=========================\n\n"); err != nil {
		return err
	}
	return l.requestLogFile.Sync()
}

func (l *FileBasedLogger) logResponse(id string, response *http.Response, body io.ReadCloser) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if err := l.openResponseLogFile(); err != nil {
		return fmt.Errorf("unable to open file for logging responses: %w", err)
	}

	dump, err := httputil.DumpResponse(response, false)
	if err != nil {
		return err
	}

	// write id
	_, err = l.responseLogFile.WriteString(fmt.Sprintf("Request-ID: %s\n", id))
	if err != nil {
		return err
	}

	// write dump
	if _, err = l.responseLogFile.WriteString(fmt.Sprintf("%s", string(dump))); err != nil {
		return err
	}

	// write body
	if body != nil {
		defer body.Close()
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		maskedData := secret.Mask(data)
		if _, err = io.Copy(l.responseLogFile, bytes.NewReader(maskedData)); err != nil {
			return err
		}
	}

	// write end indicator
	if _, err = l.responseLogFile.WriteString("\n=========================\n\n"); err != nil {
		return err
	}
	return l.responseLogFile.Sync()
}
func (l *FileBasedLogger) openRequestLogFile() error {
	if l.requestLogFile == nil {

		if err := l.prepareLogDir(); err != nil {
			return err
		}

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

		if err := l.prepareLogDir(); err != nil {
			return err
		}

		responseLogFile, err := l.fs.OpenFile(l.responseFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		l.responseLogFile = responseLogFile
	}
	return nil
}

func (l *FileBasedLogger) prepareLogDir() error {
	if exists, err := afero.Exists(l.fs, log.LogDirectory); err != nil {
		return err
	} else if !exists {
		if err := l.fs.MkdirAll(log.LogDirectory, 0777); err != nil {
			return fmt.Errorf("unable to create log directory %s: %w", log.LogDirectory, err)
		}
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
