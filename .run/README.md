# What's this?

This folder contains some shared run configurations for Jetbrains products (like GoLand) to help you get started with
some common commands and tests you might want to run/debug from your IDE.

# How do I best use these?

For each run config that hits a Dynatrace environment you will need to fill in the URL and TOKEN in the environment
variables.

To not accidentally check in any modifications copy the files out of the shared/ folder into the .run/ folder -
modifications should then apply to the upper level configuration files and not change those in shared/.

Please be mindful about checking in any changes to run configurations - by default this folder is .gitignored anyway -
this is really just meant as a shared basis.

# What configurations are there?

## Run Configs - For Debugging

Generally using the integration-all-configs/ test resources the most common commands have a run config to get you
started developing/debugging.

* Deploy
    * deploys integration-all-configs/ test resources to the configured environment
* Deploy (dry-run)
    * validates integration-all-configs/ test resources
* Download
    * downloads all configs the configured environment
    * NOTE: you need to modify the program arguments to contain your environment url as well as the environment vars
* Convert
    * converts v1 integration-all-configs/ test resources into v2

## Test Configs

Configurations with the correct flags - and env variables to fill out - to run all or specific end-to-end and unit
tests.

* Integration Tests - all
    * run v1 and v2 integration tests
* Integration Tests - v1
    * run v1 integration tests
* Integration Tests - v2
    * run v2 integration tests
* Integration Tests - download/restore
    * run the download restore integration test
* Integration Tests - cleanup
    * remove test configurations from the given environments
    * NOTE: be mindful that this might impact CI and other people if using a shared Dynatrace environment
* Unit Tests
    * run all unit tests
