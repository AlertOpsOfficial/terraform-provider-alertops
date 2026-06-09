---
page_title: "alertops_outbound_integration Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps outbound integration for sending alert notifications to external systems via webhooks.
---

# alertops_outbound_integration

Manages an AlertOps outbound integration. Outbound integrations send alert notifications to external systems (webhooks, REST endpoints, ticketing systems) when alerts are created or updated in AlertOps.

Complex nested fields like `security` (credentials) and `methods` (webhook definitions) can be passed as JSON-encoded strings via `extra_attributes` using Terraform's `jsonencode()` function.

## Example Usage

### Webhook with Basic authentication

```terraform
resource "alertops_outbound_integration" "webhook" {
  outbound_integration_name = "ServiceNow Sync"
  security_type             = "Basic"

  extra_attributes = {
    "security" = jsonencode({
      user_name = "alertops-user@example.com"
      password  = var.webhook_password
    })
    "methods" = jsonencode([
      {
        method_name         = "POST Alert"
        alert_type          = "Standard Alert"
        update_alert_fields = false
        method_type = {
          type               = "REST"
          web_method         = "POST"
          uri                = "https://events.servicenow.com/v2/enqueue"
          request_type       = "JSON"
          response_data_type = "JSON"
          request_data       = jsonencode({
            routing_key  = "YOUR_ROUTING_KEY"
            event_action = "trigger"
          })
          response_data = ""
          headers       = jsonencode({
            "Content-Type" = "application/json"
          })
        }
      }
    ])
  }
}
```

### Webhook with API key authentication

```terraform
resource "alertops_outbound_integration" "jira_webhook" {
  outbound_integration_name = "Jira Ticket Creator"
  security_type             = "ApiKey"

  extra_attributes = {
    "security" = jsonencode({
      api_key_name  = "Authorization"
      api_key_value = "Bearer ${var.jira_api_token}"
    })
    "methods" = jsonencode([
      {
        method_name         = "Create Issue"
        alert_type          = "Standard Alert"
        update_alert_fields = true
        method_type = {
          type               = "REST"
          web_method         = "POST"
          uri                = "https://yourorg.atlassian.net/rest/api/3/issue"
          request_type       = "JSON"
          response_data_type = "JSON"
          request_data       = jsonencode({
            fields = {
              project   = { key = "OPS" }
              summary   = "{{alert_summary}}"
              issuetype = { name = "Bug" }
            }
          })
          response_data = ""
          headers       = jsonencode({})
        }
      }
    ])
  }
}
```

### Simple webhook with no authentication

```terraform
resource "alertops_outbound_integration" "slack_webhook" {
  outbound_integration_name = "Slack Notifications"
  security_type             = "None"

  extra_attributes = {
    "methods" = jsonencode([
      {
        method_name         = "Post to Slack"
        alert_type          = "Standard Alert"
        update_alert_fields = false
        method_type = {
          type               = "REST"
          web_method         = "POST"
          uri                = "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
          request_type       = "JSON"
          response_data_type = "JSON"
          request_data       = jsonencode({
            text = "Alert: {{alert_summary}}"
          })
          response_data = ""
          headers       = jsonencode({})
        }
      }
    ])
  }
}
```

## Argument Reference

### Required

- `outbound_integration_name` (String) The display name of the outbound integration.

### Optional

- `security_type` (String) The authentication type. Valid values: `None`, `Basic`, `ApiKey`.
- `interval_in_sec` (Number) The polling interval in seconds (for pull-based integrations). Defaults to `0`.
- `enabled` (Boolean) Whether the integration is active.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload. Use `jsonencode()` for nested objects like `security` (credentials) and `methods` (webhook definitions).

## Attributes Reference

- `id` (String) The unique numeric identifier of the integration, assigned by AlertOps.

## Import

Outbound integrations can be imported using their numeric AlertOps ID:

```shell
terraform import alertops_outbound_integration.example 12345
```
