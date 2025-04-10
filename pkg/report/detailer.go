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

package report

import (
	"context"
	"fmt"
)

type DetailType = string

const (
	// DetailTypeInfo indicates a detail of type info.
	DetailTypeInfo DetailType = "INFO"

	// DetailTypeWarn indicates a detail of type warning.
	DetailTypeWarn DetailType = "WARN"

	// DetailTypeError indicates a detail of type error.
	DetailTypeError DetailType = "ERROR"
)

// Detail represents additional information produced during the deployment of an configuration.
type Detail struct {
	// Type is the type of detail: info, warning or error.
	Type DetailType `json:"type"`

	// Message is the message of the detail.
	Message string `json:"msg"`
}

type detailerContextKey struct{}

// NewContextWithDetailer returns a copy of the specified Context associated with the specified Detailer.
func NewContextWithDetailer(ctx context.Context, d Detailer) context.Context {
	return context.WithValue(ctx, detailerContextKey{}, d)
}

// // Reporter is a minimal interface for recording and retrieving details.
type Detailer interface {

	// Add adds a Detail to the Detailer.
	Add(d Detail)

	// GetAll gets all Details stored by the Detailer.
	GetAll() []Detail
}

// GetDetailerFromContextOrDiscard gets the Detailer associated with the Context or returns a discarding Detailer if none is available.
func GetDetailerFromContextOrDiscard(ctx context.Context) Detailer {
	v := ctx.Value(detailerContextKey{})
	if v == nil {
		return &discardDetailer{}
	}
	switch v := v.(type) {
	case Detailer:
		return v
	default:
		panic(fmt.Sprintf("unexpected value type for detailer context key: %T", v))
	}
}

// discardDetailer implements Detailer interface but does nothing.
type discardDetailer struct{}

func (*discardDetailer) Add(_ Detail) {}

func (*discardDetailer) GetAll() []Detail { return nil }

type defaultDetailer struct {
	details []Detail
}

// NewDefaultDetailer creates a Detailer that simply stores Details in a slice.
func NewDefaultDetailer() Detailer {
	return &defaultDetailer{}
}

// Add adds a Detail.
func (dd *defaultDetailer) Add(d Detail) {
	dd.details = append(dd.details, d)
}

// GetAll gets all Details.
func (dd *defaultDetailer) GetAll() []Detail { return dd.details }
