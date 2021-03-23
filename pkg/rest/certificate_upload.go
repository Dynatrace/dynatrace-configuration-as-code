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
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func isCertificateEntityUpToDate(client *http.Client, apiToken string, fullUrl string, sslConfig api.SslCertificateConfig,
	filePath string) bool {
	var configsDir = filepath.Dir(filePath)
	certLocation := filepath.Join(configsDir, sslConfig.CertificateFile)
	certInfo, err := getCertificateInfo(certLocation)
	if err != nil {
		return true
	}
	exInfo, err := fetchExistingCertificateInfo(client, apiToken, fullUrl, sslConfig.NodeId, sslConfig.CertificateType)
	if err != nil {
		return true
	}
	return certInfo.issuer == exInfo.issuer &&
		certInfo.subject == exInfo.subject &&
		certInfo.expiration == exInfo.expiration
}

func uploadCertificate(client *http.Client, apiToken string, fullUrl string, sslConfig api.SslCertificateConfig,
	filePath string) (Response, error) {
	var resp Response
	upload, err := prepareCertificateUpload(sslConfig, filePath)
	if util.CheckError(err, "Failed to read certificate files") {
		return resp, err
	}
	fullUploadUrl := strings.TrimSuffix(fullUrl, "/") + "/store/" + sslConfig.CertificateType +
		"/" + strconv.Itoa(sslConfig.NodeId)
	jsonBody, err := json.Marshal(upload)
	if util.CheckError(err, "Failed to serialize certificate upload request") {
		return resp, err
	}

	util.Log.Info("Uploading certificate for " + sslConfig.CertificateType +
		" and node " + strconv.Itoa(sslConfig.NodeId))
	resp, err = post(client, fullUploadUrl, jsonBody, apiToken)
	if util.CheckError(err, "Failed to upload SSL certificate to managed cluster, invalid server response") {
		return resp, err
	}

	return resp, nil
}

func prepareCertificateUpload(sslConfig api.SslCertificateConfig, filePath string) (api.SslCertificateUpload, error) {
	var sslUpload api.SslCertificateUpload
	var configsDir = filepath.Dir(filePath)
	cert, err := readCertificateFile(filepath.Join(configsDir, sslConfig.CertificateFile))
	if util.CheckError(err, "Failed to read certificate file "+sslConfig.CertificateFile) {
		return sslUpload, err
	}
	chain, err := readCertificateFile(filepath.Join(configsDir, sslConfig.CertificateChainFile))
	if util.CheckError(err, "Failed to read certificate chain file "+sslConfig.CertificateChainFile) {
		return sslUpload, err
	}
	privateKey, err := readCertificateFile(filepath.Join(configsDir, sslConfig.PrivateKeyFile))
	if util.CheckError(err, "Failed to read private key "+sslConfig.PrivateKeyFile) {
		return sslUpload, err
	}
	sslUpload = api.SslCertificateUpload{
		PrivateKeyEncoded:           privateKey,
		PublicKeyCertificateEncoded: cert,
		CertificateChainEncoded:     chain,
	}
	return sslUpload, nil
}

type certificateInfo struct {
	subject    string
	issuer     string
	expiration time.Time
}

func getCertificateInfo(filePath string) (certificateInfo, error) {
	var certInfo certificateInfo
	certBytes, err := ioutil.ReadFile(filePath)
	if util.CheckError(err, "Unable to open certificate file "+filePath) {
		return certInfo, err
	}
	if len(certBytes) == 0 {
		util.Log.Error("Certificate file " + filePath + " is empty")
		return certInfo, nil
	}

	decodedBytes, _ := pem.Decode(certBytes)
	cert, err := x509.ParseCertificate(decodedBytes.Bytes)
	if util.CheckError(err, "Unable to parse certificate file "+filePath) {
		return certInfo, err
	}
	if cert == nil {
		util.Log.Error("Certificate is null")
		return certInfo, nil
	}

	certInfo = certificateInfo{
		cert.Subject.CommonName,
		cert.Issuer.CommonName + ", " + cert.Issuer.Organization[0],
		cert.NotAfter,
	}

	return certInfo, nil
}

func fetchExistingCertificateInfo(client *http.Client, apiToken string, fullUrl string, nodeId int, certificateType string) (certificateInfo, error) {
	var certInfo certificateInfo
	var nodeCertificateInfo api.SslCertificateNodeConfig
	certificateInfoUrl := strings.TrimSuffix(fullUrl, "/") + "/" + certificateType + "/" + strconv.Itoa(nodeId)
	resp, err := get(client, certificateInfoUrl, apiToken)

	if util.CheckError(err, "Unable to fetch existing certificate info from cluster.") {
		return certInfo, err
	}

	err = json.Unmarshal(resp.Body, &nodeCertificateInfo)
	if util.CheckError(err, "Unable to unmarshal certificate .") {
		return certInfo, err
	}

	certInfo = certificateInfo{
		nodeCertificateInfo.Subject,
		nodeCertificateInfo.Issuer,
		time.Time(nodeCertificateInfo.ExpirationDate),
	}

	return certInfo, nil
}

func readCertificateFile(filePath string) (string, error) {
	var certificate string
	certBytes, err := ioutil.ReadFile(filePath)
	if util.CheckError(err, "Unable to open certificate file "+filePath) {
		return certificate, err
	}
	certificate = string(certBytes)
	return certificate, nil
}
