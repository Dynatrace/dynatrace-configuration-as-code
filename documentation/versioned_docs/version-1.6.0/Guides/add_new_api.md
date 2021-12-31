---
sidebar_position: 1
---

# How to add a new API

You spotted a new API which you want to automate using `monaco`, but sadly it's not in the 
[table of supported APIs](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code#configuration-types--apis)?

Usually, the addition of new APIs to `monaco` is straightforward and requires little programming 
experience. Only some APIs require you to do more coding. There are certain criteria for differentiating the two cases.

## Easy-to-add API Characteristics
Easy-to-add APIs have these characteristics:

* It implements the following HTTP methods. E.g for configuration APIs that is: 
  * `GET <my-environment>/api/config/v1/<my-config>` (get all configs)
  * `GET <my-environment>/api/config/v1/<my-config>/<id>` (get a single config)
  * `POST <my-environment>/api/config/v1/<my-config>` (create a new config)
  * `PUT <my-environment>/api/config/v1/<my-config>/<id>` (change an existing config)
  * `DELETE <my-environment>/api/config/v1/<my-config>/<id>` (delete a config)

* The model of the configuration has a `name` property: 
 
```json
{
      "id": "acbed0c4-4ef1-4303-991f-102510a69322",
      "name: "my-name"
      ...
}
```

* The `GET (all)` REST call return `id` and `name`:

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

If your API supports these 3 characteristics, you just need to perform the steps in the following section to add it.

However, if your API does not fulfil the above requirements, please open a ticket in `monaco`'s backlog
to get implementation feedback from the maintainers.

## Steps to add an API

* Add your API to [the map in api.go](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/blob/main/pkg/api/api.go#L25):

```json
  "<my-api-folder-name>": {
      apiPath: "<path-to-my-api>",                       // mandatory
      propertyNameOfGetAllResponse: "<property-name>",   // not necessary in case of "values"
  },
```

* Fill the 4 placeholder values from above:
  * `<my-api-folder-name>`: This is the name of the API, which is also used for the folder name
  you need to place your configurations in. Please take a look at the existing API names to get a
  feeling for the naming conventions and choose it accordingly.
  * `<path-to-my-api>`: This path points to your API. `monaco` prefixes it with the environment
  URL to access the configs of your API.
  * `<property-name>`: This names the json property used in the `GET ALL` REST call to
  return the list of configs. E.g. it would be `extensions`, if the response of your API's 
  `GET ALL` REST call looks like this:
  
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

* Add a sample config for the integration tests in [cmd/monaco/test-resources/integration-all-configs](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/tree/main/cmd/monaco/test-resources/integration-all-configs)
* Add your API to the [table of supported APIs](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code#configuration-types--apis).

After performing these steps, please create the pull request in the upstream repository.
Other users of `monaco` will thank you! :rocket:
