---
page_title: "alertops_service Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps service for tracking uptime and grouping related components.
---

# alertops_service

Manages an AlertOps service. Services represent your applications or infrastructure components, enabling uptime tracking and organized alert routing.

Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.

## Example Usage

### Simple service with uptime tracking

```terraform
resource "alertops_service" "api_service" {
  service_name = "Production API"
  show_uptime  = true
}
```

### Service with components and message templates

```terraform
resource "alertops_service" "platform" {
  service_name = "Platform Services"
  show_uptime  = true

  components = [
    {
      component_name = "API Gateway"
      external_id    = "api-gateway-prod"
    },
    {
      component_name = "Database Cluster"
      external_id    = "db-cluster-prod"
    },
    {
      component_name = "Cache Layer"
      external_id    = "redis-prod"
    }
  ]

  templates = [
    {
      template_name = "Incident Open"
      subject       = "[ALERT] {{service_name}} degraded"
      message       = "Service {{service_name}} is experiencing issues. Alert ID: {{alert_id}}"
    },
    {
      template_name = "Incident Resolved"
      subject       = "[RESOLVED] {{service_name}} restored"
      message       = "Service {{service_name}} has been restored."
    }
  ]
}
```

## Argument Reference

### Required

- `show_uptime` (Boolean) Whether to display uptime metrics for this service.

### Optional

- `service_name` (String) The display name of the service.
- `components` (List of Object) Components that make up this service. See [components](#nestedblock--components) below.
- `templates` (List of Object) Custom message templates for alerts related to this service. See [templates](#nestedblock--templates) below.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload.

### `components`

- `component_name` (String, Optional) The display name of the component.
- `external_id` (String, Optional) An external identifier for the component, for correlation with external monitoring systems.

### `templates`

- `template_name` (String, Optional) The name of the message template.
- `subject` (String, Optional) The subject line for notifications using this template. Supports AlertOps template variables.
- `message` (String, Optional) The message body for notifications using this template. Supports AlertOps template variables.

## Attributes Reference

- `id` (String) The unique numeric identifier of the service, assigned by AlertOps.

## Import

Services can be imported using their numeric AlertOps ID:

```shell
terraform import alertops_service.example 12345
```
