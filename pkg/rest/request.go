/**
 * @license
 * Copyright 2020 Dynatrace LLC
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
	"io"
	"io/ioutil"
	"net/http"
	"runtime"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
	"github.com/google/uuid"
)

type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string][]string
}

// function type of put and post requests
type sendingRequest func(client *http.Client, url string, data []byte, apiToken string) (Response, error)

func get(client *http.Client, url string, apiToken string) (Response, error) {
	req, err := request(http.MethodGet, url, apiToken)

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req), nil
}

// the name delete() would collide with the built-in function
func deleteConfig(client *http.Client, url string, apiToken string, id string) error {
	req, err := request(http.MethodDelete, url+"/"+id, apiToken)

	if err != nil {
		return err
	}

	executeRequest(client, req)

	return nil
}

func post(client *http.Client, url string, data []byte, apiToken string) (Response, error) {
	req, err := requestWithBody(http.MethodPost, url, bytes.NewBuffer(data), apiToken)

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req), nil
}

func postMultiPartFile(client *http.Client, url string, data *bytes.Buffer, contentType string, apiToken string) (Response, error) {
	req, err := requestWithBody(http.MethodPost, url, data, apiToken)

	if err != nil {
		return Response{}, err
	}

	req.Header.Set("Content-type", contentType)

	return executeRequest(client, req), nil
}

func put(client *http.Client, url string, data []byte, apiToken string) (Response, error) {
	req, err := requestWithBody(http.MethodPut, url, bytes.NewBuffer(data), apiToken)

	if err != nil {
		return Response{}, err
	}

	return executeRequest(client, req), nil
}

func request(method string, url string, apiToken string) (*http.Request, error) {
	return requestWithBody(method, url, nil, apiToken)
}

func requestWithBody(method string, url string, body io.Reader, apiToken string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Api-Token "+apiToken)
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("User-Agent", "Dynatrace Monitoring as Code/"+version.MonitoringAsCode+" "+(runtime.GOOS+" "+runtime.GOARCH))
	return req, nil
}

func executeRequest(client *http.Client, request *http.Request) Response {
	var requestId string
	if util.IsRequestLoggingActive() {
		requestId = uuid.NewString()
		err := util.LogRequest(requestId, request)

		if err != nil {
			util.Log.Warn("error while writing request log for id `%s`: %v", requestId, err)
		}
	}

	rateLimitStrategy := createRateLimitStrategy()

	response, err := rateLimitStrategy.executeRequest(util.NewTimelineProvider(), func() (Response, error) {
		resp, err := client.Do(request)
		if err != nil {
			util.Log.Error("HTTP Request failed with Error: " + err.Error())
			return Response{}, err
		}
		defer func() {
			err = resp.Body.Close()
		}()
		body, err := ioutil.ReadAll(resp.Body)

		if util.IsResponseLoggingActive() {
			err := util.LogResponse(requestId, resp)

			if err != nil {
				if requestId != "" {
					util.Log.Warn("error while writing response log for id `%s`: %v", requestId, err)
				} else {
					util.Log.Warn("error while writing response log: %v", requestId, err)
				}
			}
		}

		return Response{
			StatusCode: resp.StatusCode,
			Body:       body,
			Headers:    resp.Header,
		}, err
	})

	if err != nil {
		// TODO properly handle error
		return Response{}
	}
	return response
}
