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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"runtime"

	"github.com/google/uuid"
)

// CtxUserAgentString context key used for passing a custom user-agent string to send with HTTP requests
type CtxKeyUserAgent struct{}

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

	if customUserAgentString, ok := request.Context().Value(CtxKeyUserAgent{}).(string); ok && customUserAgentString != "" {
		request.Header.Set("User-Agent", customUserAgentString)
	} else {
		request.Header.Set("User-Agent", "Dynatrace Monitoring as Code/"+version.MonitoringAsCode+" "+(runtime.GOOS+" "+runtime.GOARCH))
	}

	var requestId string
	if trafficlogs.IsRequestLoggingActive() {
		requestId = uuid.NewString()
		err := trafficlogs.LogRequest(requestId, request)

		if err != nil {
			log.WithFields(field.Error(err)).Warn("error while writing request log for id `%s`: %v", requestId, err)
		}
	}

	rateLimitStrategy := createRateLimitStrategy()

	response, err := rateLimitStrategy.executeRequest(timeutils.NewTimelineProvider(), func() (Response, error) {
		resp, err := client.Do(request)
		if err != nil {
			log.Error("HTTP Request failed with Error: " + err.Error())
			return Response{}, err
		}
		defer func() {
			err = resp.Body.Close()
		}()
		body, err := io.ReadAll(resp.Body)

		if trafficlogs.IsResponseLoggingActive() {
			err := trafficlogs.LogResponse(requestId, resp, string(body))

			if err != nil {
				if requestId != "" {
					log.WithFields(field.Error(err)).Warn("error while writing response log for id `%s`: %v", requestId, err)
				} else {
					log.WithFields(field.Error(err)).Warn("error while writing response log: %v", requestId, err)
				}
			}
		}

		nextPageKey, totalCount, pageSize := getPaginationValues(body)

		returnResponse := Response{
			StatusCode:  resp.StatusCode,
			Body:        body,
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
