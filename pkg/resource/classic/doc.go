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

/*
Package downloader is responsible for downloading configurations from Dynatrace.

The process looks like this:

	result = array[api, templates]

	for each api in apis:
		to_download = array[api, list]
		if api.isSingletonApi:
			to_download += (api, mock_download_value(api)) # Since we don't need a list-call, we can simply add this value to the array.
		else
			list <- dynatrace.ListAll(api) # List query for all configs of api
			list <- filter(list) # Remove unwanted values we already know we don't want to download
			to_download += (api, list)
		fi

		templates = array[]
		for each value in 'to_download':
			config <- dynatrace.Get(value)
			if skip(config) next # skip configs like presets

			template <- extract_and_sanitize(config)

			templates += template
		end_for

		results += (api, templates)
	end_for

	return result

```
*/
package classic
