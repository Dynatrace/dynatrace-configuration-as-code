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

package setting

import (
	"errors"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type DeprecatedSchemaValidator struct{}

var deprecatedSchemas = map[string]string{
	"builtin:span-attribute":       "this setting was replaced by 'builtin:attribute-allow-list' and 'builtin:attribute-masking'",
	"builtin:span-event-attribute": "this setting was replaced by 'builtin:attribute-allow-list' and 'builtin:attribute-masking'",
	"builtin:resource-attribute":   "this setting was replaced by 'builtin:attribute-allow-list' and 'builtin:attribute-masking'",
}

// Validate checks for each settings type whether it is using a deprecated schema.
func (v *DeprecatedSchemaValidator) Validate(_ []project.Project, c config.Config) error {

	s, ok := c.Type.(config.SettingsType)
	if !ok {
		return nil
	}

	if msg, deprecated := deprecatedSchemas[s.SchemaId]; deprecated {
		log.WithFields(field.Coordinate(c.Coordinate), field.Environment(c.Environment, c.Group)).Warn("Schema %q is deprecated - please update your configurations: %s", s.SchemaId, msg)
	}

	return nil
}

var (
	errDiffSchema                = errors.New("different schemas")
	errDiffScope                 = errors.New("different scopes")
	errReferencedProjectNotFound = errors.New("referenced project does not exist")
	errReferencedNotFound        = errors.New("reference not found")
)

type insertAfterSameScopeError struct {
	cause error

	source, target coordinate.Coordinate
}

func (e *insertAfterSameScopeError) Error() string {
	return fmt.Sprintf("configuration '%s' insertAfter references '%s': %s", e.source, e.target, e.cause)
}

func (e *insertAfterSameScopeError) Unwrap() error {
	return e.cause
}

func NewInsertAfterSameScopeError(source, target coordinate.Coordinate, cause error) error {
	return &insertAfterSameScopeError{
		source: source,
		target: target,
		cause:  cause,
	}
}

// InsertAfterSameScopeValidator verifies that if a config has an insertAfter, that the referenced config's scope is the same.
// This only works if both scopes are 'static' data and not references or something similar.
type InsertAfterSameScopeValidator struct{}

func (InsertAfterSameScopeValidator) Validate(projects []project.Project, conf config.Config) error {

	if conf.Skip {
		return nil
	}

	if conf.Type.ID() != config.SettingsTypeID {
		return nil
	}

	targetCoordinate := extractInsertAfterReference(conf)
	if targetCoordinate == (coordinate.Coordinate{}) { // no insertAfter defined
		return nil
	}

	if targetCoordinate.Type != conf.Coordinate.Type {
		return NewInsertAfterSameScopeError(conf.Coordinate, targetCoordinate, errDiffSchema)
	}

	proj, f := findProjectByName(projects, targetCoordinate.Project)
	if !f {
		return NewInsertAfterSameScopeError(conf.Coordinate, targetCoordinate, errReferencedProjectNotFound)
	}

	targetConf, f := proj.GetConfigFor(conf.Environment, targetCoordinate)
	if !f {
		return NewInsertAfterSameScopeError(conf.Coordinate, targetCoordinate, errReferencedNotFound)
	}

	configScope := extractScope(conf)
	if configScope == "" {
		return nil
	}

	targetScope := extractScope(targetConf)
	if targetScope == "" {
		return nil
	}

	if configScope != targetScope {
		return NewInsertAfterSameScopeError(conf.Coordinate, targetCoordinate, errDiffScope)
	}

	return nil
}

func findProjectByName(projects []project.Project, projectName string) (project.Project, bool) {
	for _, p := range projects {
		if p.Id == projectName {
			return p, true
		}
	}

	return project.Project{}, false
}

func extractInsertAfterReference(c config.Config) coordinate.Coordinate {
	param, f := c.Parameters[config.InsertAfterParameter]
	if !f {
		return coordinate.Coordinate{}
	}

	refParameter, ok := param.(*refParam.ReferenceParameter)
	if !ok {
		log.
			WithFields(field.Coordinate(c.Coordinate), field.Environment(c.Environment, c.Group)).
			Debug("Can't perform InsertAfterSameScopeValidator check: InsertAfter is not a reference but '%s'", param.GetType())

		return coordinate.Coordinate{}
	}

	return refParameter.Config
}

func extractScope(c config.Config) string {
	param, f := c.Parameters[config.ScopeParameter]
	if !f {
		return ""
	}

	valueParameter, ok := param.(*valueParam.ValueParameter)
	if !ok {
		log.
			WithFields(field.Coordinate(c.Coordinate), field.Environment(c.Environment, c.Group)).
			Debug("Can't perform InsertAfterSameScopeValidator check: Scope is not a plain value but '%s'", param.GetType())

		return ""
	}

	value, ok := valueParameter.Value.(string)
	if !ok {
		log.
			WithFields(field.Coordinate(c.Coordinate), field.Environment(c.Environment, c.Group)).
			Debug("Can't perform InsertAfterSameScopeValidator check: Scope is not a simple value: '%v'", valueParameter.Value)

		return ""
	}

	return value
}
