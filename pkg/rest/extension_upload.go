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
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

func uploadExtension(client *http.Client, apiPath string, extensionName string, extensionJson string, apiToken string) (api.DynatraceEntity, error) {
	buffer, contentType, err := writeMultiPartForm(extensionName, extensionJson)
	if err != nil {
		return api.DynatraceEntity{
			Name: extensionName,
		}, err
	}

	resp := postMultiPartFile(client, apiPath, buffer, contentType, apiToken)

	if resp.StatusCode != http.StatusCreated {
		util.Log.Error("\t\t\tUpload of %s failed with status %d!\n\t\t\t\t\tError-message: %s\n", extensionName, resp.StatusCode, string(resp.Body))
	} else {
		util.Log.Debug("\t\t\tExtension upload successful for %s", extensionName)

		// As other configs depend on metrics created by extensions, and metric creation seems to happen with delay...
		time.Sleep(1 * time.Second)
	}

	return api.DynatraceEntity{
		Name: extensionName,
	}, nil

}

func writeMultiPartForm(extensionName string, extensionJson string) (buffer *bytes.Buffer, contentType string, err error) {
	buffer = new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buffer)
	formFileWriter, _ := multipartWriter.CreateFormFile("file", extensionName+".zip")

	zipBuffer, err := writeInMemoryZip("custom/plugin.json", extensionJson)
	if err != nil {
		return buffer, "", err
	}

	_, err = formFileWriter.Write(zipBuffer.Bytes())
	if err != nil {
		return buffer, "", err
	}

	err = multipartWriter.Close()
	if err != nil {
		return buffer, "", err
	}

	contentType = multipartWriter.FormDataContentType()

	return buffer, contentType, nil
}

func writeInMemoryZip(fileName string, fileContent string) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buffer)
	zipFile, err := zipWriter.Create(fileName)
	if util.CheckError(err, "Failed to create .zip file") {
		return buffer, err
	}
	_, err = zipFile.Write([]byte(fileContent))
	if err != nil {
		return buffer, err
	}
	err = zipWriter.Close()
	if err != nil {
		return buffer, err
	}

	return buffer, nil
}
