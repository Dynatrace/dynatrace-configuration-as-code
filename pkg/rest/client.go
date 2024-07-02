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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
)

type Client struct {
	client            *http.Client
	rateLimitStrategy RateLimitStrategy
	trafficLogger     *trafficlogs.FileBasedLogger
}

func NewRestClient(client *http.Client, trafficLogger *trafficlogs.FileBasedLogger, strategy RateLimitStrategy) *Client {
	return &Client{
		client:            client,
		rateLimitStrategy: strategy,
		trafficLogger:     trafficLogger,
	}
}
func (c Client) Client() *http.Client {
	return c.client
}

func (c Client) Get(ctx context.Context, url string) (Response, error) {
	req, err := c.request(ctx, http.MethodGet, url)

	if err != nil {
		return Response{}, err
	}

	return c.executeRequest(req)
}

func (c Client) GetWithRetry(ctx context.Context, url string, settings RetrySetting) (resp Response, err error) {
	resp, err = c.Get(ctx, url)

	if err == nil && resp.IsSuccess() {
		return resp, nil
	}

	for i := 0; i < settings.MaxRetries; i++ {
		if err != nil {
			log.WithCtxFields(ctx).WithFields(field.Error(err)).Warn("Retrying failed GET request %s with error: %v", url, err)
		} else {
			log.WithCtxFields(ctx).Warn("Retrying failed GET request %s (HTTP %d)", url, resp.StatusCode)
		}
		time.Sleep(settings.WaitTime)
		resp, err = c.Get(ctx, url)
		if err == nil && resp.IsSuccess() {
			return resp, err
		}
	}

	if err != nil {
		return resp, fmt.Errorf("GET request %s failed after %d retries: %w", url, settings.MaxRetries, err)
	}

	return resp, RespError{
		StatusCode: resp.StatusCode,
		Reason:     fmt.Sprintf("GET request %s failed after %d retries: (HTTP %d)!\n    Response was: %s", url, settings.MaxRetries, resp.StatusCode, resp.Body),
		Body:       string(resp.Body),
	}
}

func (c Client) Delete(ctx context.Context, url string) (Response, error) {
	req, err := c.request(ctx, http.MethodDelete, url)

	if err != nil {
		return Response{}, err
	}

	return c.executeRequest(req)

}

func (c Client) Post(ctx context.Context, url string, data []byte) (Response, error) {
	req, err := c.requestWithBody(ctx, http.MethodPost, url, bytes.NewBuffer(data))

	if err != nil {
		return Response{}, err
	}

	return c.executeRequest(req)
}

func (c Client) PostMultiPartFile(ctx context.Context, url string, data *bytes.Buffer, contentType string) (Response, error) {
	req, err := c.requestWithBody(ctx, http.MethodPost, url, data)

	if err != nil {
		return Response{}, err
	}

	req.Header.Set("Content-type", contentType)

	return c.executeRequest(req)
}

func (c Client) Put(ctx context.Context, url string, data []byte) (Response, error) {
	req, err := c.requestWithBody(ctx, http.MethodPut, url, bytes.NewBuffer(data))

	if err != nil {
		return Response{}, err
	}

	return c.executeRequest(req)
}

func (c Client) request(ctx context.Context, method string, url string) (*http.Request, error) {
	return c.requestWithBody(ctx, method, url, nil)
}

func (c Client) requestWithBody(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-type", "application/json")
	return req, nil
}

func (c Client) executeRequest(request *http.Request) (Response, error) {

	request.Header.Set("User-Agent", "Dynatrace-config-as-code-http-client")

	// extract request body for logging before executing the request drains it
	var reqBody string
	if c.trafficLogger != nil && request.Body != nil {
		b, err := io.ReadAll(request.Body)
		if err == nil {
			reqBody = string(b)
		} else {
			reqBody = "failed to extract body"
		}
	}

	response, err := c.rateLimitStrategy.ExecuteRequest(timeutils.NewTimelineProvider(), func() (Response, error) {
		resp, err := c.client.Do(request)
		if err != nil {
			if isConnectionResetErr(err) {
				return Response{}, fmt.Errorf("HTTP request failed: Unable to connect to host %q, connection closed unexpectedly: %w", request.Host, err)
			}
			return Response{}, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				if err != nil {
					// don't overwrite an actual error for a body close issue
					log.WithFields(field.Error(err)).Warn("Failed to close HTTP response body after previous error. Closing error: %s", err)
					return
				}

				err = fmt.Errorf("failed to close HTTP response body: %w", closeErr)
			}
		}()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return Response{}, fmt.Errorf("failed to parse response respBody: %w", err)
		}

		if c.trafficLogger != nil {
			err := c.trafficLogger.Log(request, reqBody, resp, string(respBody))
			if err != nil {
				log.WithFields(field.Error(err)).Warn("unable to log traffic: %v", err)
			}
		}

		nextPageKey, totalCount, pageSize := getPaginationValues(respBody)

		returnResponse := Response{
			StatusCode:  resp.StatusCode,
			Body:        respBody,
			Headers:     resp.Header,
			NextPageKey: nextPageKey,
			TotalCount:  totalCount,
			PageSize:    pageSize,
		}

		return returnResponse, err
	})

	if err != nil {
		return Response{}, err
	}
	return response, nil
}

func getPaginationValues(body []byte) (nextPageKey string, totalCount int, pageSize int) {
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return
	}

	if jsonResponse["nextPageKey"] != nil {
		nextPageKey = jsonResponse["nextPageKey"].(string)
	}

	if jsonResponse["totalCount"] != nil {
		totalCount = int(jsonResponse["totalCount"].(float64))
	}

	if jsonResponse["pageSize"] != nil {
		pageSize = int(jsonResponse["pageSize"].(float64))
	}

	return
}

func isConnectionResetErr(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && errors.Is(urlErr, io.EOF) {
		// there is no direct way to discern a connection reset error, but if it's an url.Error wrapping an io.EOF we can be relatively certain it is
		// unless net/http stops reporting this as io.EOF
		return true
	}
	return false
}

// SendRequestWithBody is a function doing a PUT or POST HTTP request
type SendRequestWithBody func(ctx context.Context, url string, data []byte) (Response, error)
