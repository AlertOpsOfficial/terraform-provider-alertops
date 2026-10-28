---
page_title: "alertops_response_play Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps response play: an escalation path of users and groups with custom wait times.
---

# alertops_response_play

Manages an AlertOps response play. Response plays route alerts through a simple escalation path of users and groups tied to their schedules (rather than user roles), with a custom wait time before each tier is notified.

Reference a response play from an inbound integration with `alertops_inbound_integration.response_play`.

## Example Usage

```terraform
resource "alertops_response_play" "newrelic" {
  response_play_name = "New Relic Critical"
  description        = "Primary on-call, then the database team after 15 minutes"
  priority           = "Critical"

  members = [
    { type = "User", id = alertops_user.primary.id },
    { type = "Group", id = alertops_group.database.id, wait_time_in_mins = 15 },
  ]

  group_contact_methods = [
    { contact_method_name = "Group Email", to_bcc_or_cc = "BCC" },
    { contact_method_name = "Slack Group" },
  ]

  repeat         = true
  repeat_times   = 2
  repeat_minutes = 10

  options = {
    sla_in_hours = 4
    close = {
      phone = false
      sms   = false
    }
  }

  # Attach workflows. Omit to leave attached workflows unmanaged.
  workflows = [alertops_workflow.notify_on_ack_close.id]
}

resource "alertops_inbound_integration" "newrelic" {
  inbound_integration_name = "New Relic"
  type                     = "API"
  response_play            = alertops_response_play.newrelic.id
}
```

## Argument Reference

### Required

- `response_play_name` (String) The name of the response play.
- `description` (String) A description of the response play.
- `priority` (String) The priority: `No Priority`, `Low`, `Medium`, `High` or `Critical`.

### Optional

- `enabled` (Boolean) Whether the response play is enabled. Defaults to `true`. Workflows can only send messages through enabled response plays.
- `repeat` (Boolean) Whether to repeat the escalation path when nobody responds. Defaults to `false`.
- `repeat_times` (Number) How many times to repeat. Defaults to `0`.
- `repeat_minutes` (Number) Minutes to wait before repeating. Defaults to `0`.
- `members` (Attributes List) The escalation path, in order. The first entry is notified first; each following entry is notified after its `wait_time_in_mins`. When omitted, members are not managed. See [below for nested schema](#nestedatt--members).
- `group_contact_methods` (Attributes List) Group contact methods notified for groups in the escalation path. When omitted, they are not managed. See [below for nested schema](#nestedatt--group_contact_methods).
- `workflows` (Set of Number) IDs of workflows attached to the response play. When set, exactly these workflows are attached — including the default "Stop Delivery on Assignment or Closed" workflow AlertOps adds to every new response play, which is detached unless listed. Only enabled workflows can be attached. When omitted, attached workflows are not managed.
- `options` (Attributes) Communication options. Options that are not set keep the value AlertOps assigns. See [below for nested schema](#nestedatt--options).

<a id="nestedatt--members"></a>
### Nested Schema for `members`

- `type` (String, Required) `User` or `Group`.
- `id` (String, Required) The ID of the user or group, e.g. `alertops_user.example.id`.
- `wait_time_in_mins` (Number) Minutes to wait after the previous tier before notifying this member. Defaults to `0`.

<a id="nestedatt--group_contact_methods"></a>
### Nested Schema for `group_contact_methods`

- `contact_method_name` (String, Required) The group contact method, e.g. `Group Email`, `Group SMS`, `Slack Group`, `Microsoft Teams`.
- `to_bcc_or_cc` (String) For email methods: `To`, `CC` or `BCC`.
- `wait_time_in_mins` (Number) Minutes to wait before using this contact method. Defaults to `0`.

<a id="nestedatt--options"></a>
### Nested Schema for `options`

- `alert_type_id` (Number) The ID of the alert type used by the response play.
- `sla_in_hours` (Number) SLA in hours.
- `message_text` (String) Additional message text included in notifications.
- `include_alert_id_in_subject` (Boolean)
- `one_email_per_message` (Boolean)
- `one_message_per_recipient` (Boolean)
- `sequence_group_first` (Boolean)
- `email_notification_setting` (String) e.g. `Email Using Long Text`.
- `phone_notification_setting` (String) e.g. `VOIP`.
- `sms_notification_setting` (String) e.g. `SMS`.
- `escalation_policy_name_for_reply` (String) Escalation policy used for replies; empty means the system default.
- `acknowledgement`, `assignment`, `escalate`, `close` (Attributes) Channels notified on each alert event, each with Boolean `phone`, `sms`, `email` and `group_chat`.

Read-only: `alert_type_name` (String).

## Attributes Reference

- `id` (String) The unique numeric identifier of the response play.

## Import

```shell
terraform import alertops_response_play.example 12345
```
