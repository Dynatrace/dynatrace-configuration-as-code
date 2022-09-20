package download

func sanitizeProperties(properties map[string]interface{}) map[string]interface{} {
	properties = removeIdentifyingProperties(properties)
	return replaceTemplateProperties(properties)
}

func removeIdentifyingProperties(dat map[string]interface{}) map[string]interface{} {
	dat = removeByPath(dat, []string{"metadata"})
	dat = removeByPath(dat, []string{"id"})
	dat = removeByPath(dat, []string{"identifier"})
	dat = removeByPath(dat, []string{"rules", "id"})
	dat = removeByPath(dat, []string{"rules", "methodRules", "id"})
	dat = removeByPath(dat, []string{"entityId"})

	return dat
}

func removeByPath(dat map[string]interface{}, key []string) map[string]interface{} {
	if len(key) == 0 || dat == nil || dat[key[0]] == nil {
		return dat
	}

	if len(key) == 1 {
		delete(dat, key[0])
		return dat
	}

	if field, ok := dat[key[0]].(map[string]interface{}); ok {
		dat[key[0]] = removeByPath(field, key[1:])
		return dat
	}

	if arrayOfFields, ok := dat[key[0]].([]interface{}); ok {
		for i := range arrayOfFields {
			if field, ok := arrayOfFields[i].(map[string]interface{}); ok {
				arrayOfFields[i] = removeByPath(field, key[1:])
			}
		}

		dat[key[0]] = arrayOfFields
	}
	return dat
}

func replaceTemplateProperties(dat map[string]interface{}) map[string]interface{} {
	var nameTemplate = "{{.name}}"

	if dat["name"] != nil {
		dat["name"] = nameTemplate
	} else if dat["displayName"] != nil {
		dat["displayName"] = nameTemplate
	}

	// replace dashboard name
	if dat["dashboardMetadata"] != nil {
		if t, ok := dat["dashboardMetadata"].(map[string]interface{}); ok && t["name"] != "" {
			t["name"] = nameTemplate
			dat["dashboardMetadata"] = t
		}
	}

	return dat
}
