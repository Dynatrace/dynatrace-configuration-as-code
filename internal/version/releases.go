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

package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type release struct {
	TagName string `json:"tag_name"`
}

func GetLatestVersion(ctx context.Context, client *http.Client, url string) (Version, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return UnknownVersion, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return UnknownVersion, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UnknownVersion, fmt.Errorf("failed to fetch release data. Status code: %d", resp.StatusCode)
	}

	var release release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UnknownVersion, fmt.Errorf("unable to parse response data: %w", err)
	}

	tagName, _ := strings.CutPrefix(release.TagName, "v")
	if parsedVersion, err := ParseVersion(tagName); err != nil {
		return UnknownVersion, err
	} else {
		return parsedVersion, nil
	}
}
