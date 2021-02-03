package yamlcreator

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"gopkg.in/yaml.v2"
)

//YamlCreator implements method to create the yaml configuration file
type YamlCreator interface {
	CreateYamlFile(creator files.FileCreator, path string, name string) error
	AddConfig(name string, rawName string)
}

//YamlConfig defines the structure for the config file for each API
type YamlConfig struct {
	Config []map[string]string
	Detail map[string][]DetailConfig `yaml:",inline"`
}

//DetailConfig sets the default properties to be set replace in each json file
type DetailConfig struct {
	Name string `yaml:"name"`
}

//NewYamlConfig return a new yaml struct with Config and Detail as fields
func NewYamlConfig() *YamlConfig {
	yamlConfig := YamlConfig{}
	yamlConfig.Detail = make(map[string][]DetailConfig)
	return &yamlConfig
}

//AddConfig allows to add new configs to the yaml file
func (yc *YamlConfig) AddConfig(name string, rawName string) {

	config := DetailConfig{Name: rawName}
	mp := make(map[string]string)
	mp[name] = name + ".json"
	yc.Config = append(yc.Config, mp)
	yc.Detail[name] = append(yc.Detail[name], config)
}

//CreateYamlFile transforms the struct into a physical file on disk
func (yc *YamlConfig) CreateYamlFile(creator files.FileCreator, path string, name string) error {

	data, err := yaml.Marshal(yc)
	if err != nil {
		util.Log.Error("error parsing yaml file: %v", err)
		return err
	}
	_, err = creator.CreateFile(data, path, name, ".yaml")
	if err != nil {
		util.Log.Error("error creating yaml file %s", name)
		return err
	}
	return nil
}
