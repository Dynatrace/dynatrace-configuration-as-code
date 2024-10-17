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

const (
	TypeInfo  string = "INFO"
	TypeWarn  string = "WARN"
	TypeError string = "ERROR"
)

type Detail struct {
	Type    string `json:"type"`
	Message string `json:"msg"`
}

type detailerContextKey struct{}

func NewContextWithDetailer(ctx context.Context, d Detailer) context.Context {
	return context.WithValue(ctx, detailerContextKey{}, d)
}

type Detailer interface {
	AddDetail(d Detail)
	GetDetails() []Detail
}

func GetDetailerFromContextOrDiscard(ctx context.Context) Detailer {
	v := ctx.Value(detailerContextKey{})
	if v == nil {
		return &discardDetailer{}
	}
	switch v := v.(type) {
	case *defaultDetailer:
		return v
	default:
		panic(fmt.Sprintf("unexpected value type for detailer context key: %T", v))
	}
}

type discardDetailer struct{}

func (_ *discardDetailer) AddDetail(_ Detail) {}

func (_ *discardDetailer) GetDetails() []Detail { return nil }

type defaultDetailer struct {
	details []Detail
}

func NewDefaultDetailer() Detailer {
	return &defaultDetailer{}
}

func (dd *defaultDetailer) AddDetail(d Detail) {
	dd.details = append(dd.details, d)
}

func (dd *defaultDetailer) GetDetails() []Detail { return dd.details }
