# Terraform Provider for AlertOps

The AlertOps Terraform provider allows you to manage your [AlertOps](https://alertops.com) resources using infrastructure-as-code. It supports full CRUD lifecycle management for users, groups, on-call schedules, escalation policies, integrations, services, and maintenance windows.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build from source)
- An AlertOps account with API access

## Using the Provider

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
  # API key can also be set via the ALERTOPS_API_KEY environment variable
  api_key = var.alertops_api_key
}
```

See the [provider documentation](https://registry.terraform.io/providers/kamsrikanth/alertops/latest/docs) for full usage details.

## Authentication

The provider requires an AlertOps API key. You can provide it in two ways:

**Environment variable (recommended):**
```shell
export ALERTOPS_API_KEY="your-api-key"
```

**Provider configuration:**
```terraform
provider "alertops" {
  api_key = "your-api-key"
}
```

## Resources

| Resource | Description |
|---|---|
| `alertops_user` | Manage AlertOps users with contact methods |
| `alertops_group` | Manage on-call groups and their members |
| `alertops_schedule` | Manage on-call schedules for a group |
| `alertops_escalation_policy` | Manage escalation policies and notification chains |
| `alertops_inbound_integration` | Manage inbound integrations (API, Email, Heartbeat, etc.) |
| `alertops_outbound_integration` | Manage outbound webhook integrations |
| `alertops_service` | Manage services with uptime tracking |
| `alertops_maintenance_window` | Manage maintenance windows to suppress alerts |

## Quick Start Example

```terraform
# Create a user
resource "alertops_user" "oncall" {
  user_name  = "jsmith"
  first_name = "Jane"
  last_name  = "Smith"
  locale     = "en-US"
  time_zone  = "(UTC-05:00) Eastern Time (US & Canada)"
  type       = "Standard"
  roles      = ["App Admin"]

  contact_methods = [
    {
      contact_method_name   = "Email-Official"
      email_address         = "jsmith@example.com"
      notification_time24x7 = true
      enabled               = true
      sequence              = 1
    }
  ]
}

# Create an on-call group
resource "alertops_group" "ops_team" {
  group_name = "ops-team"

  members = [
    {
      member_type = "User"
      member      = alertops_user.oncall.user_name
      sequence    = 1
      roles       = ["Primary"]
    }
  ]
}

# Create a schedule
resource "alertops_schedule" "business_hours" {
  group         = alertops_group.ops_team.group_name
  schedule_name = "Business Hours"
  schedule_type = "Fixed"
  time_zone     = "(UTC-05:00) Eastern Time (US & Canada)"
  weekday_mon   = true
  weekday_tue   = true
  weekday_wed   = true
  weekday_thu   = true
  weekday_fri   = true
  weekday_sat   = false
  weekday_sun   = false
}

# Create an escalation policy
resource "alertops_escalation_policy" "default" {
  escalation_policy_name = "Default Policy"
  priority               = "High"
}

# Create an API inbound integration
resource "alertops_inbound_integration" "my_app" {
  inbound_integration_name = "My Application"
  type                     = "API"
  escalation_policy        = alertops_escalation_policy.default.escalation_policy_name
  recipient_groups         = [alertops_group.ops_team.group_name]
}
```

## Building From Source

```shell
git clone https://github.com/alertops/terraform-provider-alertops
cd terraform-provider-alertops
go build ./...
```

To install locally for testing:

```shell
go install ./...
```

## Developing the Provider

To run acceptance tests (requires a real AlertOps API key):

```shell
export ALERTOPS_API_KEY="your-api-key"
make testacc
```

To regenerate documentation:

```shell
make generate
```

## License

See [LICENSE](LICENSE) for details.
