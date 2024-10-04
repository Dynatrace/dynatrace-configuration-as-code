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
	"bufio"
	"bytes"
	"errors"
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

var tr *trafficLogger
var once sync.Once

type trafficLogger struct {
	fs            afero.Fs
	reqFilePath   string
	respFilePath  string
	reqLogFile    afero.File
	respLogFile   afero.File
	respBufWriter *bufio.Writer
	reqBufWriter  *bufio.Writer
	lock          sync.Mutex
}

func GetInstance() *trafficLogger {
	once.Do(func() {
		tr = &trafficLogger{
			fs:           afero.NewOsFs(),
			reqFilePath:  RequestFilePath(),
			respFilePath: ResponseFilePath(),
		}
	})
	return tr
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
func (l *trafficLogger) LogToFiles(record lib.RequestResponse) {
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
func (l *trafficLogger) Log(req *http.Request, reqBody string, resp *http.Response, respBody string) error {

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

func (l *trafficLogger) Sync() error {
	l.lock.Lock()
	defer l.lock.Unlock()
	var errs []error
	if l.reqLogFile != nil {
		if err := l.reqBufWriter.Flush(); err != nil {
			errs = append(errs, err)
		}
		if err := l.reqLogFile.Sync(); err != nil {
			errs = append(errs, err)
		}
		l.reqLogFile = nil
	}

	if l.respLogFile != nil {
		if err := l.respBufWriter.Flush(); err != nil {
			errs = append(errs, err)
		}
		if err := l.respLogFile.Sync(); err != nil {
			errs = append(errs, err)
		}
		l.respLogFile = nil
	}
	return errors.Join(errs...)
}

func (l *trafficLogger) Close() {
	l.reqLogFile.Close()
	l.respLogFile.Close()
}

func (l *trafficLogger) logRequest(id string, request *http.Request, body io.ReadCloser) error {
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
	_, err = l.reqBufWriter.WriteString(fmt.Sprintf("Request-ID: %s\n", id))
	if err != nil {
		return err
	}

	// write dump
	if _, err = l.reqBufWriter.WriteString(fmt.Sprintf("%s", string(dump))); err != nil {
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
		if _, err = io.Copy(l.reqBufWriter, bytes.NewReader(maskedData)); err != nil {
			return err
		}
	}

	// write end indicator
	if _, err = l.reqBufWriter.WriteString("\n=========================\n\n"); err != nil {
		return err
	}
	return nil
}

func (l *trafficLogger) logResponse(id string, response *http.Response, body io.ReadCloser) error {
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
	_, err = l.respBufWriter.WriteString(fmt.Sprintf("Request-ID: %s\n", id))
	if err != nil {
		return err
	}

	// write dump
	if _, err = l.respBufWriter.WriteString(fmt.Sprintf("%s", string(dump))); err != nil {
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
		if _, err = io.Copy(l.respBufWriter, bytes.NewReader(maskedData)); err != nil {
			return err
		}
	}

	// write end indicator
	if _, err = l.respBufWriter.WriteString("\n=========================\n\n"); err != nil {
		return err
	}
	return nil
}

func (l *trafficLogger) openRequestLogFile() error {
	if l.reqLogFile == nil {
		var err error
		if l.reqLogFile, l.reqBufWriter, err = l.obtainFileAndWriter(l.reqFilePath); err != nil {
			return err
		}
	}
	return nil
}

func (l *trafficLogger) openResponseLogFile() error {
	if l.respLogFile == nil {
		var err error
		if l.respLogFile, l.respBufWriter, err = l.obtainFileAndWriter(l.respFilePath); err != nil {
			return err
		}
	}
	return nil
}

func (l *trafficLogger) obtainFileAndWriter(path string) (afero.File, *bufio.Writer, error) {
	if err := l.prepareLogDir(); err != nil {
		return nil, nil, err
	}

	file, err := l.fs.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	return file, bufio.NewWriter(file), nil
}

func (l *trafficLogger) prepareLogDir() error {
	if exists, err := afero.Exists(l.fs, log.LogDirectory); err != nil {
		return err
	} else if !exists {
		if err := l.fs.MkdirAll(log.LogDirectory, 0777); err != nil {
			return fmt.Errorf("unable to create log directory %s: %w", log.LogDirectory, err)
		}
	}
	return nil
}

func (l *trafficLogger) logError(requestId, logType string, err error) {
	logMessage := fmt.Sprintf("error while writing %s log", logType)
	if requestId != "" {
		logMessage += fmt.Sprintf(" for id `%s`", requestId)
	}

	log.WithFields(field.Error(err)).Warn(logMessage+": %v", err)
}
