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

package logging

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/loggers"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

func LogProjectsInfo(environments []project.Environment) {
	projectNames := make(map[string]struct{})

	for _, e := range environments {
		for _, p := range e.Projects {
			projectNames[p.String()] = struct{}{}
		}
	}
	log.Info("Projects to be deployed (%d):", len(projectNames))

	for p, _ := range projectNames {
		log.Info("  - %s", p)
	}

	if log.Level() == loggers.LevelDebug {
		logConfigInfo(environments)
	}
}

func logConfigInfo(environments []project.Environment) {
	log.Debug("Configurations per environment:")
	for _, env := range environments {
		log.Debug("  - %s:\t%d configurations", env, len(env.AllConfigs()))
	}
}

func LogEnvironmentsInfo(environments manifest.Environments) {
	log.Info("Environments to deploy to (%d):", len(environments))
	for _, name := range environments.Names() {
		log.Info("  - %s", name)
	}
}
func LogDeploymentInfo(dryRun bool, envName string) {
	if dryRun {
		log.Info("Validating configurations for environment `%s`...", envName)
	} else {
		log.Info("Deploying configurations to environment `%s`...", envName)
	}
}

func GetOperationNounForLogging(dryRun bool) string {
	if dryRun {
		return "Validation"
	}
	return "Deployment"
}
