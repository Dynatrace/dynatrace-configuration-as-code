---
sidebar_position: 2
---

# Deployment Manifest

In order for monaco to know what to deploy, there has to be a manifest file present.
This file provides details on what to deploy and where to deploy it.

## Structure

The manifest is written in yaml syntax. It has two top level keys `projects` and `environments`.

### Projects

All entries under the `projects` top level key specify projects to deploy by monaco. To specify
the type of a project, one has to provide the `type` key in the project item.

There are currently two types of projects:
- simple
- grouping

#### Simple Projects
This is the default type. All you need to provide is a `name` and `path` property.
If no `path` property is provided, the name will be used as `path`.

**Note**: It is not allowed for the name to contain either `/` nor `\`. This decision
was made to explicitly distinquish it from filesystem paths.

**Note**: Paths should always use `/` as separator, no matter what OS you use (Linux, Windows, Mac)!

E.g.
```yaml
projects:
- name: infra
  path: shared/infrastructure
```

#### Grouping Projects
Grouping projects offer a simplified way of grouping multiple projects together.
The difference to a simple project is, that a grouping project will load all sub-folders of a given path
as simple projects. You have to specify a name, which will then be used as a prefix for
the resulting simple projects. As separator `.` will be used.

E.g.
Given the following file structure:
- general
 - infrastructure
 - zones

The following project definition:
```yaml
projects:
- name: general
  path: general
  type: grouping
```
will yield two projects:
- general.infrastructure
- general.zones

### Environments

If projects are the what, environments are the where configuration for monaco.
Here a quick example of how it looks like:

```yaml
environments:
- group: dev
  entries:
  - name: test-env-1
    url: https://aaa.bbb.cc
    token:
      name: TEST_ENV_TOKEN

  - name: test-env-2
    url: https://ddd.bbb.cc
    token:
      name: TEST_ENV_2_TOKEN

- group: prod
  entires:
  - name: prod-env-1
    url: https://prod.env.cc
    token:
      name: PROD_TOKEN
```

As you can see, every environment has to be part of a group and can only be present
in one group.

An environment configuration consists  of three parts:
- name
- url
- token

The name has to be unique. The token configuration specifies a name of the environment
variable from where monaco will load the access token for the dynatrace api.
