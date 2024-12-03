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

package environment

import (
	"os"
	"strconv"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

const (
	ConcurrentRequestsEnvKey          = "MONACO_CONCURRENT_REQUESTS"
	ConcurrentDeploymentsEnvKey       = "MONACO_CONCURRENT_DEPLOYMENTS"
	defaultValueKey                   = "DEFAULT"
	KeyUserActionWebWaitSecondsEnvKey = "MONACO_KUA_WEB_WAIT_SECONDS"
	MaxFilenameLenKey                 = "MONACO_MAX_FILENAME_LEN"
	DeploymentReportFilename          = "MONACO_DEPLOYMENT_REPORT_FILENAME"
)

var defaultValuesInt = map[string]int{
	ConcurrentRequestsEnvKey:          5,
	ConcurrentDeploymentsEnvKey:       0,
	defaultValueKey:                   0,
	KeyUserActionWebWaitSecondsEnvKey: 1,
	MaxFilenameLenKey:                 254,
}

var logStringInt = map[string]string{
	ConcurrentRequestsEnvKey:          "Concurrent Request Limit: %d, from '%s' environment variable",
	ConcurrentDeploymentsEnvKey:       "Concurrent Deployments Limit: %d, from '%s' environment variable",
	defaultValueKey:                   "Environment variable %s: %d",
	KeyUserActionWebWaitSecondsEnvKey: "Key User Action Web wait seconds: %d, from '%s' environment variable",
}
var logStringIntDefault = map[string]string{
	ConcurrentRequestsEnvKey:          "Concurrent Request Limit: %d, '%s' environment variable is NOT set, using default value",
	ConcurrentDeploymentsEnvKey:       "Concurrent Deployments Limit: %d, '%s' environment variable is NOT set, using default value",
	defaultValueKey:                   "Environment variable %s: %d, variable is NOT set, using default value",
	KeyUserActionWebWaitSecondsEnvKey: "Key User Action Web wait seconds: %d, from '%s' environment variable is NOT set, using default value",
}

func getDefaultInt(env string) int {
	defValue, ok := defaultValuesInt[env]
	if ok {
		return defValue
	}
	return defaultValuesInt[defaultValueKey]
}

func parseEnvToInt(env string, val string) (int, bool) {
	limit, err := strconv.Atoi(val)
	if err != nil || limit < 0 {
		return getDefaultInt(env), true
	}
	return limit, false
}

func getEnvValueIntInternal(env string) (int, bool) {
	val, ok := os.LookupEnv(env)
	if ok {
		return parseEnvToInt(env, val)
	}
	return getDefaultInt(env), true
}

func getLogMessage(env string, messageMap map[string]string) string {
	logMessage, ok := messageMap[env]
	if ok {
		return logMessage
	}
	return messageMap[defaultValueKey]
}

func GetEnvValueInt(env string) int {
	value, _ := getEnvValueIntInternal(env)
	return value
}

func GetEnvValueIntLog(env string) int {
	value, isDefault := getEnvValueIntInternal(env)

	var logMessage string

	if isDefault {
		logMessage = getLogMessage(env, logStringIntDefault)
	} else {
		logMessage = getLogMessage(env, logStringInt)
	}

	log.Debug(logMessage, value, env)

	return value
}
