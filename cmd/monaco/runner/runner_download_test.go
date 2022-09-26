//go:build unit

package runner

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/download"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

func Test_NoArgsSupplied(t *testing.T) {
	commandMock := createDownloadCommandMock(t)

	cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "either '--environments' or '--url' has to be provided")
}

func Test_UrlWithoutArg(t *testing.T) {
	commandMock := createDownloadCommandMock(t)

	cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{"--url"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "--url")
}

func Test_UrlWithArgButMissingEnvAndToken(t *testing.T) {
	commandMock := createDownloadCommandMock(t)

	cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{"--url", "test"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "environment-name")
	assert.ErrorContains(t, err, "token-name")
}

func Test_UrlAndEnvButMissingToken(t *testing.T) {
	commandMock := createDownloadCommandMock(t)

	cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{"--url", "test", "--environment-name", "test"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "environment-name")
}

func Test_UrlAndEnvButMissingValue(t *testing.T) {
	commandMock := createDownloadCommandMock(t)

	cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{"--url", "test", "--environment-name"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "environment-name")
}

func Test_UrlAndTokenButMissingValue(t *testing.T) {
	commandMock := createDownloadCommandMock(t)

	cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
	cmd.SetArgs([]string{"--url", "test", "--token-name"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "token-name")
}

func createDownloadCommandMock(t *testing.T) *download.MockCommand {
	mockCtrl := gomock.NewController(t)
	commandMock := download.NewMockCommand(mockCtrl)
	return commandMock
}
