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

package dtclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
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

// SendRequestWithBody is a function doing a PUT or POST HTTP request
type SendRequestWithBody func(ctx context.Context, endpoint string, body io.Reader, options corerest.RequestOptions) (*http.Response, error)

// SendWithRetry will retry to call sendWithBody for a given number of times, waiting a give duration between calls
func SendWithRetry(ctx context.Context, sendWithBody SendRequestWithBody, endpoint string, requestOptions corerest.RequestOptions, body []byte, setting RetrySetting) (*coreapi.Response, error) {
	var err error
	var resp *coreapi.Response

	for i := 0; i < setting.MaxRetries; i++ {
		log.WithCtxFields(ctx).Warn("Failed to send HTTP request. Waiting for %s before retrying. (%d of %d).", setting.WaitTime, i, setting.MaxRetries)
		time.Sleep(setting.WaitTime)
		resp, err = coreapi.AsResponseOrError(sendWithBody(ctx, endpoint, bytes.NewReader(body), requestOptions))
		if err == nil {
			return resp, nil
		}

		apierror := coreapi.APIError{}
		if !errors.As(err, &apierror) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("HTTP send request %s failed after %d retries: %w", endpoint, setting.MaxRetries, err)
}

// SendWithRetryWithInitialTry will try to call sendWithBody and if it didn't succeed call [SendWithRetry]
func SendWithRetryWithInitialTry(ctx context.Context, sendWithBody SendRequestWithBody, endpoint string, requestOptions corerest.RequestOptions, body []byte, setting RetrySetting) (*coreapi.Response, error) {
	resp, err := coreapi.AsResponseOrError(sendWithBody(ctx, endpoint, bytes.NewReader(body), requestOptions))
	if err == nil {
		return resp, nil
	}

	apiError := coreapi.APIError{}
	if !errors.As(err, &apiError) || !corerest.ShouldRetry(apiError.StatusCode) {
		return nil, err
	}

	return SendWithRetry(ctx, sendWithBody, endpoint, requestOptions, body, setting)
}

func GetWithRetry(ctx context.Context, c corerest.Client, endpoint string, requestOptions corerest.RequestOptions, settings RetrySetting) (resp *coreapi.Response, err error) {
	resp, err = coreapi.AsResponseOrError(c.GET(ctx, endpoint, requestOptions))
	if err == nil {
		return resp, nil
	}

	apiError := coreapi.APIError{}
	if !errors.As(err, &apiError) || !corerest.ShouldRetry(apiError.StatusCode) {
		return nil, err
	}

	url := c.BaseURL().JoinPath(endpoint).String()
	for i := 0; i < settings.MaxRetries; i++ {
		log.WithCtxFields(ctx).Warn("Retrying failed GET request %s (HTTP %d)", url, apiError.StatusCode)
		time.Sleep(settings.WaitTime)

		resp, err = coreapi.AsResponseOrError(c.GET(ctx, endpoint, requestOptions))
		if err == nil {
			return resp, nil
		}

		if !errors.As(err, &apiError) {
			return nil, err
		}
	}

	return resp, fmt.Errorf("GET request %s failed after %d retries: %w", url, settings.MaxRetries, err)
}
