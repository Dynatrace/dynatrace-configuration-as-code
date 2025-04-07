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

package validate

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/setting"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type Validator interface {
	Validate(projects []project.Project, config config.Config) error
}

// Validate verifies that the passed projects are sound to an extent that can be checked before deployment.
// This means, that only checks can be performed that work on 'static' data.
func Validate(environments []project.Environment) error {
	defaultValidators := []Validator{
		classic.NewValidator(),
		classic.NewDeprecatedApiValidator(),
		&setting.DeprecatedSchemaValidator{},
		&setting.InsertAfterSameScopeValidator{},
	}

	return validate(environments, defaultValidators)
}

func validate(environments []project.Environment, validators []Validator) error {
	errs := make(errors.EnvironmentDeploymentErrors)

	for _, e := range environments {
		for _, p := range e.Projects {
			p.ForEveryConfigDo(func(c config.Config) {
				for _, v := range validators {
					if err := v.Validate(e.Projects, c); err != nil {
						errs = errs.Append(c.Environment, err)
					}
				}
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
