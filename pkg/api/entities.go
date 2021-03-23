/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package api

type ValuesResponse struct {
	Values []Value `json:"values"`
}

type SyntheticLocationResponse struct {
	Locations []SyntheticValue `json:"locations"`
}

type SyntheticMonitorsResponse struct {
	Monitors []SyntheticValue `json:"monitors"`
}

type ManagedUserConfigResponse struct {
	Users []UserConfig
}

type Value struct {
	Id    string  `json:"id"`
	Name  string  `json:"name"`
	Owner *string `json:"owner,omitempty"`
}

type SyntheticValue struct {
	Name          string    `json:"name"`
	EntityId      string    `json:"entityId"`
	Type          string    `json:"type"`
	CloudPlatform *string   `json:"cloudPlatform"`
	Ips           *[]string `json:"ips"`
	Stage         *string   `json:"stage"`
	Enabled       *bool     `json:"enabled"`
}

type SyntheticEntity struct {
	EntityId string `json:"entityId"`
}

type DynatraceEntity struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UserConfig struct {
	Id                string   `json:"id"`
	Email             string   `json:"email`
	FirstName         string   `json:"firstName"`
	LastName          string   `json:"lastName"`
	PasswordClearText string   `json:"passwordClearText"`
	Groups            []string `json:"groups"`
}

type GroupConfig struct {
	Id                 string       `json:"id"`
	Name               string       `json:"name`
	LDAPGroupNames     []string     `json:"ldapGroupNames"`
	SSOGroupNames      []string     `json:"ssoGroupNames"`
	AccessRight        *AccessRight `json:"accessRight"`
	IsCluterAdminGroup bool         `json:"isClusterAdminGroup"`
	IsAccessAccount    bool         `json:"isAccessAccount"`
	IsManageAccount    bool         `json:"isManageAccount"`
}

type AccessRight struct {
	Rights *string `json:"rights"`
}
type ManagementZone struct {
	GroupId                     string                        `json:"groupId"`
	MzPermissionsPerEnvironment []MzPermissionsPerEnvironment `json:"mzPermissionsPerEnvironment"`
}

type MzPermissionsPerEnvironment struct {
	EnvironmentUuid string              `json:"environmentUuid"`
	MzPermissions   []MzPermissionsList `json:"mzPermissions"`
}

type MzPermissionsList struct {
	MzId        string   `json:"mzId"`
	Permissions []string `json:"permissions"`
}

type Preferences struct {
	CertificateManagementEnabled   bool `json:"certificateManagementEnabled"`
	CertificateManagementPossible  bool `json:"certificateManagementPossible"`
	SupportSendBilling             bool `json:"supportSendBilling"`
	SuppressNonBillingRelevantData bool `json:"suppressNonBillingRelevantData"`
	SupportSendClusterHealth       bool `json:"supportSendClusterHealth"`
	SupportSendEvents              bool `json:"supportSendEvents"`
	SupportAllowRemoteAccess       bool `json:"supportAllowRemoteAccess"`
	RemoteAccessOnDemandOnly       bool `json:"remoteAccessOnDemandOnly"`
	CommunityCreateUser            bool `json:"communityCreateUser"`
	CommunityExternalSearch        bool `json:"communityExternalSearch"`
	RuxitMonitorsRuxit             bool `json:"ruxitMonitorsRuxit"`
	WoopraIntegration              bool `json:"woopraIntegration"`
	TelemetrySharing               bool `json:"telemetrySharing"`
	HelpChatEnabled                bool `json:"helpChatEnabled"`
	ReadOnlyRemoteAccessAllowed    bool `json:"readOnlyRemoteAccessAllowed"`
}

type SettingsItems struct {
	Items      []Item `json:"items"`
	TotalCount int    `json:"totalCount"`
	PageSize   int    `json:"pageSize"`
}

type Item struct {
	ObjectId string `json:"objectId"`
	Value    Note   `json:"value"`
}

type Note struct {
	Title        string `json:"title"`
	Introduction string `json:"introduction"`
	Details      string `json:"details"`
}

type SmtpConfiguration struct {
	HostName                       string  `json:"hostName"`
	Port                           int     `json:"port"`
	UserName                       string  `json:"userName"`
	Password                       *string `json:"password"`
	IsPasswordConfigured           bool    `json:"isPasswordConfigured"`
	ConnectionSecurity             string  `json:"connectionSecurity"`
	SenderEmailAddress             string  `json:"senderEmailAddress"`
	AllowFallbackViaMissionControl bool    `json:"allowFallbackViaMissionControl"`
	UseSmtpServer                  bool    `json:"useSmtpServer"`
}
