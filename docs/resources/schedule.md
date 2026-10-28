---
page_title: "alertops_schedule Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps on-call schedule for a group. Schedules define when group members are on-call and can be Fixed or Rotating.
---

# alertops_schedule

Manages an AlertOps on-call schedule for a group. Schedules control when group members are on-call and can span custom weekday ranges.

-> **Note:** The group referenced by `group` must have at least one member before a schedule can be retrieved via the API after creation.

The `id` of this resource is a composite value in the format `group_id/schedule_id`. Use this composite value when importing.

## Example Usage

### Fixed weekday schedule (Mon–Fri)

```terraform
resource "alertops_schedule" "business_hours" {
  group         = alertops_group.ops_team.group_name
  schedule_name = "Business Hours"
  schedule_type = "Fixed"
  time_zone     = "(UTC-05:00) Eastern Time (US & Canada)"
  color         = "#0052CC"
  start_weekday = "Mon"
  end_weekday   = "Fri"

  weekday_mon = true
  weekday_tue = true
  weekday_wed = true
  weekday_thu = true
  weekday_fri = true
  weekday_sat = false
  weekday_sun = false
}
```

### 24/7 rotating schedule

```terraform
resource "alertops_schedule" "always_on" {
  group                      = alertops_group.ops_team.group_name
  schedule_name              = "24x7 Rotation"
  schedule_type              = "Rotating"
  time_zone                  = "(UTC) Coordinated Universal Time"
  continuous                 = true
  include_all_users_in_group = false # must be false for rotating schedules
  color                      = "#FF5733"
  start_weekday              = "Sun"
  end_weekday                = "Sat"

  weekday_sun = true
  weekday_mon = true
  weekday_tue = true
  weekday_wed = true
  weekday_thu = true
  weekday_fri = true
  weekday_sat = true

  # Rotation settings are passed through extra_attributes. Rotating schedules
  # require start/end dates (yyyy-MM-dd), a rotation frequency and the users to rotate.
  extra_attributes = {
    start_date       = jsonencode({ date = "2026-11-02", hour = 0, minute = 0 })
    end_date         = jsonencode({ date = "2027-12-31", hour = 0, minute = 0 })
    rotate_frequency = "Weekly"
    rotate_weekly = jsonencode({
      rotate_x_users        = 1
      every_x_weeks         = 1
      rotate_at_day_of_week = "Mon"
      rotate_at_time        = { hour = 9, minute = 0 }
    })
    users = jsonencode([
      { user = alertops_user.alice.user_name, role = "Primary" },
      { user = alertops_user.bob.user_name, role = "Primary" },
    ])
  }
}
```

### After-hours schedule (evenings and weekends)

```terraform
resource "alertops_schedule" "after_hours" {
  group         = alertops_group.ops_team.group_name
  schedule_name = "After Hours"
  schedule_type = "Fixed"
  time_zone     = "(UTC-05:00) Eastern Time (US & Canada)"
  color         = "#FF0000"

  weekday_sun = true
  weekday_mon = false
  weekday_tue = false
  weekday_wed = false
  weekday_thu = false
  weekday_fri = false
  weekday_sat = true
}
```

## Argument Reference

### Required

- `group` (String) The name of the group this schedule belongs to.
- `schedule_name` (String) The display name of the schedule.
- `schedule_type` (String) The schedule rotation type. Valid values: `Fixed`, `Rotating`.
- `time_zone` (String) The time zone for the schedule in AlertOps format (e.g., `(UTC-05:00) Eastern Time (US & Canada)`).

### Optional

- `continuous` (Boolean) Whether the schedule runs without a defined start/end date. Defaults to `true`.
- `include_all_users_in_group` (Boolean) Whether all users in the group are included in this schedule. Defaults to `true`.
- `enabled` (Boolean) Whether the schedule is active. Defaults to `true`.
- `color` (String) Display color for the schedule in hex format (e.g., `#FF5733`).
- `start_weekday` (String) The weekday the schedule starts. Valid values: `Sun`, `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`.
- `end_weekday` (String) The weekday the schedule ends. Valid values: `Sun`, `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`.

~> **Note:** The AlertOps API requires `start_weekday` and `end_weekday`, and derives the active days from that range: a `weekday_*` flag set to `true` outside the range is ignored by AlertOps.
- `weekday_sun` (Boolean) Whether Sunday is active. Defaults to `false`.
- `weekday_mon` (Boolean) Whether Monday is active. Defaults to `true`.
- `weekday_tue` (Boolean) Whether Tuesday is active. Defaults to `true`.
- `weekday_wed` (Boolean) Whether Wednesday is active. Defaults to `true`.
- `weekday_thu` (Boolean) Whether Thursday is active. Defaults to `true`.
- `weekday_fri` (Boolean) Whether Friday is active. Defaults to `true`.
- `weekday_sat` (Boolean) Whether Saturday is active. Defaults to `false`.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload. JSON objects and arrays are automatically parsed.

## Attributes Reference

- `id` (String) The composite identifier in the format `group_id/schedule_id`.
- `group_id` (String) The numeric ID of the group this schedule belongs to.

## Import

Schedules are imported using the composite `group_id/schedule_id` format:

```shell
terraform import alertops_schedule.example 100/200
```
