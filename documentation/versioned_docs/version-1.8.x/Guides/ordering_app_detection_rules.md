---
sidebar_position: 4
title: How to ensure ordering of application detection rules via monaco
---

While the UI allows you to define the ordering of Application Detection Rules, it is not possible to define an order via monaco or the API.
(Using the API directly, you *can* control if rules are appended or pre-pended but not in the exact order. By using monaco, this control is not possible)

However, you can use monaco's handling of dependencies between configurations to enforce the ordering of rules.

# Defining rule ordering using dependencies

By creating "fake" dependencies between rules, monaco will ensure that a rule will be created before another one that depends on it.

This workaround only works if all the rules are created from monaco and don't already exist.

> If rules already exist, you can manually re-order them, with updates from future configuration deployments not impacting the order.

As newly added rules will be pre-pended to existing ones, you will likely need to define dependencies in the opposite order you expect.

The following sample details how to achieve an order using dependencies:

```yaml
config:
- det3: "det.json"
- det2: "det.json"
- det1: "det.json"

det3:
- name: "App-Detection Rule 3"
- application_id: "project/application-web/application.id"
- order: "project/app-detection-rule/det2.name"

det2:
- name: "App-Detection Rule 2"
- application_id: "project/application-web/application.id"
- order: "project/app-detection-rule/det1.name"

det1:
- name: "App-Detection Rule 1"
- application_id: "project/application-web/application.id"
```

As you can see, `Rule 3` depends on `Rule 2`, and `Rule 2` depends on `Rule 1`.

With this, monaco will ensure that the rules are created in the order 1, 2, 3.

As mentioned above, the API pre-pends new rules, so the rules will be applied as 3,2,1.

Flipping the dependencies will result in a 1,2,3 ordering of the created rules:

```yaml
config:
- det3: "det.json"
- det2: "det.json"
- det1: "det.json"

det3:
- name: "App-Detection Rule 3"
- application_id: "project/application-web/application.id"

det2:
- name: "App-Detection Rule 2"
- application_id: "project/application-web/application.id"
- order: "project/app-detection-rule/det3.name"

det1:
- name: "App-Detection Rule 1"
- application_id: "project/application-web/application.id"
- order: "project/app-detection-rule/det2.name"
```
