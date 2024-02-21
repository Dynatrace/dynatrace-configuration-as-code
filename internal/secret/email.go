/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package secret

import (
	"fmt"
	"regexp"
	"strings"
)

// MaskedMail is a string that masks parts of its string representation
// if it's an email. If it's no valid email address the value is unchanged
type MaskedMail string

func (m MaskedMail) String() string {
	return maskMail(string(m))
}

func maskMail(str string) string {
	if !isValidEmail(str) {
		return str
	}
	parts := strings.Split(str, "@")
	if len(parts) != 2 {
		return "Invalid email format"
	}

	domainParts := strings.SplitN(parts[1], ".", 2)
	uname, domain, tlDomain := parts[0], domainParts[0], domainParts[1]

	if len(uname) >= 3 {
		uname = uname[:2] + "***"
	} else {
		uname = "***"
	}
	if len(domain) >= 3 {
		domain = domain[:2] + "***"
	} else {
		domain = "***"
	}
	return fmt.Sprintf("%s@%s.%s", uname, domain, tlDomain)
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
