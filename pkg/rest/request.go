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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"io"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"

	"github.com/google/uuid"
	"golang.org/x/net/context"
)

func Get(ctx context.Context, client *http.Client, url string) (Response, error) {
	req, err := request(ctx, http.MethodGet, url)

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)
}

func Delete(ctx context.Context, client *http.Client, url string) (Response, error) {
	req, err := request(ctx, http.MethodDelete, url)
	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)

}

func Post(ctx context.Context, client *http.Client, url string, data []byte) (Response, error) {
	req, err := requestWithBody(ctx, http.MethodPost, url, bytes.NewBuffer(data))

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)
}

func PostMultiPartFile(ctx context.Context, client *http.Client, url string, data *bytes.Buffer, contentType string) (Response, error) {
	req, err := requestWithBody(ctx, http.MethodPost, url, data)

	if err != nil {
		return Response{}, err
	}

	req.Header.Set("Content-type", contentType)

	return executeRequest(client, req)
}

func Put(ctx context.Context, client *http.Client, url string, data []byte) (Response, error) {
	req, err := requestWithBody(ctx, http.MethodPut, url, bytes.NewBuffer(data))

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)
}

// SendRequestWithBody is a function doing a PUT or POST HTTP request
type SendRequestWithBody func(ctx context.Context, client *http.Client, url string, data []byte) (Response, error)

func request(ctx context.Context, method string, url string) (*http.Request, error) {
	return requestWithBody(ctx, method, url, nil)
}

func requestWithBody(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)

	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-type", "application/json")
	return req, nil
}

func executeRequest(client *http.Client, request *http.Request) (Response, error) {

	request.Header.Set("User-Agent", "Dynatrace-config-as-code-http-client")

	// extract request body for logging before executing the request drains it
	var reqBody string
	if trafficlogs.IsRequestLoggingActive() && request.Body != nil {
		b, err := io.ReadAll(request.Body)
		if err == nil {
			reqBody = string(b)
		} else {
			reqBody = "failed to extract body"
		}
	}

	rateLimitStrategy := createRateLimitStrategy()

	response, err := rateLimitStrategy.executeRequest(timeutils.NewTimelineProvider(), func() (Response, error) {
		resp, err := client.Do(request)
		if err != nil {
			return Response{}, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer func() {
			closeErr := resp.Body.Close()
			if err == nil {
				err = fmt.Errorf("failed to close HTTP response body: %w", closeErr)
			} else {
				// don't overwrite an actual error for a body close issue
				log.Warn("Failed to close HTTP response body after previous error. Closing error: %w", err)
			}
		}()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return Response{}, fmt.Errorf("failed to parse response respBody: %w", err)
		}

		writeTrafficLog(request, reqBody, resp, string(respBody))

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

func writeTrafficLog(req *http.Request, reqBody string, resp *http.Response, respBody string) {
	var requestId string
	if trafficlogs.IsRequestLoggingActive() {
		requestId = uuid.NewString()
		err := trafficlogs.LogRequest(requestId, req, reqBody)

		if err != nil {
			log.WithFields(field.Error(err)).Warn("error while writing request log for id `%s`: %v", requestId, err)
		}
	}
	if trafficlogs.IsResponseLoggingActive() {
		err := trafficlogs.LogResponse(requestId, resp, respBody)

		if err != nil {
			if requestId != "" {
				log.WithFields(field.Error(err)).Warn("error while writing response log for id `%s`: %v", requestId, err)
			} else {
				log.WithFields(field.Error(err)).Warn("error while writing response log: %v", requestId, err)
			}
		}
	}
}
