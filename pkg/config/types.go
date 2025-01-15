/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package config

const (
	SettingsTypeID     TypeID = "settings"
	ClassicApiTypeID   TypeID = "classic"
	EntityTypeID       TypeID = "entity"
	AutomationTypeID   TypeID = "automation"
	BucketTypeID       TypeID = "bucket"
	DocumentTypeID     TypeID = "document"
	OpenPipelineTypeID TypeID = "openpipeline"
	SegmentID          TypeID = "segment"
)

var _ Type = SettingsType{}

type SettingsType struct {
	SchemaId, SchemaVersion string
}

func (SettingsType) ID() TypeID {
	return SettingsTypeID
}

var _ Type = ClassicApiType{}

type ClassicApiType struct {
	Api string
}

func (ClassicApiType) ID() TypeID {
	return ClassicApiTypeID
}

var _ Type = EntityType{}

type EntityType struct {
	EntitiesType string
}

func (EntityType) ID() TypeID {
	return EntityTypeID
}

var _ Type = AutomationType{}

// AutomationType represents any Dynatrace Platform automation-resource
type AutomationType struct {
	// Resource identifies which Automation resource is used in this config.
	// Currently, this can be Workflow, BusinessCalendar, or SchedulingRule.
	Resource AutomationResource
}

// AutomationResource defines which resource is an AutomationType
type AutomationResource string

const (
	Workflow         AutomationResource = "workflow"
	BusinessCalendar AutomationResource = "business-calendar"
	SchedulingRule   AutomationResource = "scheduling-rule"
)

func (AutomationType) ID() TypeID {
	return AutomationTypeID
}

var _ Type = BucketType{}

type BucketType struct{}

func (BucketType) ID() TypeID {
	return BucketTypeID
}

var _ Type = DocumentType{}

// DocumentType represents a Dynatrace platform document.
type DocumentType struct {
	// Kind indicates the type of document.
	Kind DocumentKind

	// Private indicates if a document is private, otherwise by default it is visible to other users.
	Private bool
}

// DocumentKind defines the type of document. Currently, it can be a dashboard or a notebook.
type DocumentKind string

const (
	DashboardKind DocumentKind = "dashboard"
	NotebookKind  DocumentKind = "notebook"
	LaunchpadKind DocumentKind = "launchpad"
)

var KnownDocumentKinds = []DocumentKind{
	DashboardKind,
	NotebookKind,
	LaunchpadKind,
}

func (DocumentType) ID() TypeID {
	return DocumentTypeID
}

var _ Type = OpenPipelineType{}

// OpenPipelineType represents an OpenPipeline configuration.
type OpenPipelineType struct {
	// Kind indicates the type of OpenPipeline.
	Kind string
}

func (OpenPipelineType) ID() TypeID {
	return OpenPipelineTypeID
}

var _ Type = Segment{}

type Segment struct{}

func (Segment) ID() TypeID {
	return SegmentID
}
