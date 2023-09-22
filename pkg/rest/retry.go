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
		WaitTime:   time.Second,
		MaxRetries: 15,
	},
	Long: RetrySetting{
		WaitTime:   time.Second,
		MaxRetries: 30,
	},
	VeryLong: RetrySetting{
		WaitTime:   time.Second,
		MaxRetries: 60,
	},
}

// SendWithRetry will retry to call sendWithBody for a given number of times, waiting a give duration between calls
func SendWithRetry(ctx context.Context, sendWithBody SendRequestWithBody, objectName string, path string, body []byte, setting RetrySetting) (resp Response, err error) {

	for i := 0; i < setting.MaxRetries; i++ {
		log.WithCtxFields(ctx).Warn("Failed to send HTTP request. Waiting for %s before retrying...", setting.WaitTime)
		time.Sleep(setting.WaitTime)
		resp, err = sendWithBody(ctx, path, body)
		if err == nil && resp.IsSuccess() {
			return resp, err
		}
	}

	if err != nil {
		return Response{}, fmt.Errorf("HTTP send request %s failed after %d retries: %w", path, setting.MaxRetries, err)
	}
	return Response{}, NewRespErr(fmt.Sprintf("HTTP send request %s failed after %d retries: (HTTP %d)", path, setting.MaxRetries, resp.StatusCode), resp)
}

// SendWithRetryWithInitialTry will try to call sendWithBody and if it didn't succeed call [SendWithRetry]
func SendWithRetryWithInitialTry(ctx context.Context, sendWithBody SendRequestWithBody, objectName string, path string, body []byte, setting RetrySetting) (resp Response, err error) {
	resp, err = sendWithBody(ctx, path, body)
	if err == nil && resp.IsSuccess() {
		return resp, err
	}

	return SendWithRetry(ctx, sendWithBody, objectName, path, body, setting)
}
