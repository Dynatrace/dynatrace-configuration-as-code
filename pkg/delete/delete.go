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

package delete

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"net/http"
	"reflect"
	"strings"
)

// DeletePointer contains all data needed to identify an object to be deleted from a Dynatrace environment.
// DeletePointer is similar but not fully equivalent to config.Coordinate as it may contain an Identifier that is either
// a Name or a ConfigID - only in case of a ConfigID is it actually equivalent to a Coordinate
type DeletePointer struct {
	Project string
	Type    string

	//Identifier will either be the Name of a classic Config API object, or a configID for newer types like Settings
	Identifier string
}

func (d DeletePointer) asCoordinate() coordinate.Coordinate {
	return coordinate.Coordinate{
		Project:  d.Project,
		Type:     d.Type,
		ConfigId: d.Identifier,
	}
}

func (d DeletePointer) String() string {
	if d.Project != "" {
		return d.asCoordinate().String()
	}
	return fmt.Sprintf("%s:%s", d.Type, d.Identifier)
}

type ClientSet struct {
	Classic    dtclient.Client
	Settings   dtclient.Client
	Automation automationClient
	Buckets    bucketClient
}

type automationClient interface {
	Delete(ctx context.Context, resourceType automation.ResourceType, id string) (automation.Response, error)
	List(ctx context.Context, resourceType automation.ResourceType) (automation.ListResponse, error)
}

type bucketClient interface {
	Delete(ctx context.Context, id string) (buckets.Response, error)
	List(ctx context.Context) (buckets.ListResponse, error)
}

type configurationType = string

// DeleteEntries is a map of configuration type to slice of delete pointers
type DeleteEntries = map[configurationType][]DeletePointer

// Configs removes all given entriesToDelete from the Dynatrace environment the given client connects to
func Configs(ctx context.Context, clients ClientSet, apis api.APIs, automationResources map[string]config.AutomationResource, entriesToDelete DeleteEntries) error {
	deleteErrors := 0
	for entryType, entries := range entriesToDelete {
		if targetApi, isClassicAPI := apis[entryType]; isClassicAPI {
			errs := deleteClassicConfig(ctx, clients.Classic, targetApi, entries, entryType)
			if len(errs) > 0 {
				deleteErrors += 1
			}
		} else if targetAutomation, isAutomationAPI := automationResources[entryType]; isAutomationAPI {
			if reflect.ValueOf(clients.Automation).IsNil() {
				log.WithCtxFields(ctx).WithFields(field.Type(entryType)).Warn("Skipped deletion of %d Automation configurations of type %q as API client was unavailable.", len(entries), entryType)
				continue
			}
			errs := deleteAutomations(ctx, clients.Automation, targetAutomation, entries)
			if len(errs) > 0 {
				deleteErrors += 1
			}
		} else if entryType == "bucket" {
			errs := deleteBuckets(ctx, clients.Buckets, entries)
			if len(errs) > 0 {
				deleteErrors += 1
			}
		} else { // assume it's a Settings Schema
			errs := deleteSettingsObject(ctx, clients.Settings, entries)
			if len(errs) > 0 {
				deleteErrors += 1
			}
		}
	}

	if deleteErrors > 0 {
		return fmt.Errorf("encountered %d errors", deleteErrors)
	}
	return nil
}

func deleteClassicConfig(ctx context.Context, client dtclient.Client, theApi api.API, entries []DeletePointer, targetApi string) []error {
	errors := make([]error, 0)

	values, err := client.ListConfigs(ctx, theApi)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to fetch existing configs of api `%v`. Skipping deletion all configs of this api. Reason: %w", theApi.ID, err))
	}

	values, errs := filterValuesToDelete(ctx, entries, values, theApi.ID)
	errors = append(errors, errs...)

	log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID)).Info("Deleting %d config(s) of type %q...", len(entries), theApi.ID)

	if len(values) == 0 {
		log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID)).Debug("No values found to delete for type %q.", targetApi)
	}

	for _, v := range values {
		log.WithCtxFields(ctx).WithFields(field.Type(theApi.ID), field.F("value", v)).Debug("Deleting %s:%s (%s)", targetApi, v.Name, v.Id)
		if err := client.DeleteConfigById(theApi, v.Id); err != nil {
			errors = append(errors, fmt.Errorf("could not delete %s:%s (%s): %w", theApi.ID, v.Name, v.Id, err))
		}
	}

	return errors
}

func deleteSettingsObject(ctx context.Context, c dtclient.Client, entries []DeletePointer) []error {
	errors := make([]error, 0)

	if len(entries) > 0 {
		log.WithCtxFields(ctx).WithFields(field.Type(entries[0].Type)).Info("Deleting %d config(s) of type %q...", len(entries), entries[0].Type)
	}

	for _, e := range entries {

		ctx = context.WithValue(ctx, log.CtxKeyCoord{}, e.asCoordinate())

		if e.Project == "" {
			log.WithCtxFields(ctx).Warn("Generating legacy externalID for deletion of %q - this will fail to identify a newer Settings object. Consider defining a 'project' for this delete entry.", e)
		}
		externalID, err := idutils.GenerateExternalID(e.asCoordinate())

		if err != nil {
			errors = append(errors, fmt.Errorf("unable to generate externalID for %s: %w", e, err))
			continue
		}
		// get settings objects with matching external ID
		objects, err := c.ListSettings(ctx, e.Type, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
		if err != nil {
			errors = append(errors, fmt.Errorf("could not fetch settings 2.0 objects with schema ID %s: %w", e.Type, err))
			continue
		}

		if len(objects) == 0 {
			log.WithCtxFields(ctx).Debug("No settings object found to delete for %s", e)
			continue
		}

		for _, obj := range objects {
			if obj.ModificationInfo != nil && !obj.ModificationInfo.Deletable {
				log.WithCtxFields(ctx).WithFields(field.F("object", obj)).Warn("Requested settings object %s (%s) is not deletable.", e, obj.ObjectId)
				continue
			}

			log.WithCtxFields(ctx).Debug("Deleting settings object %s with objectId %q.", e, obj.ObjectId)
			err := c.DeleteSettings(obj.ObjectId)
			if err != nil {
				errors = append(errors, fmt.Errorf("could not delete settings 2.0 object %s with object ID %s: %w", e, obj.ObjectId, err))
			}
		}
	}

	return errors
}

func deleteAutomations(ctx context.Context, c automationClient, automationResource config.AutomationResource, entries []DeletePointer) []error {
	log.WithCtxFields(ctx).WithFields(field.Type(string(automationResource))).Info("Deleting %d config(s) of type %q...", len(entries), automationResource)
	errors := make([]error, 0)

	for _, e := range entries {

		id := idutils.GenerateUUIDFromCoordinate(e.asCoordinate())

		resourceType, err := automationutils.ClientResourceTypeFromConfigType(automationResource)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not delete Automation object %s with ID %q: %w", e, id, err))
		}

		resp, err := c.Delete(ctx, resourceType, id)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not delete Automation object %s with ID %q: %w", e, id, err))
		}

		if err, isErr := resp.AsAPIError(); isErr && resp.StatusCode != http.StatusNotFound { // 404 means it's gone already anyway
			errors = append(errors, fmt.Errorf("could not delete Automation object %s with ID %q: %w", e, id, err))
		}
	}

	return errors
}

func deleteBuckets(ctx context.Context, c bucketClient, entries []DeletePointer) []error {
	log.WithCtxFields(ctx).WithFields(field.Type("bucket")).Info("Deleting %d config(s) of type %q...", len(entries), "bucket")
	errors := make([]error, 0)
	for _, e := range entries {
		bucketName := idutils.GenerateBucketName(e.asCoordinate())
		resp, err := c.Delete(ctx, bucketName)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not delete Bucket Definition object %s with name %q: %w", e, bucketName, err))
		}
		if err, ok := resp.AsAPIError(); ok && err.StatusCode != http.StatusNotFound {
			errors = append(errors, fmt.Errorf("could not delete Bucket Definition object %s with name %q: %w", e, bucketName, err))
		}
	}
	return errors
}

// filterValuesToDelete filters the given values for only values we want to delete.
// We first search the names of the config-to-be-deleted, and if we find it, return them.
// If we don't find it, we look if the name is actually an id, and if we find it, return them.
// If a given name is found multiple times, we return an error for each name.
func filterValuesToDelete(ctx context.Context, entries []DeletePointer, existingValues []dtclient.Value, apiName string) ([]dtclient.Value, []error) {

	toDeleteByName := make(map[string][]dtclient.Value, len(entries))
	valuesById := make(map[string]dtclient.Value, len(existingValues))

	for _, v := range existingValues {
		valuesById[v.Id] = v

		for _, entry := range entries {
			if toDeleteByName[entry.Identifier] == nil {
				toDeleteByName[entry.Identifier] = []dtclient.Value{}
			}

			if v.Name == entry.Identifier {
				toDeleteByName[entry.Identifier] = append(toDeleteByName[entry.Identifier], v)
			}
		}
	}

	result := make([]dtclient.Value, 0, len(entries))

	var errs []error

	for name, valuesToDelete := range toDeleteByName {

		switch len(valuesToDelete) {
		case 1:
			result = append(result, valuesToDelete[0])
		case 0:
			v, found := valuesById[name]

			if found {
				result = append(result, v)
			} else {
				log.WithCtxFields(ctx).WithFields(field.Type(apiName), field.F("expectedID", name)).Debug("No config of type %s found with the name or ID %q", apiName, name)
			}

		default:
			// multiple configs with this name found -> error
			errs = append(errs, fmt.Errorf("multiple configs of type %s found with the name %q. Configs: %v", apiName, name, valuesToDelete))
		}
	}

	return result, errs
}

// AllConfigs collects and deletes classic API configuration objects using the provided ConfigClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - client (dtclient.ConfigClient): An implementation of the ConfigClient interface for managing configuration objects.
//   - apis (api.APIs): A list of APIs for which configuration values need to be collected and deleted.
//
// Returns:
//   - []error: A slice of errors encountered during the collection and deletion of configuration values.
func AllConfigs(ctx context.Context, client dtclient.ConfigClient, apis api.APIs) (errors []error) {

	for _, a := range apis {
		log.WithCtxFields(ctx).WithFields(field.Type(a.ID)).Info("Collecting configs of type %q...", a.ID)
		values, err := client.ListConfigs(ctx, a)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(a.ID)).Info("Deleting %d configs of type %q...", len(values), a.ID)

		for _, v := range values {
			log.WithCtxFields(ctx).WithFields(field.Type(a.ID), field.F("value", v)).Debug("Deleting config %s:%s...", a.ID, v.Id)
			// TODO(improvement): this could be improved by filtering for default configs the same way as Download does
			err := client.DeleteConfigById(a, v.Id)

			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// AllSettingsObjects collects and deletes settings objects using the provided SettingsClient.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (dtclient.SettingsClient): An implementation of the SettingsClient interface for managing settings objects.
//
// Returns:
//   - []error: A slice of errors encountered during the collection and deletion of settings objects.
func AllSettingsObjects(ctx context.Context, c dtclient.SettingsClient) []error {
	var errs []error

	schemas, err := c.ListSchemas()
	if err != nil {
		return []error{fmt.Errorf("failed to fetch settings schemas. No settings will be deleted. Reason: %w", err)}
	}

	schemaIds := make([]string, len(schemas))
	for i := range schemas {
		schemaIds[i] = schemas[i].SchemaId
	}

	log.WithCtxFields(ctx).Debug("Deleting settings of schemas %v...", schemaIds)

	for _, s := range schemaIds {
		log.WithCtxFields(ctx).WithFields(field.Type(s)).Info("Collecting objects of type %q...", s)
		settings, err := c.ListSettings(ctx, s, dtclient.ListSettingsOptions{DiscardValue: true})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(s)).Info("Deleting %d objects of type %q...", len(settings), s)
		for _, setting := range settings {
			if setting.ModificationInfo != nil && !setting.ModificationInfo.Deletable {
				continue
			}

			log.WithCtxFields(ctx).WithFields(field.Type(s), field.F("object", setting)).Debug("Deleting settings object with objectId %q...", setting.ObjectId)
			err := c.DeleteSettings(setting.ObjectId)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

// AllAutomations collects and deletes automations resources using the given automation client.
//
// Parameters:
//   - ctx (context.Context): The context in which the function operates.
//   - c (automationClient): An implementation of the automationClient interface for performing automation-related operations.
//
// Returns:
//   - []error: A slice of errors encountered during the collection and deletion of automations.
func AllAutomations(ctx context.Context, c automationClient) []error {
	var errs []error

	resources := []config.AutomationResource{config.Workflow, config.BusinessCalendar, config.SchedulingRule}
	for _, resource := range resources {
		t, err := automationutils.ClientResourceTypeFromConfigType(resource)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(string(resource))).Info("Collecting objects of type %q...", resource)
		resp, err := c.List(ctx, t)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		objects, err := automationutils.DecodeListResponse(resp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.WithCtxFields(ctx).WithFields(field.Type(string(resource))).Info("Deleting %d objects of type %q...", len(objects), resource)
		for _, o := range objects {
			log.WithCtxFields(ctx).WithFields(field.Type(string(resource)), field.F("object", o)).Debug("Deleting Automation object with id %q...", o.ID)
			resp, err := c.Delete(ctx, t, o.ID)
			if err != nil {
				errs = append(errs, err)
			}
			if err, isErr := resp.AsAPIError(); isErr && resp.StatusCode != http.StatusNotFound { // 404 means it's gone already anyway
				errs = append(errs, err)
			}
		}
	}

	return errs
}

// AllBuckets collects and deletes objects of type "bucket" using the provided bucketClient.
//
// Parameters:
//   - ctx (context.Context): The context for the operation.
//   - c (bucketClient): The bucketClient used for listing and deleting objects.
//
// Returns:
//   - []error: A slice of errors encountered during the operation. It may contain listing errors,
//     deletion errors, or API errors.
func AllBuckets(ctx context.Context, c bucketClient) []error {
	var errs []error

	log.WithCtxFields(ctx).WithFields(field.Type("bucket")).Info("Collecting objects of type %q...", "bucket")
	response, err := c.List(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	if err, ok := response.AsAPIError(); ok {
		errs = append(errs, err)
	}

	log.WithCtxFields(ctx).WithFields(field.Type("bucket")).Info("Deleting %d objects of type %q...", len(response.Objects), "bucket")
	for _, obj := range response.Objects {
		var bucketName struct {
			BucketName string `json:"bucketName"`
		}

		if err := json.Unmarshal(obj, &bucketName); err != nil {
			errs = append(errs, err)
			continue
		}

		// exclude builtin bucket names, they cannot be deleted anyway
		if strings.HasPrefix(bucketName.BucketName, "dt_") || strings.HasPrefix(bucketName.BucketName, "default_") {
			continue
		}

		result, err := c.Delete(ctx, bucketName.BucketName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err, ok := result.AsAPIError(); ok && result.StatusCode != http.StatusNotFound { // 404 means it's gone already anyway
			errs = append(errs, err)
			continue
		}
	}
	return errs
}
