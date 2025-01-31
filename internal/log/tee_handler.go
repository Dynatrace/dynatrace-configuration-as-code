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
	"errors"
	"log/slog"
	"slices"
	"sync"
)

var _ slog.Handler = (*TeeHandler)(nil)

type TeeHandler struct {
	mu       *sync.Mutex
	handlers []slog.Handler
}

func NewTeeHandler(h ...slog.Handler) *TeeHandler {
	return &TeeHandler{
		handlers: h,
		mu:       &sync.Mutex{},
	}
}

func (t *TeeHandler) Enabled(ctx context.Context, l slog.Level) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	e := false
	for _, h := range t.handlers {
		e = e || h.Enabled(ctx, l)
	}
	return e
}

func (t *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	errs := []error{}
	for _, h := range t.handlers {
		errs = append(errs, h.Handle(ctx, r))
	}
	return errors.Join(errs...)
}

func (t *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := []slog.Handler{}
	for _, h := range t.handlers {
		newHandlers = append(newHandlers, h.WithAttrs(slices.Clone(attrs)))
	}
	return NewTeeHandler(newHandlers...)
}

func (t *TeeHandler) WithGroup(name string) slog.Handler {
	newHandlers := []slog.Handler{}
	for _, h := range t.handlers {
		newHandlers = append(newHandlers, h.WithGroup(name))
	}
	return NewTeeHandler(newHandlers...)
}
