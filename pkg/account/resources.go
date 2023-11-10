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

package account

type (
	PolicyId = string
	GroupId  = string
	UserId   = string

	Resources struct {
		Policies map[PolicyId]Policy
		Groups   map[GroupId]Group
		Users    map[UserId]User
	}
	Policies struct {
		Policies []Policy
	}
	Policy struct {
		ID          string
		Name        string
		Level       any
		Description string
		Policy      string
	}
	PolicyLevelAccount struct {
		Type string
	}
	PolicyLevelEnvironment struct {
		Type        string
		Environment string
	}
	Groups struct {
		Groups []Group
	}
	Group struct {
		ID             string
		Name           string
		Description    string
		Account        *Account
		Environment    []Environment
		ManagementZone []ManagementZone
	}
	Account struct {
		Permissions []any
		Policies    []any
	}
	Environment struct {
		Name        string
		Permissions []any
		Policies    []any
	}
	ManagementZone struct {
		Environment    string
		ManagementZone string
		Permissions    []any
	}
	Users struct {
		Users []User
	}
	User struct {
		Email  string
		Groups []any
	}
	Reference struct {
		Type string
		Id   string
	}
)
