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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetailer_ContextWithNoDetailerDiscardsDetails tests that the Detailer obtained from an context without the default one discards details.
func TestDetailer_ContextWithNoDetailerDiscardsDetails(t *testing.T) {
	ctx := context.TODO()
	GetDetailerFromContextOrDiscard(ctx).AddDetail(Detail{Type: TypeInfo, Message: "Message"})
	assert.Empty(t, GetDetailerFromContextOrDiscard(ctx).GetDetails())
}

// TestDetailer_ContextWithDefaultDetailerCollectsDetails tests that the Detailer obtained from an context with the default one attached collects and returns details.
func TestDetailer_ContextWithDefaultDetailerCollectsDetails(t *testing.T) {
	detail1 := Detail{Type: TypeInfo, Message: "Message1"}
	detail2 := Detail{Type: TypeWarn, Message: "Message2"}
	detail3 := Detail{Type: TypeError, Message: "Message3"}

	ctx := NewContextWithDetailer(context.TODO(), NewDefaultDetailer())
	GetDetailerFromContextOrDiscard(ctx).AddDetail(detail1)
	GetDetailerFromContextOrDiscard(ctx).AddDetail(detail2)
	GetDetailerFromContextOrDiscard(ctx).AddDetail(detail3)

	details := GetDetailerFromContextOrDiscard(ctx).GetDetails()
	require.Len(t, details, 3)
	assert.EqualValues(t, details[0], detail1)
	assert.EqualValues(t, details[1], detail2)
	assert.EqualValues(t, details[2], detail3)
}
