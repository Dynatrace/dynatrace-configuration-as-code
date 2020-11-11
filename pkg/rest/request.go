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

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
)

type Response struct {
	StatusCode int
	Body       []byte
}

func Get(url string, apiToken string) Response {
	req := request(http.MethodGet, url, apiToken)
	return executeRequest(req)
}

func Delete(url string, apiToken string, id string) {
	req := request(http.MethodDelete, url+"/"+id, apiToken)
	executeRequest(req)
}

func Post(url string, data string, apiToken string) Response {
	req := requestWithBody(http.MethodPost, url, bytes.NewBuffer([]byte(data)), apiToken)
	return executeRequest(req)
}

func PostMultiPartFile(url string, data *bytes.Buffer, contentType string, apiToken string) Response {
	req := requestWithBody(http.MethodPost, url, data, apiToken)
	req.Header.Set("Content-type", contentType)
	return executeRequest(req)
}

func Put(url string, data string, apiToken string) Response {
	req := requestWithBody(http.MethodPut, url, bytes.NewBuffer([]byte(data)), apiToken)
	return executeRequest(req)
}

func request(method string, url string, apiToken string) *http.Request {
	return requestWithBody(method, url, nil, apiToken)
}

func requestWithBody(method string, url string, body io.Reader, apiToken string) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Api-Token "+apiToken)
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("User-Agent", "Dynatrace Monitoring as Code/"+version.MonitoringAsCode+" "+(runtime.GOOS+" "+runtime.GOARCH))
	return req
}

func executeRequest(request *http.Request) Response {
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		println("HTTP Request failed with Error: " + err.Error())
		return Response{}
	}
	defer func() {
		err = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	return Response{resp.StatusCode, body}
}
