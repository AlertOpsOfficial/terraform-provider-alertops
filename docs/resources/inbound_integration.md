---
page_title: "alertops_inbound_integration Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps inbound integration for receiving alerts from external monitoring tools and systems.
---

# alertops_inbound_integration

Manages an AlertOps inbound integration. Inbound integrations are endpoints that receive alerts from external tools (monitoring systems, CI/CD pipelines, etc.) and route them to the appropriate on-call team via an escalation policy or recipient list.

Complex nested fields like `api_settings`, `email_settings`, and `chat_settings` can be passed as JSON-encoded strings via `extra_attributes` using Terraform's `jsonencode()` function.

## Example Usage

### Basic API integration

```terraform
resource "alertops_inbound_integration" "monitoring" {
  inbound_integration_name = "Prometheus Alerts"
  type                     = "API"
  enabled                  = true
  escalation_policy        = alertops_escalation_policy.default.escalation_policy_name
}
```

### API integration with routing and alert mapping

```terraform
resource "alertops_inbound_integration" "app_alerts" {
  inbound_integration_name = "My Application"
  type                     = "API"
  enabled                  = true
  escalation_policy        = alertops_escalation_policy.default.escalation_policy_name
  recipient_groups         = [alertops_group.ops_team.group_name]

  extra_attributes = {
    "api_settings" = jsonencode({
      is_bidirection = false
      is_secure      = false
      url_mapping = {
        method      = "POST"
        content     = "JSON"
        source      = "myapp"
        source_name = "my_application"
        static      = false
        open_alert_when = {
          field_name = "SourceStatus"
          type       = "Contains Any"
          values     = ["firing", "critical"]
        }
        close_alert_when = {
          field_name = "SourceStatus"
          type       = "Contains Any"
          values     = ["resolved", "ok"]
        }
      }
    })
  }
}
```

### Integration routed through a response play

```terraform
resource "alertops_inbound_integration" "newrelic" {
  inbound_integration_name = "New Relic"
  type                     = "API"
  response_play            = alertops_response_play.newrelic.id
}

# Pass the generated endpoint to the monitoring tool.
output "newrelic_webhook_url" {
  value     = alertops_inbound_integration.newrelic.api_url
  sensitive = true
}
```

### Email integration

```terraform
resource "alertops_inbound_integration" "email_alerts" {
  inbound_integration_name = "Email Alerts"
  type                     = "Email"
  enabled                  = true
  recipient_groups         = [alertops_group.ops_team.group_name]

  # Email integrations require a mailbox name.
  extra_attributes = {
    mail_box = "email-alerts"
  }
}
```

### Heartbeat integration (dead man's switch)

```terraform
resource "alertops_inbound_integration" "heartbeat" {
  inbound_integration_name = "Service Heartbeat"
  type                     = "Heartbeat"
  enabled                  = true
  escalation_policy        = alertops_escalation_policy.default.escalation_policy_name
}
```

## Argument Reference

### Required

- `inbound_integration_name` (String) The display name of the integration.
- `type` (String) The integration type. Common values: `API`, `Email`, `Heartbeat`, `ChatOps`.

### Optional

- `enabled` (Boolean) Whether the integration is active. Defaults to `true`.
- `sequence` (Number) The processing sequence order of the integration. Defaults to `1`.
- `escalation_policy` (String) The name of the escalation policy to use for routing alerts. Conflicts with `response_play`.
- `response_play` (String) The ID of the response play to use for routing alerts, e.g. `alertops_response_play.example.id`. Conflicts with `escalation_policy`.
- `recipient_groups` (List of String) Names of groups that will receive alerts from this integration.
- `recipient_users` (List of String) Usernames of users that will receive alerts from this integration.
- `inbound_template_id` (Number) The ID of the inbound template to apply to incoming alerts. Defaults to `0`.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload. Use `jsonencode()` for nested objects like `api_settings`, `email_settings`, `chat_settings`.

~> **Note:** AlertOps requires a URL mapping for `API` integrations. If `api_settings` is not supplied in `extra_attributes`, the provider sends the same default mapping as the AlertOps "Custom" API template (source `Custom`, open an alert for every POST).

## Attributes Reference

- `id` (String) The unique numeric identifier of the integration, assigned by AlertOps.
- `api_url` (String, Sensitive) The generated endpoint URL that monitoring tools post alerts to. It contains a secret token.
- `response_play_name` (String) The name of the response play the integration uses, if any.

## Import

Inbound integrations can be imported using their numeric AlertOps ID:

```shell
terraform import alertops_inbound_integration.example 12345
```
