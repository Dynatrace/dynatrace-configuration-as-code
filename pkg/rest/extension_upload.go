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
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

type extensionStatus int

const (
	extensionValidationError extensionStatus = iota
	extensionUpToDate
	extensionConfigOutdated
	extensionNeedsUpdate
)

func uploadExtension(client *http.Client, apiPath string, extensionName string, payload []byte, apiToken string) (api.DynatraceEntity, error) {

	status, err := validateIfExtensionShouldBeUploaded(client, apiPath, extensionName, payload, apiToken)
	if err != nil {
		return api.DynatraceEntity{}, err
	}

	if status == extensionUpToDate {
		return api.DynatraceEntity{
			Name: extensionName,
		}, nil
	}

	buffer, contentType, err := writeMultiPartForm(extensionName, payload)
	if err != nil {
		return api.DynatraceEntity{
			Name: extensionName,
		}, err
	}

	resp, err := postMultiPartFile(client, apiPath, buffer, contentType, apiToken)

	if err != nil {
		return api.DynatraceEntity{}, err
	}

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

func validateIfExtensionShouldBeUploaded(client *http.Client, apiPath string, extensionName string, payload []byte, apiToken string) (status extensionStatus, err error) {
	response, err := get(client, apiPath+"/"+extensionName, apiToken)
	if err != nil {
		return extensionValidationError, err
	}
	if response.StatusCode == http.StatusNotFound {
		return extensionNeedsUpdate, nil
	}
	var extProperties struct{ Version *string }
	if err := json.Unmarshal(response.Body, &extProperties); err != nil {
		return extensionValidationError, err
	}

	if extProperties.Version == nil {
		return extensionValidationError, fmt.Errorf("API failed to return a version for extension (%s)", extensionName)
	}
	curVersion := *extProperties.Version

	var extension struct{ Version *string }
	if err := json.Unmarshal(payload, &extension); err != nil {
		return extensionValidationError, err
	}

	if extension.Version == nil {
		return extensionValidationError, fmt.Errorf("extension configuration (%s) does not define a version", extensionName)
	}
	newVersion := *extension.Version

	if curVersion > newVersion {
		err := fmt.Errorf("already deployed version (%s) of extension (%s) is newer than local (%s)", extensionName, curVersion, newVersion)
		return extensionConfigOutdated, err
	}

	if curVersion == newVersion {
		util.Log.Info("Extension (%s) already deployed in version (%s), skipping.", extensionName, newVersion)
		return extensionUpToDate, nil
	}

	return extensionNeedsUpdate, nil
}

func writeMultiPartForm(extensionName string, extensionJson []byte) (buffer *bytes.Buffer, contentType string, err error) {
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

func writeInMemoryZip(fileName string, fileContent []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buffer)
	zipFile, err := zipWriter.Create(fileName)
	if util.CheckError(err, "Failed to create .zip file") {
		return buffer, err
	}
	_, err = zipFile.Write(fileContent)
	if err != nil {
		return buffer, err
	}
	err = zipWriter.Close()
	if err != nil {
		return buffer, err
	}

	return buffer, nil
}
