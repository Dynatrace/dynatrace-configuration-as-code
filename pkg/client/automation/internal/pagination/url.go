/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package pagination

import (
	"net/url"
	"strconv"
)

func NextPageURL(baseURL, path string, offset int) (string, error) {
	u, e := url.Parse(baseURL)
	if e != nil {
		return "", e
	}

	u.Path, e = url.JoinPath(u.Path, path)
	if e != nil {
		return "", e
	}

	q := u.Query()
	q.Add("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
