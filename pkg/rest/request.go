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
	"io"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/timeutils"

	"github.com/google/uuid"
)

func Get(client *http.Client, url string) (Response, error) {
	req, err := request(http.MethodGet, url)

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)
}

// the name delete() would collide with the built-in function
func DeleteConfig(client *http.Client, url string, id string) error {
	fullPath := url + "/" + id
	req, err := request(http.MethodDelete, fullPath)

	if err != nil {
		return err
	}

	resp, err := executeRequest(client, req)

	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		log.Debug("No config with id '%s' found to delete (HTTP 404 response)", id)
		return nil
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("failed call to DELETE %s (HTTP %d)!\n Response was:\n %s", fullPath, resp.StatusCode, string(resp.Body))
	}

	return nil
}

func Post(client *http.Client, url string, data []byte) (Response, error) {
	req, err := requestWithBody(http.MethodPost, url, bytes.NewBuffer(data))

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)
}

func PostMultiPartFile(client *http.Client, url string, data *bytes.Buffer, contentType string) (Response, error) {
	req, err := requestWithBody(http.MethodPost, url, data)

	if err != nil {
		return Response{}, err
	}

	req.Header.Set("Content-type", contentType)

	return executeRequest(client, req)
}

func Put(client *http.Client, url string, data []byte) (Response, error) {
	req, err := requestWithBody(http.MethodPut, url, bytes.NewBuffer(data))

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req)
}

// function type of Put and Post requests
type SendingRequest func(client *http.Client, url string, data []byte) (Response, error)

func request(method string, url string) (*http.Request, error) {
	return requestWithBody(method, url, nil)
}

func requestWithBody(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-type", "application/json")
	return req, nil
}

func executeRequest(client *http.Client, request *http.Request) (Response, error) {
	var requestId string
	if log.IsRequestLoggingActive() {
		requestId = uuid.NewString()
		err := log.LogRequest(requestId, request)

		if err != nil {
			log.Warn("error while writing request log for id `%s`: %v", requestId, err)
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

		if log.IsResponseLoggingActive() {
			err := log.LogResponse(requestId, resp, string(body))

			if err != nil {
				if requestId != "" {
					log.Warn("error while writing response log for id `%s`: %v", requestId, err)
				} else {
					log.Warn("error while writing response log: %v", requestId, err)
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
