// +build unit

package yamlcreator

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"gotest.tools/assert"
)

func TestNewYamlConfig(t *testing.T) {
	config := NewYamlConfig()
	assert.Check(t, config.Detail != nil, "map not initialized")
}

func TestAddConfig(t *testing.T) {
	//test special name in config file
	config := NewYamlConfig()
	config.AddConfig("test", "test 1234")
	assert.Check(t, len(config.Detail["test"]) == 1)
	assert.Check(t, config.Detail["test"][0].Name == "test 1234")
}

func TestCreateYamlFile(t *testing.T) {
	// ctrl := gomock.NewController(t)
	config := NewYamlConfig()
	config.AddConfig("test", "test 1234")
	fileCreator := files.NewInMemoryFileCreator()
	err := config.CreateYamlFile(fileCreator, "", "test")
	assert.NilError(t, err)
}
