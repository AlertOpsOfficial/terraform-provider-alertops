---
page_title: "Provider: AlertOps"
description: |-
  The AlertOps provider allows you to manage AlertOps resources — users, groups, on-call schedules, escalation policies, integrations, services, and maintenance windows — using Terraform.
---

# AlertOps Provider

The AlertOps provider enables infrastructure-as-code management of your [AlertOps](https://alertops.com) account. Use it to automate the lifecycle of on-call users, groups, schedules, escalation policies, integrations, services, and maintenance windows.

## Example Usage

```terraform
terraform {
  required_providers {
    alertops = {
      source  = "kamsrikanth/alertops"
      version = "~> 2.0"
    }
  }
}

provider "alertops" {
  # Best practice: set via ALERTOPS_API_KEY environment variable
  # api_key = "your-api-key"
}
```

## Authentication

The provider requires an AlertOps API key. The following methods are supported, in order of precedence:

### Environment variable (recommended)

```shell
export ALERTOPS_API_KEY="your-api-key"
terraform plan
```

### Static credentials

```terraform
provider "alertops" {
  api_key = "your-api-key"
}
```

~> **Warning:** Hard-coding credentials in Terraform configuration is not recommended. Use the environment variable method or a secrets manager to keep credentials out of version control.

## Schema

### Optional

- `api_key` (String, Sensitive) AlertOps API key. Can also be set via the `ALERTOPS_API_KEY` environment variable.
