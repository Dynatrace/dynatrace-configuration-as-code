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

package customclient

import (
	"context"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

func ContextWithCustomClient(ctx context.Context) context.Context {
	t := &http.Transport{
		TLSHandshakeTimeout: 20 * time.Minute,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Minute, // Timeout for establishing TCP connection
			KeepAlive: 20 * time.Minute, // Keep-alive period for TCP connection
		}).DialContext,
		IdleConnTimeout:       20 * time.Minute, // How long idle connections stay in the pool
		ExpectContinueTimeout: 20 * time.Minute, // Wait time for 100-continue response
		MaxIdleConns:          100,              // Max idle connections across all hosts
		MaxIdleConnsPerHost:   100,               // Max idle connections per host
		DisableKeepAlives:     true,
	}
	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: t,
		Timeout:   20 * time.Minute,
	})
}
