# Contributing to the Configuration as Code Tool `Monaco`

Thank you for considering contributing to Monaco.
We welcome contributions from the community to help improve and expand Monaco's capabilities.
Please follow the guidelines below to ensure a smooth contribution process.

- [What to contribute](#what-to-contribute)
- [How to contribute](#how-to-contribute)
  - [Commit structure](#Commits)
- [Code of Conduct and Shared Values](#code-of-conduct-and-shared-values)
- [Building the Dynatrace Configuration as Code Tool](#building-the-dynatrace-configuration-as-code-tool)
- [Testing the Dynatrace Configuration as Code Tool](#testing-the-dynatrace-configuration-as-code-tool)
  - [Integration Tests](#integration-tests)
  - [Writing Tests](#writing-tests)
- [Checking in go mod and sum files](#checking-in-go-mod-and-sum-files)
- [General information on code](#general-information-on-code)
  - [Test Mocks](#test-mocks)
  - [Formatting](#formatting)
- [Pre-Commit Hook](#pre-commit-hook)
- [A note on Dynatrace APIs](#a-note-on-dynatrace-apis)

## What to contribute

Monaco was created to easily manage Dynatrace configurations at scale across different Dynatrace environments and accounts.
Thus, this tool aims to provide a way to reproducibly deploy Dynatrace configuration in a "configuration as code"-way.

## How to contribute

The easiest way to start contributing or helping with the Configuration as Code project is to pick an existing [issue/bug](https://github.com/dynatrace/dynatrace-configuration-as-code/issues) and [get to work](#building-the-Dynatrace-Configuration-as-Code-Tool).

For proposing a change, we'd like to discuss potential changes in GitHub issues before implementation.
That will allow us to give design feedback up front and set expectations about the scope of the change and, for more significant changes, 
how best to approach the work such that the Configuration as Code team can review it and merge it with other concurrent work. 
This allows being respectful of the time of community contributors.

The repo follows a relatively standard branching & PR workflow.

Branches naming follows the `feature/{Issue}/{description}` or `bugfix/{Issue}/{description}` pattern. \
Branches are rebased, and only fast-forward merges to main are permitted. \
No merge commits.

By default, commits are not auto-squashed when merging a PR, so please ensure your commits are fit to go into main.

For convenience auto-squashing all PR commits into a single one is an optional merge strategy - but we strive for [atomic commits](https://www.freshconsulting.com/insights/blog/atomic-commits/)
with [good commit messages](https://cbea.ms/git-commit/) in main, so not auto-squashing is recommended.

### Commits

Each commit must conform to a [Conventional Commit], with a [good commit message] and [atomic commits].

[Conventional Commit]: https://www.conventionalcommits.org/
[good commit message]: https://cbea.ms/git-commit/
[atomic commits]: https://dev.to/samuelfaure/how-atomic-git-commits-dramatically-increased-my-productivity-and-will-increase-yours-too-4a84

#### Conventional commits - prefixes

* Production code
  * `feat`: New code has been written to support a new feature
  * `fix`: Bug fix of production code (not in build scripts)
  * `refactor`: Refactor production code
  * `test`: Add missing tests, refactor tests, ... 
* Non-production code
  * `ci`: Build system changes (workflows, linting, ...) 
  * `chore`: Updating non-production code (that is not `ci:`) 
  * `docs`: Changes to the documentation (not GoDoc documentation)

#### Examples

**New feature change**
``` 
feat: Add support for federated attribute values in account groups

This change adds support for `federatedAttributeValues` to account groups.
This allows Monaco to deploy groups with owner `SAML`.
```

Bug Fix Changes
```
fix: Very important feature misbehaved

Instead of thing A, B happened. This change fixes this behavior by introducing C and checking for D.
```

More examples can be found [here](https://www.conventionalcommits.org/en/v1.0.0/#examples)


## Code of Conduct and Shared Values

Before contributing, please read and approve [our Code Of Conduct](https://github.com/dynatrace/dynatrace-configuration-as-code/blob/main/CODE_OF_CONDUCT.md) outlining our shared values and expectations. 

## Building the Dynatrace Configuration as Code Tool

The `monaco` tool is written in [Go](https://golang.org/), so you'll need to have [installed Go](https://golang.org/dl/) to build it.  

To build the tool, run `make build` in the repository root folder.

**_NOTE:_**  `$GOPATH/bin` is required to be loaded in your `$PATH`

> This guide references the make target for each step. If you want to see the actual Go commands take a look at the [Makefile](./Makefile)

To install the tool to your machine, run `make install` in the repository root folder.

This will create a `monaco` executable you can use.

To build a platform-specific executable, run: `GOOS={OS} GOARCH={ARCH} make build`.

For example, a Windows executable can be built with `GOOS=windows GOARCH=386 make build BINARY=monaco.exe`.

## Testing the Dynatrace Configuration as Code Tool

Run the unit tests for the whole module with `make test` in the root folder.

For convenience, single package tests can be run with `make test-package pkg={PACKAGE}` - e.g. `make test-package pkg=api`.

### Integration Tests

In addition to unit tests, the module contains integration tests that upload configuration to two test environments. Those are tagged `integration` and will be run for any pull request opened for Monitoring as Code.

To run the integration tests, you'll need at least one Dynatrace environment - the tests run against two configurable environments.

Define the following environment variables to test for these environments:
* `URL_ENVIRONMENT_1` ... URL of the first test environment
* `TOKEN_ENVIRONMENT_1` ... API token for the first test environment
* `URL_ENVIRONMENT_2` ... URL of the second test environment
* `TOKEN_ENVIRONMENT_2` ... API token for the second test environment

Run the integration tests using `make integration-test`.

### Writing Tests

Take a look at [Go Testing](https://golang.org/pkg/testing/) for more info on testing in Go.

Tests should be written in a way that keeps them OS-independent, so don't just use `/` or `\`for paths!

Instead, whenever you need to test a path, make sure to do it in one of these ways:

* Construct any paths you need using `os.PathSeparator`
* Use the public function `ReplacePathSeparators`, which replaces path separators in a given string with `os.PathSeparator`

We use [github.com/stretchr/testify](github.com/stretchr/testify) as our testing library.

> You might still find `gotest.tools` used for asserts in a few places, as it's being replaced. If you change a test file using it, replace it.
 
We use `require` for asserting test requirements after which it makes no sense to continue - e.g. no error was returned, a slice has the expected length, pointers aren't nil, etc. - as it will fail the test immediately and exit. 

We use `assert` for asserting results after which we still want to know about more results - e.g. checking the values of several struct fields, checking the complete contents of a slice, etc.

If in doubt, use `require` to avoid follow-on errors and panics if data was already invalid.

## Checking in go mod and sum files

Go module files `go.mod` and `go.sum` are checked-in in the root folder of the repo, so generally, run `go` from there.

`mod` and `sum` may change while building the project.
To keep those files clean of unnecessary changes, please always run `go mod tidy` before committing changes to these files!

## General information on code

You can find the source code of the tool in the `cmd/monaco` and `pkg/` folders.

### Test Mocks

Go Mockgen is used for some generated mock files.
You'll have to generate them.
To explicitly generate the mocked files, run `make mocks` in the root folder.

### Formatting

This project uses the default go formatting tool `go fmt`.

To format all files, you can use the Make target `make format`.

## Pre-Commit Hook

Before committing changes, please make sure you've added the `pre-commit` hook from the hooks folder.
On Unix, you can use the `setup-git-hooks.sh` to symlink that file into your `.git/hooks` folder.

## A note on Dynatrace APIs

Some APIs this tool uses are 'Earlier Adopter' APIs. They may change, and we can't do anything but deal with that when it happens.

If you [add a new API](./New_API.md), please mark it correctly if it should be an 'Early  Adopter' API.

If you see that an API has been moved to a final release, please remove the respective comment.
