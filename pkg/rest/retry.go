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

package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

type RetrySetting struct {
	WaitTime   time.Duration
	MaxRetries int
}

type RetrySettings struct {
	Normal   RetrySetting
	Long     RetrySetting
	VeryLong RetrySetting
}

var DefaultRetrySettings = RetrySettings{
	Normal: RetrySetting{
		WaitTime:   5 * time.Second,
		MaxRetries: 3,
	},
	Long: RetrySetting{
		WaitTime:   5 * time.Second,
		MaxRetries: 6,
	},
	VeryLong: RetrySetting{
		WaitTime:   15 * time.Second,
		MaxRetries: 5,
	},
}

// GetWithRetry will retry a GET request for a given number of times, waiting a give duration between calls
// this method can be used for API calls we know to have occasional timing issues on GET - e.g. paginated queries that are impacted by replication lag, returning unequal amounts of objects/pages per node
func GetWithRetry(ctx context.Context, client *http.Client, url string, settings RetrySetting) (resp Response, err error) {
	resp, err = Get(ctx, client, url)

	if err == nil && resp.IsSuccess() {
		return resp, nil
	}

	for i := 0; i < settings.MaxRetries; i++ {
		log.WithCtxFields(ctx).Warn("Retrying failed GET request %s with error (HTTP %d)", url, resp.StatusCode)
		time.Sleep(settings.WaitTime)
		resp, err = Get(ctx, client, url)
		if err == nil && resp.IsSuccess() {
			return resp, err
		}
	}

	if err != nil {
		return resp, fmt.Errorf("GET request %s failed after %d retries: %w", url, settings.MaxRetries, err)
	}

	return resp, RespError{
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("GET request %s failed after %d retries: (HTTP %d)!\n    Response was: %s", url, settings.MaxRetries, resp.StatusCode, resp.Body),
		Body:       string(resp.Body),
	}
}

// SendWithRetry will retry to call sendWithBody for a given number of times, waiting a give duration between calls
func SendWithRetry(ctx context.Context, client *http.Client, sendWithBody SendRequestWithBody, objectName string, path string, body []byte, setting RetrySetting) (resp Response, err error) {

	for i := 0; i < setting.MaxRetries; i++ {
		log.WithCtxFields(ctx).Warn("Failed to send HTTP request. Waiting for %s before retrying...", setting.WaitTime)
		time.Sleep(setting.WaitTime)
		resp, err = sendWithBody(ctx, client, path, body)
		if err == nil && resp.IsSuccess() {
			return resp, err
		}
	}

	if err != nil {
		return Response{}, fmt.Errorf("HTTP send request %s failed after %d retries: %w", path, setting.MaxRetries, err)
	}
	return Response{}, NewRespErr(fmt.Sprintf("HTTP send request %s failed after %d retries: (HTTP %d)!\n    Response was: %s", path, setting.MaxRetries, resp.StatusCode, string(resp.Body)), resp)
}

// SendWithRetryWithInitialTry will try to call sendWithBody and if it didn't succeed call [SendWithRetry]
func SendWithRetryWithInitialTry(ctx context.Context, client *http.Client, sendWithBody SendRequestWithBody, objectName string, path string, body []byte, setting RetrySetting) (resp Response, err error) {
	resp, err = sendWithBody(ctx, client, path, body)
	if err == nil && resp.IsSuccess() {
		return resp, err
	}

	return SendWithRetry(ctx, client, sendWithBody, objectName, path, body, setting)
}
