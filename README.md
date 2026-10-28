# Terraform Provider for AlertOps

The AlertOps Terraform provider allows you to manage your [AlertOps](https://alertops.com) resources using infrastructure-as-code. It supports full CRUD lifecycle management for users, groups, on-call schedules, escalation policies, integrations, services, and maintenance windows.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.25.8 (to build from source)
- An AlertOps account with API access

## Using the Provider

```terraform
terraform {
  required_providers {
    alertops = {
      source  = "alertops/alertops"
      version = "~> 2.0"
    }
  }
}

provider "alertops" {
  # API key can also be set via the ALERTOPS_API_KEY environment variable
  api_key = var.alertops_api_key
}
```

See the [provider documentation](https://registry.terraform.io/providers/alertops/alertops/latest/docs) for full usage details.

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
| `alertops_response_play` | Manage response plays: escalation paths of users and groups with wait times |
| `alertops_workflow` | Manage workflows (conditions and actions, e.g. notify on acknowledge/close) |
| `alertops_topic` | Manage topics and the groups subscribed to them |

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

### Testing a local build with Terraform

Build the binary for the platform of your Terraform CLI (`terraform version` shows it, e.g. `windows_386` or `windows_amd64`):

```shell
GOOS=windows GOARCH=386 go build -o bin/terraform-provider-alertops.exe .
```

Point Terraform at it with a CLI config file (e.g. `dev.tfrc`) and `TF_CLI_CONFIG_FILE`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/alertops/alertops" = "C:/path/to/terraform-provider-alertops/bin"
  }
  direct {}
}
```

```shell
export TF_CLI_CONFIG_FILE=dev.tfrc
export ALERTOPS_API_KEY="your-api-key"
```

`terraform init` is not needed with `dev_overrides`.

## Testing

Unit tests run offline, without an API key:

```shell
go test ./...
```

Live end-to-end suites are in `examples/`. They create real objects (all named `TF Test ...`, integrations disabled, contact details on reserved `example.com` addresses), so run them against a test account:

| Suite | What it covers |
|---|---|
| `examples/regression` | Every resource type and the complex arguments. `terraform apply`, then `terraform apply -var revision=2` to exercise adds/removes/changes of nested items, then `terraform destroy -var revision=2`. After every apply, `terraform plan` must report "No changes". |
| `examples/full-stack` | A simple user → group → schedule → escalation policy → response play → integrations stack, with `-var revision=2` updates. |
| `examples/tf-test` | Response play, workflow, topic and escalation policy wiring using existing users/groups (`-var user_id=... -var group_id=... -var group_name=...`). |

The regression suite skips `alertops_service` (needs the Business Services plan) and `alertops_outbound_integration` by default; enable them with `-var test_services=true` / `-var test_outbound=true`.

Because the API allows only 15 requests per minute, a full regression run takes 10–15 minutes.

## Developing the Provider

The resource documentation in `docs/` is maintained by hand. Do not run `make generate` (tfplugindocs) without first adding templates under `templates/`, or it will overwrite the hand-written examples.

## License

Mozilla Public License 2.0 — see [LICENSE](LICENSE).
