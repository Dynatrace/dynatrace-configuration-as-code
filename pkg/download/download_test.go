// +build unit

package download

import (
	"os"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/jsoncreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/yamlcreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

func TestGetConfigs(t *testing.T) {
	os.Setenv("token", "test")
	env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")
	envs := make(map[string]environment.Environment)
	envs["e1"] = env
	status := GetConfigs(envs, "", "")
	assert.Equal(t, status, 0)
}
func TestCreateConfigsFromAPI(t *testing.T) {
	apiMock := api.CreateAPIMockFactory(t)
	fcreator := files.CreateFileCreatorMockFactory(t)
	client := rest.CreateDynatraceClientMockFactory(t)
	jcreator := jsoncreator.CreateJSONCreatorMock(t)
	ycreator := yamlcreator.CreateYamlCreatorMock(t)

	list := []api.Value{{Id: "d", Name: "namevalue"}}

	client.EXPECT().
		List(gomock.Any()).Return(list, nil)

	apiMock.EXPECT().
		GetId().Return("synthetic-monitor").AnyTimes()

	jcreator.EXPECT().
		CreateJSONConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("demo.json", false, nil)

	ycreator.EXPECT().
		CreateYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	ycreator.EXPECT().AddConfig(gomock.Any(), gomock.Any())
	fcreator.EXPECT().
		CreateFolder("/synthetic-monitor").
		Return("synthetic-monitor", nil)

	err := createConfigsFromAPI(apiMock, "123", fcreator, "/", client, jcreator, ycreator)
	assert.NilError(t, err)
}

func TestTransFormSpecialCasesAPIPath(t *testing.T) {
	apiurl := transFormSpecialCasesAPIPath("synthetic-location", "/test/api")
	assert.Equal(t, apiurl, "/test/api?type=PRIVATE")
	apiurl = transFormSpecialCasesAPIPath("other-api", "/test/api")
	assert.Equal(t, apiurl, "/test/api")
}

func TestDownloadConfigFromEnvironment(t *testing.T) {
	os.Setenv("token", "test")
	env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")
	err := downloadConfigFromEnvironment(env, "", nil)
	assert.NilError(t, err)
}
func TestGetAPIList(t *testing.T) {
	//multiple options
	list, err := getAPIList("synthetic-location,   extension, alerting-profile")
	assert.Check(t, len(err) == 0)
	assert.Check(t, list["synthetic-location"].GetId() == "synthetic-location")
	assert.Check(t, list["dashboard"] == nil)
	list, err = getAPIList("synthetic-location,extension,dashboard")
	assert.Check(t, len(err) == 0)
	//single option
	list, err = getAPIList("synthetic-location")
	assert.Check(t, len(err) == 0)
	//no option
	list, err = getAPIList("")
	assert.Check(t, len(err) == 0)
	list, err = getAPIList(" ")
	assert.Check(t, len(err) == 0)
	//not a real API
	list, err = getAPIList("synthetic-location-test,   extension-test, alerting-profile")
	assert.Check(t, err[0].Error() == "Value synthetic-location-test is not a valid API name ")
	assert.Check(t, err[1].Error() == "Value extension-test is not a valid API name ")
}
