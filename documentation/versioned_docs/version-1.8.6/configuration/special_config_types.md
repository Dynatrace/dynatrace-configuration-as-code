---
sidebar_position: 8
---

# Special Types of Configuration

Most [types of configurations](configTypes_tokenPermissions.md) behave the same and entirely as described in other sections of this documentation.

However, some have special behavior and constraints as they deal with special Dynatrace APIs - these are described in the following sections.

## Single Configuration Endpoint

These configurations are global to a Dynatrace environment and only exist once.

Unlike other configurations, there is usually some default configuration that the API - or Monaco configuration - allows to update.

Be aware that only one such configuration should be present in your Monaco configuration.

Having several configurations - for example, in several projects deployed in one run - will result in the last applied one being active on the Dynatrace environment.

## Non-Unique Name

Monaco assumes that the "Name" of a configuration is unique and will use it as the identifier when deciding to create or update a configuration.

This is also the case for most configurations when created in the Dynatrace UI or via API calls.

However, some configurations can have overlapping names, which causes issues for Monaco - for example, there can be several Dashboards named "My Dashboard".

If more than one configuration of a name is present, Monaco can not ensure the correct one is updated when searching by name.
Similar problems are present when downloading.

To work around this, special handling is present for these configuration APIS, ensuring:
* they receive a known identifier when originating from Monaco
* they are stored with their Dynatrace identifier rather than their name when downloading

As this switch from names to IDs results in re-creating configurations if they were previously created by name,
already existing configuration types like `dashboard` are retained with the previous flawed handling, while new `-v2` configuration types were added with the non-unique-name constraint/handling.

To ensure configurations are correctly updated, please see the manual steps in the [Migration Guide](/Guides/deprecated_migration.md) for how to deal with this.

> NOTE: As the `-v2` naming implies, the previous handling is deprecated and will be dropped in version 2.0.
