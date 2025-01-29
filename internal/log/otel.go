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

package log

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/Dynatrace/OneAgent-SDK-for-Go/sdk"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
)

func initOpenTelemetryHandler() slog.Handler {

	// ===== GENERAL SETUP =====

	DT_API_HOST := os.Getenv("MONACO_OTEL_DT_API_HOST") // Only the host part of your Dynatrace URL
	DT_API_TOKEN := os.Getenv("MONACO_OTEL_DT_API_TOKEN")

	if DT_API_HOST == "" || DT_API_TOKEN == "" {
		return nil
	}

	DT_API_BASE_PATH := "/api/v2/otlp"

	authHeader := map[string]string{"Authorization": "Api-Token " + DT_API_TOKEN}

	ctx := context.Background()

	oneagentsdk := sdk.CreateInstance()
	dtMetadata := oneagentsdk.GetEnrichmentMetadata()

	var attributes []attribute.KeyValue
	for k, v := range dtMetadata {
		attributes = append(attributes, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.StringValue(v)})
	}
	attributes = append(attributes,
		semconv.ServiceNameKey.String(version.ApplicationName),
		semconv.ServiceVersionKey.String(version.MonitoringAsCode),
	)

	res, err := resource.New(ctx, resource.WithAttributes(attributes...))
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	// ===== LOG SETUP =====
	logExporter, err := otlploghttp.New(
		ctx,
		otlploghttp.WithEndpoint(DT_API_HOST),
		otlploghttp.WithURLPath(DT_API_BASE_PATH+"/v1/logs"),
		otlploghttp.WithHeaders(authHeader),
	)

	if err != nil {
		log.Fatalf("Failed to create OTLP exporter: %v", err)
	}

	lp := sdklog.NewLoggerProvider(
		//sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(lp)

	return otelslog.NewHandler("my-logger-scope", otelslog.WithLoggerProvider(lp))
}
