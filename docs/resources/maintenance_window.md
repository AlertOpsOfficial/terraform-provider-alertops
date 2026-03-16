---
page_title: "alertops_maintenance_window Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps maintenance window to suppress alert notifications during planned downtime.
---

# alertops_maintenance_window

Manages an AlertOps maintenance window. Maintenance windows suppress alert notifications for specified integrations during planned maintenance, preventing false on-call pages.

-> **Note:** When `recurrence` is set to `Once`, the API ignores weekday fields and returns all weekday values as `false`. Set all weekday attributes to `false` in this case to avoid perpetual diffs.

## Example Usage

### One-time maintenance window

```terraform
resource "alertops_maintenance_window" "deployment" {
  description = "Production deployment window"
  recurrence  = "Once"
  start_date  = "2026-06-15"
  end_date    = "2026-06-15"
  start_time  = "02:00"
  end_time    = "04:00"

  # Pass integration IDs (as strings) to silence during maintenance
  integration = [tostring(alertops_inbound_integration.monitoring.id)]

  # Set all weekday fields to false for one-time windows
  sunday    = false
  monday    = false
  tuesday   = false
  wednesday = false
  thursday  = false
  friday    = false
  saturday  = false
}
```

### Weekly recurring maintenance window (Monday–Friday nights)

```terraform
resource "alertops_maintenance_window" "nightly_backup" {
  description = "Nightly backup window"
  recurrence  = "Weekly"
  start_time  = "22:00"
  end_time    = "23:00"

  integration = [tostring(alertops_inbound_integration.monitoring.id)]

  sunday    = false
  monday    = true
  tuesday   = true
  wednesday = true
  thursday  = true
  friday    = true
  saturday  = false
}
```

### Monthly maintenance window

```terraform
resource "alertops_maintenance_window" "monthly_patching" {
  description = "Monthly OS patching"
  recurrence  = "Monthly"
  start_date  = "2026-01-01"
  end_date    = "2026-12-31"
  start_time  = "01:00"
  end_time    = "05:00"

  integration = [
    tostring(alertops_inbound_integration.monitoring.id),
    tostring(alertops_inbound_integration.app_alerts.id),
  ]

  sunday    = false
  monday    = false
  tuesday   = false
  wednesday = false
  thursday  = false
  friday    = false
  saturday  = true
}
```

## Argument Reference

### Optional

- `description` (String) A human-readable description of the maintenance window.
- `recurrence` (String) The recurrence pattern. Valid values: `Once`, `Daily`, `Weekly`, `Monthly`. When `Once`, weekday fields are ignored by the API.
- `integration` (List of String) Integration IDs (as strings) to suppress during the maintenance window. Use `tostring()` to convert numeric IDs.
- `start_date` (String) Start date in `YYYY-MM-DD` format.
- `end_date` (String) End date in `YYYY-MM-DD` format.
- `start_time` (String) Start time in `HH:MM` format (24-hour).
- `end_time` (String) End time in `HH:MM` format (24-hour).
- `sunday` (Boolean) Whether Sunday is included. Defaults to `false`.
- `monday` (Boolean) Whether Monday is included. Defaults to `true`.
- `tuesday` (Boolean) Whether Tuesday is included. Defaults to `true`.
- `wednesday` (Boolean) Whether Wednesday is included. Defaults to `true`.
- `thursday` (Boolean) Whether Thursday is included. Defaults to `true`.
- `friday` (Boolean) Whether Friday is included. Defaults to `true`.
- `saturday` (Boolean) Whether Saturday is included. Defaults to `false`.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload.

## Attributes Reference

- `id` (String) The unique numeric identifier of the maintenance window, assigned by AlertOps.

## Import

Maintenance windows can be imported using their numeric AlertOps ID:

```shell
terraform import alertops_maintenance_window.example 12345
```
