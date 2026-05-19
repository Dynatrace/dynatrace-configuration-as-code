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

package loader

import (
	"fmt"
	"net/url"
	"strings"
)

// validateDynatraceDomain checks that a URL's hostname is on the Dynatrace domain allowlist (dynatrace.com, dynatracelabs.com).
// Used for tokenEndpoint and apiUrl to prevent SSRF and credential redirection attacks.
func validateDynatraceDomain(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("not a valid URL: %w", err)
	}

	host := strings.ToLower(parsed.Hostname())
	if strings.HasSuffix(host, ".dynatracelabs.com") || strings.HasSuffix(host, ".dynatrace.com") {
		return nil
	}

	return fmt.Errorf("host %q is not allowed: must be on a '.dynatrace.com' or '.dynatracelabs.com' domain", host)
}
