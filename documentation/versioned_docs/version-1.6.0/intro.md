---
id: intro
sidebar_position: 1
title: What is Monaco?
slug: /
---

# Welcome to the Documentation for Monaco 1.6.x

:::caution
**Monaco 1.x is deprecated by Monaco 2.x since April 2023.**

**This documentation only exists for reference purposes. We strongly suggest you upgrade!**

Take a look at [the current documentation](https://docs.dynatrace.com/docs/shortlink/configuration-as-code)!

Or jump directly into [the step-by-step migration guide](https://docs.dynatrace.com/docs/shortlink/configuration-as-code-migrate-to-v2)
:::

Monaco is CLI tool that automates deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.

## Why monaco?

Configuring monitoring and observability is both hard and time consuming to do at scale. Monaco provides self-service capabilities
tht enable Application teams to set up and configure Monitoring and Alerting without causing manual work for the team responsible for monitoring.

With Monaco, developers can define what to monitor and what to be alerted on by merely checking a monitoring configuration file into version control
along with the application source code.
With the next commit or Pull Request, the code gets built and deployed and the developers automatically get the monitoring dashboards and alerting notifications. 
This self-service model ensures that teams can focus more time on building business services.
Monaco eliminates the need to build a custom monitoring solution that fits into a team's development process and mindset.

## Features

- Templatize configuration for reusability across multiple environments
- Handle Interdependencies between configurations without tracking unique identifiers
- The same configuration can easily be applied (and updated) to hundreds of Dynatrace environments, or can be rolled out only to specific environments
- Provides an easy way to promote application specific configurations from one environment to another, following their deployments from development to hardening to production
- Supports all the mechanisms and best-practices of Git-based workflows such as pull requests, approvals, and merging
- Allows configurations to be easily promoted from one environment to another following their deployment from development to hardening to production

To get started, install the tool:

[Installation](./installation.md)
