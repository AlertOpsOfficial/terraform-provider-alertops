---
page_title: "alertops_group Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps group, which is a named collection of users (and sub-groups) that can be referenced in schedules, escalation policies, and integrations.
---

# alertops_group

Manages an AlertOps group. Groups are used to organize on-call users and are the primary target for schedules, escalation policies, and integrations.

Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.

## Example Usage

### Simple group with a single user

```terraform
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
```

### Group with multiple members and a group contact method

```terraform
resource "alertops_group" "platform_team" {
  group_name  = "platform-team"
  description = ["Platform engineering on-call rotation"]

  members = [
    {
      member_type = "User"
      member      = alertops_user.alice.user_name
      sequence    = 1
      roles       = ["Primary", "Manager"]
    },
    {
      member_type = "User"
      member      = alertops_user.bob.user_name
      sequence    = 2
      roles       = ["Primary"]
    }
  ]

  contact_methods = [
    {
      contact_method_name = "Slack-Ops"
      url                 = "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
      get_alert_update    = true
      enabled             = true
      sequence            = 1
    }
  ]

  topics = ["infrastructure", "kubernetes"]
}
```

### Nested group (group of groups)

```terraform
resource "alertops_group" "all_engineering" {
  group_name = "all-engineering"

  members = [
    {
      member_type = "Group"
      member      = alertops_group.platform_team.group_name
      sequence    = 1
      roles       = ["Primary"]
    },
    {
      member_type = "Group"
      member      = alertops_group.ops_team.group_name
      sequence    = 2
      roles       = ["Primary"]
    }
  ]
}
```

## Argument Reference

### Required

- `group_name` (String) The unique name of the group.

### Optional

- `dynamic` (Boolean) Whether the group uses dynamic membership. Defaults to `false`.
- `description` (List of String) One or more description strings for the group.
- `members` (List of Object) Members of the group. See [members](#nestedblock--members) below.
- `contact_methods` (List of Object) Contact methods for the group (e.g., a Slack webhook). See [contact_methods](#nestedblock--contact_methods) below.
- `topics` (List of String) Topic names associated with the group for routing purposes.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload.

### `members`

- `member_type` (String, Required) The type of member. Valid values: `User`, `Group`.
- `member` (String, Required) The username (for `User`) or group name (for `Group`) of the member.
- `sequence` (Number, Optional) The escalation order of this member within the group. Defaults to `1`.
- `roles` (List of String, Optional) Roles assigned to this member in the group (e.g., `["Primary"]`, `["Primary", "Manager"]`).

### `contact_methods`

- `contact_method_name` (String, Required) The name/label for this contact method.
- `email_address` (String, Optional) Email address for email-based contact methods.
- `country_code` (String, Optional) Country code for phone/SMS contact methods.
- `phone_number` (String, Optional) Phone number for phone/SMS contact methods.
- `extension` (String, Optional) Phone extension.
- `url` (String, Optional) Webhook URL for chat-based contact methods (e.g., Slack, Teams).
- `get_alert_update` (Boolean, Optional) Whether this contact method receives alert status updates. Defaults to `false`.
- `enabled` (Boolean, Optional) Whether this contact method is active. Defaults to `true`.
- `sequence` (Number, Optional) The order of this contact method. Defaults to `1`.

~> **Note:** When a schedule assigns roles to its users (for example a rotating schedule with a `users` list), AlertOps manages roles per shift and clears group-level member `roles`. Omit `roles` for those members so AlertOps can manage them; setting roles there causes Terraform to keep planning to restore them.

## Attributes Reference

- `id` (String) The unique numeric identifier of the group, assigned by AlertOps.

## Import

Groups can be imported using their numeric AlertOps group ID:

```shell
terraform import alertops_group.example 12345
```
