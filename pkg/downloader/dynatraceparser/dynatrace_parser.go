// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dynatraceparser

import (
	"encoding/json"
	"net/url"
	"path/filepath"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

//go:generate mockgen -source=dynatrace_parser.go -destination=dynatrace_parser_mock.go -package=dynatraceparser DynatraceParser

//DynatraceParser Core class that deals with Dynatrace API payload transformation based on config type
type DynatraceParser interface {
	GetConfig(client rest.DynatraceClient, api api.Api, value api.Value,
		path string) (file string, dynatraceId string, fullpath string, cleanName string, filter bool, err error)
}

//DynatraceParserImp object
type DynatraceParserImp struct{}

//NewDynatraceParser creates a new instance of the DynatraceParser
func NewDynatraceParser() *DynatraceParserImp {
	result := DynatraceParserImp{}
	return &result
}

//GetConfig returns a config file as a string from Dynatrace API
func (d *DynatraceParserImp) GetConfig(client rest.DynatraceClient, api api.Api, value api.Value,
	path string) (file string, dynatraceId string, fullPath string, cleanName string, filter bool, err error) {
	data, filter, err := getDetailFromAPI(client, api, value.Id)
	if err != nil {
		log.Error("error getting detail %s from API %s", value.Name, api.GetId())
		return "", "", "", "", false, err
	}
	if filter {
		log.Debug("configuration filtered %s from API %s", value.Name, api.GetId())
		return "", "", "", "", true, nil
	}
	stringfile, dynatraceId, cleanName, err := processConfiguration(data, value.Id, value.Name, api)
	if err != nil {
		log.Error("error processing jsonfile %s", api.GetId())
		return "", "", "", "", false, err
	}
	fullPath = filepath.Join(path, cleanName+".json")
	return stringfile, dynatraceId, fullPath, cleanName, false, nil
}

func getDetailFromAPI(client rest.DynatraceClient, api api.Api, name string) (dat map[string]interface{}, filter bool, err error) {

	name = url.QueryEscape(name)
	resp, err := client.ReadById(api, name)
	if err != nil {
		log.Error("error getting detail for API %s", api.GetId(), name)
		return nil, false, err
	}
	err = json.Unmarshal(resp, &dat)
	if err != nil {
		log.Error("error transforming %s from json to object", name)
		return nil, false, err
	}
	filter = NotSupportedConfiguration(api.GetId(), dat)
	if filter {
		log.Debug("Configuration %s from api %s has been filtered out", name, api.GetId())
		return nil, true, err
	}
	return dat, false, nil
}

//processConfiguration removes and replaces properties for each json config to make them compatible with monaco standard
func processConfiguration(dat map[string]interface{}, id string, name string,
	api api.Api) (string, string, string, error) {

	name, err := GetNameForConfig(name, dat, api)
	if err != nil {
		return "", "", "", err
	}
	dynatraceId, dat := ReplaceKeyProperties(dat)
	cleanName := util.SanitizeName(name)
	jsonfile, err := json.MarshalIndent(dat, "", " ")

	if err != nil {
		log.Error("error creating json file  %s", id)
		return "", "", "", err
	}
	stringfile := string(jsonfile)
	return stringfile, dynatraceId, cleanName, nil
}
