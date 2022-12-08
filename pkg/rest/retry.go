// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"net/http"
	"time"
)

type retrySetting struct {
	waitTime   time.Duration
	maxRetries int
}

type retrySettings struct {
	normal   retrySetting
	long     retrySetting
	veryLong retrySetting
}

var defaultRetrySettings = retrySettings{
	normal: retrySetting{
		waitTime:   5 * time.Second,
		maxRetries: 3,
	},
	long: retrySetting{
		waitTime:   10 * time.Second,
		maxRetries: 3,
	},
	veryLong: retrySetting{
		waitTime:   15 * time.Second,
		maxRetries: 5,
	},
}

// getWithRetry will retry a GET request for a given number of times, waiting a give duration between calls
// this method can be used for API calls we know to have occasional timing issues on GET - e.g. paginated queries that are impacted by replication lag, returning unequal amounts of objects/pages per node
func getWithRetry(client *http.Client, url string, apiToken string, settings retrySetting) (resp Response, err error) {
	resp, err = get(client, url, apiToken)

	if err == nil && success(resp) {
		return resp, nil
	}

	for i := 0; i < settings.maxRetries; i++ {
		log.Warn("Retrying failed GET request %s after error (HTTP %d): %w", url, resp.StatusCode, err)
		time.Sleep(settings.waitTime)
		resp, err = get(client, url, apiToken)
		if err == nil && success(resp) {
			return resp, err
		}
	}

	var retryErr error
	if err != nil {
		retryErr = fmt.Errorf("GET request %s failed after %d retries: %w", url, settings.maxRetries, err)
	} else {
		retryErr = fmt.Errorf("GET request %s failed after %d retries: (HTTP %d)!\n    Response was: %s", url, settings.maxRetries, resp.StatusCode, resp.Body)
	}
	return Response{}, retryErr
}

// getWithRetry will retry a sendingRequest(PUT or POST) for a given number of times, waiting a give duration between calls
func sendWithRetry(client *http.Client, restCall sendingRequest, objectName string, path string, body []byte, apiToken string, setting retrySetting) (resp Response, err error) {

	for i := 0; i < setting.maxRetries; i++ {
		log.Warn("\t\t\tDependency of config %s was not available. Waiting for %s before retry...", objectName, setting.waitTime)
		time.Sleep(setting.waitTime)
		resp, err = restCall(client, path, body, apiToken)
		if err == nil && success(resp) {
			return resp, err
		}
	}

	var retryErr error
	if err != nil {
		retryErr = fmt.Errorf("dependency of config %s was not available after %d retries: %w", objectName, setting.maxRetries, err)
	} else {
		retryErr = fmt.Errorf("dependency of config %s was not available after %d retries: (HTTP %d)!\n    Response was: %s", objectName, setting.maxRetries, resp.StatusCode, resp.Body)
	}
	return Response{}, retryErr
}
