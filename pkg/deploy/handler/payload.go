/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package handler

import (
	"encoding/json"
	"errors"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	deployErr "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

var ErrFailedAddExternalID = errors.New("failed to add externalID to payload")

// Modifier is an exposed decorator providing possibility to modify the request payload
type Modifier func(request map[string]any, data *HandlerData) (map[string]any, error)

// Validator is an exposed decorator providing possibility to validate request payload
type Validator func(request map[string]any) error

// Generator is an exposed decorator providing possibility to generate data and add it to HandlerData
type Generator func(data *HandlerData) error

// PayloadHandler is responsible to add a generated externalID to the payload
type PayloadHandler struct {
	BaseHandler
	Generators []Generator
	Validators []Validator
	Modifiers  []Modifier
}

func (h *PayloadHandler) Handle(data *HandlerData) (entities.ResolvedEntity, error) {
	err := generate(data, h.Generators)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	var request map[string]any
	err = json.Unmarshal(data.payload, &request)
	if err != nil {
		return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c, ErrFailedAddExternalID, err)
	}

	err = validate(request, h.Validators)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	request, err = modify(request, h.Modifiers, data)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	data.payload, err = json.Marshal(request)
	if err != nil {
		return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c, ErrFailedAddExternalID, err)
	}

	if h.next != nil {
		return h.next.Handle(data)
	}

	return entities.ResolvedEntity{}, deployErr.NewFromErr(data.c, ErrUndefinedNextHandler{handler: "PayloadHandler"})
}

func generate(data *HandlerData, generators []Generator) error {
	for _, generator := range generators {
		err := generator(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func modify(request map[string]any, modifier []Modifier, data *HandlerData) (map[string]any, error) {
	var err error
	for _, modifier := range modifier {
		request, err = modifier(request, data)
		if err != nil {
			return nil, err
		}
	}

	return request, nil
}

func validate(payload map[string]any, validators []Validator) error {
	for _, validator := range validators {
		err := validator(payload)
		if err != nil {
			return err
		}
	}
	return nil
}

func ExternalIDGenerator(data *HandlerData) error {
	externalID := idutils.GenerateExternalID(data.c.Coordinate)
	data.externalID = &externalID
	return nil
}

func AddExternalID(request map[string]any, data *HandlerData) (map[string]any, error) {
	request["externalId"] = *data.externalID
	return request, nil
}
