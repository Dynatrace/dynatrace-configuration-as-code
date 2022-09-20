package download

import "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"

type apiFilter struct {
	// shouldBeSkippedPreDownload is an optional callback indicating that a config should not be downloaded after the list of the configs
	shouldBeSkippedPreDownload func(value api.Value) bool

	// shouldConfigBePersisted is an optional callback to check whether a config should be persisted after being downloaded
	shouldConfigBePersisted func(json map[string]interface{}) bool
}

var apiFilters = map[string]apiFilter{
	"dashboard": {
		shouldBeSkippedPreDownload: func(value api.Value) bool {
			return value.Owner != nil && *value.Owner == "Dynatrace"
		},
		shouldConfigBePersisted: func(json map[string]interface{}) bool {
			if json["dashboardMetadata"] != nil {
				metadata := json["dashboardMetadata"].(map[string]interface{})

				if metadata["preset"] != nil && metadata["preset"] == true {
					return false
				}
			}

			return true
		},
	},
	"synthetic-location": {
		shouldConfigBePersisted: func(json map[string]interface{}) bool {
			return json["type"] != "PRIVATE"
		},
	},
}

func shouldConfigBeSkipped(a api.Api, value api.Value) bool {
	if cases := apiFilters[a.GetId()]; cases.shouldBeSkippedPreDownload != nil {
		return cases.shouldBeSkippedPreDownload(value)
	}

	return false
}

func shouldConfigBePersisted(a api.Api, json map[string]interface{}) bool {
	if cases := apiFilters[a.GetId()]; cases.shouldConfigBePersisted != nil {
		return cases.shouldConfigBePersisted(json)
	}

	return true
}
