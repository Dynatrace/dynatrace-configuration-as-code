# Contributing to the Monitoring as Code Tool

- [Contributing to the Monitoring as Code Tool](#contributing-to-the-monitoring-as-code-tool)
  - [What to contribute](#what-to-contribute)
  - [How to contribute](#how-to-contribute)
  - [Code of Conduct and Shared Values](#code-of-conduct-and-shared-values)
  - [Building the Dynatrace Monitoring as Code Tool](#building-the-dynatrace-monitoring-as-code-tool)
  - [Testing the Dynatrace Monitoring as Code Tool](#testing-the-dynatrace-monitoring-as-code-tool)
    - [Integration Tests](#integration-tests)
    - [Writing Tests](#writing-tests)
  - [Checking in go mod and sum files](#checking-in-go-mod-and-sum-files)
  - [General information on code](#general-information-on-code)
    - [Test Mocks](#test-mocks)
    - [Formatting](#formatting)
  - [Pre-Commit Hook](#pre-commit-hook)
  - [A note on Dynatrace APIs](#a-note-on-dynatrace-apis)

## What to contribute

This tool was created out of the following needs but no limited to:

* Provide an easy way to deploy numerous Dynatrace monitoring configuration for several applications across different environments such as Development, Pre-production and Production environments to maintain consistency.

Thus, this tool aims to provide a way to reproducibly deploy Dynatrace monitoring configuration in a "configuration as code"-way.

As all things Dynatrace, scalability is an important requirement, both in number of configuration files and number of environments.
This is also the area currently offering the most opportunity for improvement of the tool.

## How to contribute

The easiest way to start contributing or helping with the Monitoring as Code project is to pick an existing [issue/bug](#https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/issues) and [get to work](#building-the-Dynatrace-Monitoring-as-Code-Tool).

For proposing a change, we seek to discuss potential changes in GitHub issues in advance before implementation. That will allow us to give design feedback up front and set expectations about the scope of the change, and, for larger changes, how best to approach the work such that the Monitoring as Code team can review it and merge it along with other concurrent work. This allows to be respectful of the time of community contributors.

The repo follows a rather standard branching & PR workflow.

Branches naming follows the `feature/{Issue}/{description}` or `bugfix/{Issue}/{description}` pattern.

Branches are rebased and only fast-forward merges to main permitted. No merge commits.

Commits are not auto-squashed when merging a PR, so please make sure your commits are fit to go into main (DIY squash when necessary), and write [good commit messages](https://chris.beams.io/posts/git-commit/).

## Code of Conduct and Shared Values

Before contributing please read and approve [our Code Of Conduct](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/blob/main/CODE_OF_CONDUCT.md) outlining our shared values and expectations. 

## Building the Dynatrace Monitoring as Code Tool

The `monaco` tool is written in [Go](https://golang.org/), so you will need to have [installed Go](https://golang.org/dl/) to build it.

To build the tool run `make build` in the repository root folder. 

> This guide references the make target for each step. If you want to see the actual Go commands take a look at the [Makefile](./Makefile)

To install the tool to your machine run `make install` in the repository root folder.

This will create a `monaco` executable you can use.

To build a platform specific executable run: `GOOS={OS} GOARCH={ARCH} make build`.

For example a Windows executable can be built with `GOOS=windows GOARCH=386 make build BINARY=monaco.exe`.

## Testing the Dynatrace Monitoring as Code Tool

Run the unit tests for the whole module with `make test` in the root folder.

### Integration Tests

In addition to unit tests, the module contains integration tests, that upload configuration to two test environments. Those are tagged `integration` and will be run for any pull request opened for Monitoring as Code.

To run the integration tests you will need at least one Dynatrace environment - the tests run against two configurable environments.

The following environment variables need to be defined for these test environments:
* `URL_ENVIRONMENT_1` ... URL of the first test environment
* `TOKEN_ENVIRONMENT_1` ... API token for the first test environment 
* `URL_ENVIRONMENT_2` ... URL of the second test environment
* `TOKEN_ENVIRONMENT_2` ... API token for the second test environment 

The integration tests can be run with `make integration-test`

For convenience single package tests can be run with `make test-package pkg={PACKAGE}` - e.g. `make test-package pkg=api`.

### Writing Tests

Take a look at [Go Testing](https://golang.org/pkg/testing/) for more info on testing in Go.

Tests should be written in a way that keeps them OS independent, so don't just use `/` or `\`for paths!

Instead, whenever you need to test a path, make sure to do it in one of these ways:

* Construct any paths you need using `os.PathSeparator`
* Use the public function `ReplacePathSeparators`, which replaces path separators in a given string with `os.PathSeparator`

## Checking in go mod and sum files

Go module files `go.mod` and `go.sum` are check in, in the root folder of the repo, so generally run `go` from there.

`mod` and `sum` may change on building the project. To keep those files clean on unnecessary changes, please always run `go mod tidy` before commiting changes to these files!

## General information on code

Source code of the tool is found in the `cmd/monaco` and `pkg/` folders.

### Test Mocks

Go Mockgen is used for some generated mock files. They are not generated every time, but rather checked in, so be careful
when introducing changes to objects that are mocked. You will have to regenerate and probably manually modify them to remove
e.g. the reference to the module.
​To explicitly generate the mocked files, run `make mocks` in the root folder.

### Formatting

This project uses the default go formatting tool `go fmt`.

You can use the Make target `make format` to format all files.

## Pre-Commit Hook

Before committing changes, please make sure you've added the `pre-commit` hook from the hooks folder.
On Unix you can use the `setup-git-hooks.sh` to symlink that file into your `.git/hooks` folder.

## A note on Dynatrace APIs

Some of the APIs this tool uses are 'Earlier Adopter' APIs. They may change, and we can't do anything but deal with that when it happens.

To avoid keeping two documents up to date, the volatile/EA endpoints are marked with a comment in [apis.yaml](apis.yaml).

If you add a new API please mark it correctly if it should be an 'Early  Adopter' API.

If you see that an API has been moved to a final release, please remove the respective comment.
