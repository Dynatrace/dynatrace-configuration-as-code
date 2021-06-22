---
id: intro
sidebar_position: 1
title: What is Monaco?
slug: /
---

Monaco is CLI tool that tool automates deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.

## Why monaco?

Configuring monitoring and observability be both hard and time consuming to do at scale. Monaco enables Application Teams through self-service capabilities to setup and configure Monitoring and Alerting without causing manual work on the team responsible for monitoring.

With monaco, defining what to monitor and what to be alerted on is easy for developers as checking in a monitoring configuration file into version control along with the applications source code. With the next commit or Pull Request the code gets built, deployed and the automatically get the monitoring dashboards and alerting notifications. This self-service model will ensure teams can focus more time on building business services. Monaco eliminates the need of  building a custom monitoring solution that fits into a team's development process and mindset.

## Features

- Templatize configuration for reusability across multiple environments
- Handle Interdependencies between configurations without keeping track of unique identifiers
- Introducing the capability to easily apply – and update – the same configuration to hundreds of Dynatrace environments as well as being able to roll out to specific environments
- Provides an easy way to promote application specific configurations from one environment to another – following their deployments from development, to hardening to production
- Supports all the mechanisms and best-practices of git-based workflows such as pull requests, merging and approvals
- Allows to easily promote configuration from one environment to another following their deployment from development to hardening to production

To get started, install the tool:

[Installation](./installation.md)
