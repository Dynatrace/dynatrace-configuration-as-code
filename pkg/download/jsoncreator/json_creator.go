package jsoncreator

import (
	"encoding/json"
	"net/url"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
)

//JSONCreator interface allows to mock the methods for unit testing
type JSONCreator interface {
	CreateJSONConfig(client rest.DynatraceClient, api api.Api, value api.Value, creator files.FileCreator,
		path string) (name string, filter bool, err error)
}

//JSONCreatorImp object
type JsonCreatorImp struct{}

//NewJSONCreator creates a new instance of the jsonCreator
func NewJSONCreator() *JsonCreatorImp {
	result := JsonCreatorImp{}
	return &result
}

//CreateJSONConfig creates a json file using the specified path and API data
func (d *JsonCreatorImp) CreateJSONConfig(client rest.DynatraceClient, api api.Api, value api.Value, creator files.FileCreator,
	path string) (name string, filter bool, err error) {
	data, filter, err := getDetailFromAPI(client, api, value.Id)
	if err != nil {
		util.Log.Error("error getting detail %s from API", api.GetId())
		return "", false, err
	}
	if filter == true {
		return "", true, nil
	}
	jsonfile, err := processJSONFile(data, value.Id)

	name, err = creator.CreateFile(jsonfile, path, value.Name, ".json")
	if err != nil {
		util.Log.Error("error writing detail %s", api.GetId())
		return "", false, err
	}
	return name, false, nil
}

func getDetailFromAPI(client rest.DynatraceClient, api api.Api, name string) (dat map[string]interface{}, filter bool, err error) {

	name = url.QueryEscape(name)
	resp, err := client.ReadById(api, name)
	if err != nil {
		util.Log.Error("error getting detail for API %s", api.GetId(), name)
		return nil, false, err
	}
	err = json.Unmarshal(resp, &dat)
	if err != nil {
		util.Log.Error("error transforming %s from json to object", name)
		return nil, false, err
	}
	filter = isDefaultEntity(api.GetId(), dat)
	if filter == true {
		util.Log.Debug("Non-user-created default Object has been filtered out", name)
		return nil, true, err
	}
	return dat, false, nil
}

//processJSONFile removes and replaces properties for each json config to make them compatible with monaco standard
func processJSONFile(dat map[string]interface{}, id string) ([]byte, error) {

	//removes id field
	delete(dat, "id")
	//replaces name or displayname
	if dat["name"] != nil {
		dat["name"] = "{{.name}}"
	}
	if dat["displayName"] != nil {
		dat["displayName"] = "{{.name}}"
	}

	jsonfile, err := json.MarshalIndent(dat, "", " ")

	if err != nil {
		util.Log.Error("error creating json file  %s", id)
		return nil, err
	}
	return jsonfile, nil
}

//isDefaultEntity returns if the object from the dynatrace API is readonly, in which case it shouldn't be downloaded
func isDefaultEntity(apiID string, dat map[string]interface{}) bool {

	switch apiID {
	case "dashboard":
		if dat["dashboardMetadata"] != nil {
			metadata := dat["dashboardMetadata"].(map[string]interface{})
			if metadata["preset"] != nil && metadata["preset"] == true {
				return true
			}
		}
		return false
	case "synthetic-location":
		return false
	case "synthetic-monitor":
		return false
	case "extension":
		return false
	case "aws-credentials":
		return false
	default:
		return false
	}
}
