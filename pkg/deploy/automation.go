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

package deploy

import (
	"errors"
	client "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation" //TODO: rename to something better
)

type Automation struct {
	client automationClient
}

func New(cli *client.Client) (*Automation, error) {
	if cli == nil {
		return nil, errors.New("client isn't valid")
	}
	return &Automation{client: cli}, nil
}

//go:generate mockgen -source=automation.go -destination=automation_mock.go -package=deploy automationClient
type automationClient interface {
	Upsert(resourceType client.ResourceType, id string, data []byte) (result *client.Response, err error)
}
