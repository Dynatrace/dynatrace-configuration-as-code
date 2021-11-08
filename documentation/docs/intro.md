---
id: intro
sidebar_position: 1
title: What is Monaco ?
slug: /
---

Monaco is CLI tool that automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.

## Why Monaco?

Monaco’s self-service model enables development teams to set up monitoring and observability easily and efficiently, even for large scale applications. It eliminates the need for building custom monitoring solutions and reduces the manual work for monitoring teams.  

## How does it work? 

Developers define a monitoring configuration file that is checked into version control alongside the application’s source code. With the next commit or pull request, the code gets built and deployed, automatically creating monitoring dashboards and alerting notifications. 

## Features

Monaco currently offers the following features:
- Creating configuration templates for reuse across multiple environments. 
- Handling interdependencies between configurations without needing to keep track of unique identifiers. 
- Applying the same configuration to hundreds of Dynatrace environments and updating them. 
- Rolling out specific configurations to specific environments. 
- Promoting application-specific configurations from one environment to another, following deployments in every stage. 
- Supporting all mechanisms and best practices of git-based workflows such as pull requests, merging, and approvals. 

## Get started

To get started, follow our [Getting Started](./Guides/Get-Started/get-started.md) guide.