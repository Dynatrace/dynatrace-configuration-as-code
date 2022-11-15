---
sidebar_position: 2
title: Migrating deprecated configuration types
---

This guide shows you how to migrate deprecated configuration types.

## *dashboard*, *request-naming-service*, *app-detection-rule*

Initial *dashboard*, *request-naming-service*, *app-detection-rule* configurations were all affected by conflicts between their DT entities name attributes.

Dashboards for example (same applies to *request-naming-service*, *app-detection-rule*) don't have a unique name within a Dynatrace environment. Unfortunately, Monaco depends on name uniqueness in order to identify resources. In the case of dashboards, this resulted in missed/invalid downloads and conflicts during deployments. The solution to this was generating custom UUIDs based on Monaco configuration metadata. As many advantages this brings, the one downside is that Monaco lost track of already deployed dashboards. A dashboard deployment would therefore result in a redeployment (and duplicating - isn't it ironic) of potentially dozens of dashboards in Dynatrace.

The following guide is referencing *dashboard* configurations. However, the same applies to *request-naming-service* and *app-detection-rule* configurations.

1) Existing *dashboard* configurations usually look similar to this:

    *config.yaml*
    ```
    ---
    config:
    - DashboardConfigId: config.json

    DashboardConfigId:
    - name: Monaco Test
    - owner: Monaco User
    - isShared: true
    ```

    With *DashboardConfigId* as the user defined key that links configuration details and config.json. *name*, *owner* and *isShared* are custom properties which are subsituted in config.json:

    *config.json*
    ```
    {
      "dashboardMetadata": {
        "dashboardFilter": null,
        "name": "{{ .name }}",
        "owner": "{{ .owner }}",
        "shared": {{ .isShared }},
        "tilesNameSize": null
      },
      "tiles": [
        ...
      ]
    }
    ```

    In a folder structure similar to this:
    ```
    workdir/
      project/
        app-detection-rule/
          ...
        dashboard/
          config.json
          config.yaml
      environment.yaml
    ```

2. Recommended: Since the user defined key (*DashboardConfigId* in our example) is used to automatically generate DT entity ids in version 2, the easiest way to migrate existing configuration is to substitute it with the actual Dynatrace enitity id. Dashboard entity ids can be looked up either via API or UI:
    
    *config.yaml*
    ```
    ---
    config:
    - <DT entity UUID>: config.json

    <DT entity UUID>:
    - name: Monaco Test
    - owner: Monaco User
    - isShared: true
    ```

    The configuration is now compatible with version 2 of the dashboard configuration type.

    Alternatively: Once a configuration is deprecated and a new version provided, all subsequent downloads create configurations of the new version. Existing configuration is kept, but not updated anymore:

    ```
    workdir/
      project/
        app-detection-rule-v2/
          ...
        dashboard/
          config.json
          config.yaml
        dashboard-v2/
          config.json
          config.yaml
      environment.yaml
    ```

    Although the newly downloaded *config.yaml* includes valid configuration keys, other custom properties (e.g. owner, ...) are dropped:

    *dashboard-v2/config.yaml*
    ```
    ---
    config:
    - <DT entity UUID>: config.json

    <DT entity UUID>:
    - name: Monaco Test
    ```

    This method however allows us to identify configuration instances by their name property and copy/paste their existing DT entity ids instead of retrieving them by API or UI.

3. In order for Monaco to recognize version 2 configurations as such, the incremental version has to be appended to the config folder, *dashboard* becomes *dashboard-v2*:

    ```
    workdir/
      project/
        app-detection-rule-v2/
          ...
        dashboard-v2/
          config.json
          config.yaml
      environment.yaml
    ```
