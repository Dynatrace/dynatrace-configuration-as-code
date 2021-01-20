# Download configuration

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. For this feature you will have to enable CLI version 2.0.

### steps
1. Enable CLI version 2.0 by adding an environment variable call NEW_CLI with a value greater than 0.
``export NEW_CLI=1``
Create an [environment](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code#environments-file) file. 
2. Run monaco using the download command ``download``
i.e. ``./monaco download --environments=my-environment.yaml ``

#### Options
Instead of downloading all the configurations for all the API's you can pass a list of API values separated by comma using the following flag ``--downloadSpecificAPI``.

i.e. ``./monaco download --downloadSpecificAPI alerting-profiles,dashboard --environments=my-environment.yaml ``


#### Notes
You should take in consideration the following limitations of the current process.
##### Application Detection Rules:
When using download functionality you will only be able to update existing application dectection rules. If you want to create a new app detection rule you can only do so if there are no other app detection rules for that application.



