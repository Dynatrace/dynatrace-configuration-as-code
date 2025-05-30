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

package log

import (
	"context"
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// ContextHandler is an implementation of slog.Handler that extracts attributes from the context provided in a log calls, adds them to the record and delegates to the specified handler.
type ContextHandler struct {
	handler slog.Handler
}

// NewContextHandler creates a new ContextHandler wrapping the specified handler.
func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{
		handler: h,
	}
}

// Enabled returns true iff the wrapped handler in enabled for the specified level.
func (h *ContextHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.handler.Enabled(ctx, l)
}

// Handle adds attributes extracted from the context before calling Handle on the wrapped Handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if e, ok := ctx.Value(CtxKeyEnv{}).(CtxValEnv); ok {
			r.AddAttrs(slog.Group("environment",
				slog.String("name", e.Name),
				slog.String("group", e.Group)))
		}

		if c, ok := ctx.Value(CtxKeyCoord{}).(coordinate.Coordinate); ok {
			r.AddAttrs(slog.Any("coordinate", c))
		}

		if a, ok := ctx.Value(CtxKeyAccount{}).(string); ok {
			r.AddAttrs(slog.String("account", a))
		}
		if c, ok := ctx.Value(CtxGraphComponentId{}).(CtxValGraphComponentId); ok {
			r.AddAttrs(slog.Int("gid", int(c)))
		}
	}

	return h.handler.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler created by calling WithAttr on the wrapped handler.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{handler: h.handler.WithAttrs(attrs)}
}

// WithGroup returns a new ContextHandler created by calling WithGroup on the wrapped handler.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{handler: h.handler.WithGroup(name)}
}
