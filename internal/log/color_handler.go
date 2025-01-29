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
	"fmt"
	"log/slog"
	"strconv"
)

var _ slog.Handler = (*ColorHandler)(nil)

type ColorHandler struct {
	handler slog.Handler
}

func NewColorHandler(h slog.Handler) *ColorHandler {
	return &ColorHandler{
		handler: h,
	}
}

func (c *ColorHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return c.handler.Enabled(ctx, l)
}

func (c *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level <= slog.LevelDebug {
		r.Message = colorize(lightGray, r.Message)
	} else if r.Level <= slog.LevelInfo {
		r.Message = colorize(cyan, r.Message)
	} else if r.Level < slog.LevelWarn {
		r.Message = colorize(lightBlue, r.Message)
	} else if r.Level < slog.LevelError {
		r.Message = colorize(lightYellow, r.Message)
	} else if r.Level <= slog.LevelError+1 {
		r.Message = colorize(lightRed, r.Message)
	} else if r.Level > slog.LevelError+1 {
		r.Message = colorize(lightMagenta, r.Message)
	}

	return c.handler.Handle(ctx, r)
}

func (c *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewColorHandler(c.handler.WithAttrs(attrs))
}

func (c *ColorHandler) WithGroup(name string) slog.Handler {
	return NewColorHandler(c.handler.WithGroup(name))
}

var (
	reset = "\033[0m"

	//black        = 30
	//red          = 31
	//green        = 32
	//yellow       = 33
	//blue         = 34
	//magenta      = 35
	cyan      = 36
	lightGray = 37
	//darkGray     = 90
	lightRed = 91
	//lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	//lightCyan    = 96
	//white        = 97
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}
