---
page_title: "alertops_escalation_policy Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps escalation policy, which defines how and when alerts are escalated through notification chains.
---

# alertops_escalation_policy

Manages an AlertOps escalation policy. Escalation policies define the priority and notification chain for alerts — they are referenced by inbound integrations to route alerts to the right on-call team.

Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.

## Example Usage

### Simple escalation policy

```terraform
resource "alertops_escalation_policy" "default" {
  escalation_policy_name = "Default Policy"
  description            = "Default escalation for production alerts"
  priority               = "High"
  enabled                = true
}
```

### Policy with SLA and notification options

```terraform
resource "alertops_escalation_policy" "critical" {
  escalation_policy_name = "Critical Incidents"
  description            = "24-hour SLA for P1 incidents"
  priority               = "Critical"
  enabled                = true

  options = {
    sla_in_hours                = 24
    include_alert_id_in_subject = true
    one_email_per_message       = false
    one_message_per_recipient   = true
    sequence_group_first        = false
  }
}
```

### Policy with member roles and notification chain

```terraform
resource "alertops_escalation_policy" "tiered" {
  escalation_policy_name                       = "Tiered Escalation"
  description                                  = "Primary on-call with retries"
  priority                                     = "High"
  wait_time_before_notifying_next_group_in_min = 15

  # Per-role retries and contact methods require policy-level (centralized) settings;
  # with user settings AlertOps requires no_of_retries = 0 and no contact_methods.
  notify_using_centralized_settings = true

  member_roles = [
    {
      member_role_type                  = "Primary"
      wait_time_between_members_in_mins = 5
      no_of_retries                     = 3
      retry_interval                    = 5

      contact_methods = [
        {
          contact_method_name = "Email-Official"
          wait_time_in_mins   = 0
          sequence            = 1
          to_bcc_or_cc        = "To" # required for email contact methods
        }
      ]
    }
  ]
}
```

## Argument Reference

### Required

- `escalation_policy_name` (String) The unique name of the escalation policy.

### Optional

- `description` (String) A description of the escalation policy. Although optional in the schema, the AlertOps API rejects an empty description when creating a policy.
- `priority` (String) The alert priority level. Valid values: `Low`, `Medium`, `High`, `Critical`.
- `enabled` (Boolean) Whether the escalation policy is active. Defaults to `true`.
- `quick_launch` (Boolean) Whether quick launch is enabled for this policy. Defaults to `false`.
- `notify_using_centralized_settings` (Boolean) Whether to use account-level centralized notification settings. Defaults to `false`.
- `wait_time_before_notifying_next_group_in_min` (Number) Minutes to wait before escalating to the next group. Defaults to `0`.
- `options` (Object) Notification and SLA options. See [options](#nestedblock--options) below.
- `member_roles` (List of Object) Ordered list of member roles defining the notification chain. See [member_roles](#nestedblock--member_roles) below.
- `workflows` (Set of Number) IDs of workflows attached to the policy, e.g. `[alertops_workflow.example.id]`. When set, exactly these workflows are attached — including the default "Stop Delivery on Assignment or Closed" workflow AlertOps adds to new policies, which is detached unless listed. Only enabled workflows can be attached. When omitted, attached workflows are not managed.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload.

### `options`

- `sla_in_hours` (Number, Optional) SLA target in hours. The API enforces a minimum of 24; setting `0` results in `24` being stored.
- `include_alert_id_in_subject` (Boolean, Optional) Include the alert ID in notification subjects. Defaults to `false`.
- `one_email_per_message` (Boolean, Optional) Send a separate email for each message. Defaults to `false`.
- `one_message_per_recipient` (Boolean, Optional) Send a separate message to each recipient. Defaults to `false`.
- `sequence_group_first` (Boolean, Optional) Notify the group before individual members. Defaults to `false`.

### `member_roles`

Each entry in `member_roles` defines a tier in the escalation chain:

- `wait_time_between_members_in_mins` (Number, Required) Minutes to wait between notifying each member in this role.
- `no_of_retries` (Number, Required) Number of retry attempts for this role.
- `retry_interval` (Number, Required) Minutes between retry attempts.
- `member_role_type` (String, Optional) The role type label (e.g., `Primary`, `Backup`).
- `role_wait_time_in_mins` (Number, Optional) Minutes to wait before activating this role. Defaults to `0`.
- `contact_methods` (List of Object, Optional) Contact methods used for this role. See [contact_methods](#nestedblock--member_roles--contact_methods) below.

### `member_roles.contact_methods`

- `contact_method_name` (String, Required) The contact method name (e.g., `Email-Official`, `Phone-Official`).
- `wait_time_in_mins` (Number, Required) Minutes to wait before using this contact method.
- `sequence` (Number, Required) The order in which this contact method is used.
- `repeat` (Boolean, Optional) Whether to retry this contact method. Defaults to `false`.
- `repeat_times` (Number, Optional) Number of retry attempts. Defaults to `0`.
- `repeat_minutes` (Number, Optional) Minutes between retries. Defaults to `0`.
- `to_bcc_or_cc` (String, Optional) For email contact methods: `To`, `CC` or `BCC`. Required by the API for email methods.

## Attributes Reference

- `id` (String) The unique numeric identifier of the escalation policy, assigned by AlertOps.

## Import

Escalation policies can be imported using their numeric AlertOps ID:

```shell
terraform import alertops_escalation_policy.example 12345
```
