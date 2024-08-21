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

package dtclient

import (
	"context"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"io"
	"net/http"
	"time"
)

type RetrySetting struct {
	WaitTime   time.Duration
	MaxRetries int
}

type RetrySettings struct {
	Normal   RetrySetting
	Long     RetrySetting
	VeryLong RetrySetting
}

var DefaultRetrySettings = RetrySettings{
	Normal: RetrySetting{
		WaitTime:   time.Second,
		MaxRetries: 15,
	},
	Long: RetrySetting{
		WaitTime:   time.Second,
		MaxRetries: 30,
	},
	VeryLong: RetrySetting{
		WaitTime:   time.Second,
		MaxRetries: 60,
	},
}

// SendRequestWithBody is a function doing a PUT or POST HTTP request
type SendRequestWithBody func(ctx context.Context, endpoint string, body io.Reader, options corerest.RequestOptions) (*http.Response, error)
