---
sidebar_position: 1
title: Add a new API
---
​
This guide shows you how to add a new API to Monaco that is not included in the [table of supported APIs](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code#configuration-types--apis) and how to determine whether an API is easy to add. 
​
> :warning: Adding APIs to Monaco is straightforward in most cases. However, some APIs require more coding.
​

## Determine if an API is easy to add
​
Easy-to-add APIs fulfill the following criteria: 
​
* Configuration APIs that implement the following HTTP methods: 
  * `GET <my-environment>/api/config/v1/<my-config>` (get all configs)
  * `GET <my-environment>/api/config/v1/<my-config>/<id>` (get a single config)
  * `POST <my-environment>/api/config/v1/<my-config>` (create a new config)
  * `PUT <my-environment>/api/config/v1/<my-config>/<id>` (change an existing config)
  * `DELETE <my-environment>/api/config/v1/<my-config>/<id>` (delete a config)
​

* The model of the configuration has a `name` property: 
 
```json
{
      "id": "acbed0c4-4ef1-4303-991f-102510a69322",
      "name: "my-name"
      ...
}
```
​

* The `GET (all)` REST call return `id` and `name`:
​

```json
{
    "values": [
      {
        "id": "string",
        "name": "string"
      }
    ]
}
```

​
If your API fulfills these three criteria, perform the steps in the following section to add it to Monaco.
​
> :warning: If your API does not fulfil these requirements, please open a ticket in Monaco's backlog
to get implementation feedback from the maintainers.
​
​

## Add a new API to Monaco
​
Take the following steps to add a new API to Monaco.

1. Open your preferred CLI and enter the following code to add your API to [the map in api.go](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/blob/main/pkg/api/api.go#L25) and replace the placeholder values as described below. 
​

```json
  "<my-api-folder-name>": {
      apiPath: "<path-to-my-api>",                       // mandatory
      propertyNameOfGetAllResponse: "<property-name>",   // not necessary in case of "values"
  },
```
​

| Placeholder     | Description | 
| ----------- | ----------- | 
| <nobr>`<my-api-folder-name>`</nobr> | The name of the API, also used for the folder name for the configurations. Study the existing API names to get a feeling for the naming conventions and choose one accordingly.|
| <nobr>`<path-to-my-api>`</nobr> | Path for your API. Monaco prefixes it with the environment URL to access the configs of your API. |
| <nobr>`<property-name>`</nobr> | Name of the json property used in the `GET ALL` REST call to return the list of configs. E.g. it would be `extensions`, if the response of your PI's `GET ALL` REST call looks like the snippet below|
​
  
```json
    {
      "extensions": [
        {
          "id": "custom.python.connectionpool",
          "name": "Connection Pool",
          "type": "ONEAGENT"
        }
      ],
        "totalResults": 9,
        "nextPageToken": "LlUdYmu5S2MfX/ppfCInR9M="
      }
```

​
2. Add a sample config for the integration tests in [cmd/monaco/test-resources/integration-all-configs](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/tree/main/cmd/monaco/test-resources/integration-all-configs)

​
3. Add your API to the [table of supported APIs](../configuration/configTypes_tokenPermissions).
​
> :rocket: After performing these steps, please create the pull request in the upstream repository to share it with the community!
