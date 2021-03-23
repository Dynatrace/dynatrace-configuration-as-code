package rest

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func isCertificateEntityUpToDate(client *http.Client, apiToken string, sslConfig api.SslCertificateConfig, filePath string,
	fullUrl string) bool {
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

func prepareCertificateUpload() {

}

type CertificateUpload struct {
	
}

type certificateInfo struct {
	subject    string
	issuer     string
	expiration time.Time
}

func getCertificateInfo(filePath string) (certificateInfo, error) {
	var certInfo certificateInfo
	certFile, err := os.Open(filePath)

	if util.CheckError(err, "Unable to open certificate file "+filePath) {
		return certInfo, err
	}

	defer certFile.Close()

	certBytes, err := ioutil.ReadAll(certFile)

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
