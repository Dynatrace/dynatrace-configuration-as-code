# Download configuration

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco.

### steps
1. Create an environment file. 
2. Run the monaco command using the download flag ``-download``
i.e. ``./monaco -download --environments=my-environment.yaml ``

#### Options
Instead of downloading all the configurations for all the API's you can pass a list of API values separated by comma using the following flag ``-dl-specific-api``.

i.e. ``./monaco -download -dl-specific-api alerting-profiles,dashboard --environments=my-environment.yaml ``


#### Notes
You should take in consideration the following limitations of the current process.
##### Application Detection Rules:
When using download functionality you will only be able to update existing application dectection rules. If you want to create a new app detection rule you can only do so if there are no other app detection rules for that application.



