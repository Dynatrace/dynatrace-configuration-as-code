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

package report_test

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDetailer_ContextWithNoDetailerDiscardsDetails tests that the Detailer obtained from an context without the default one discards details.
func TestDetailer_ContextWithNoDetailerDiscardsDetails(t *testing.T) {
	ctx := context.TODO()
	detailer := report.GetDetailerFromContextOrDiscard(ctx)
	require.NotNil(t, detailer)

	detailer.AddDetail(report.Detail{Type: report.TypeInfo, Message: "Message"})
	assert.Empty(t, report.GetDetailerFromContextOrDiscard(ctx).GetDetails())
}

// TestDetailer_ContextWithDefaultDetailerCollectsDetails tests that the Detailer obtained from an context with the default one attached collects and returns details.
func TestDetailer_ContextWithDefaultDetailerCollectsDetails(t *testing.T) {
	detail1 := report.Detail{Type: report.TypeInfo, Message: "Message1"}
	detail2 := report.Detail{Type: report.TypeWarn, Message: "Message2"}
	detail3 := report.Detail{Type: report.TypeError, Message: "Message3"}

	ctx := report.NewContextWithDetailer(context.TODO(), report.NewDefaultDetailer())
	detailer := report.GetDetailerFromContextOrDiscard(ctx)
	require.NotNil(t, detailer)

	report.GetDetailerFromContextOrDiscard(ctx).AddDetail(detail1)
	report.GetDetailerFromContextOrDiscard(ctx).AddDetail(detail2)
	report.GetDetailerFromContextOrDiscard(ctx).AddDetail(detail3)

	details := report.GetDetailerFromContextOrDiscard(ctx).GetDetails()
	require.Len(t, details, 3)
	assert.EqualValues(t, details[0], detail1)
	assert.EqualValues(t, details[1], detail2)
	assert.EqualValues(t, details[2], detail3)
}
