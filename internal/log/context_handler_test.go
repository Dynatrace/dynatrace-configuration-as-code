//go:build unit

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

package log_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

func TestContextHandler_Handle(t *testing.T) {
	ctx := t.Context()
	ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: "environment1", Group: "group1"})
	ctx = context.WithValue(ctx, log.CtxKeyCoord{}, coordinate.Coordinate{Project: "project1", Type: "api1", ConfigId: "configId1"})
	ctx = context.WithValue(ctx, log.CtxKeyAccount{}, "account1")
	ctx = context.WithValue(ctx, log.CtxGraphComponentId{}, log.CtxValGraphComponentId(4))

	handler := NewTestHandler(&slog.HandlerOptions{})

	testingHandler := log.NewContextHandler(handler)

	r := slog.Record{
		Level:   slog.LevelWarn,
		Message: "test",
	}

	err := testingHandler.Handle(ctx, r)
	require.NoError(t, err)

	assert.Contains(t, handler.Output.String(), "environment.name=environment1 environment.group=group1")
	assert.Contains(t, handler.Output.String(), "coordinate.project=project1 coordinate.type=api1 coordinate.configId=configId1")
	assert.Contains(t, handler.Output.String(), "account=account1")
	assert.Contains(t, handler.Output.String(), "gid=4")
}
