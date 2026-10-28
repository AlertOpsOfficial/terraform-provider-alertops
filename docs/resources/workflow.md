---
page_title: "alertops_workflow Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps workflow: start conditions and actions that run when alerts change.
---

# alertops_workflow

Manages an AlertOps workflow. A workflow runs its actions (for example, sending a message) when its start conditions are met, such as an alert being acknowledged (`Assigned`) or `Closed`. Attach workflows to escalation policies and response plays through their `workflows` attribute.

## Example Usage

### Notify responders when an alert is acknowledged or closed

```terraform
resource "alertops_workflow" "notify_on_ack_close" {
  workflow_name = "Notify on Ack or Close"
  workflow_type = "Alert"
  alert_type    = "Standard Alert"

  conditions = [
    { match = "any", name = "AlertStatus", operator = "is", value = "Assigned" },
    { match = "any", name = "AlertStatus", operator = "is", value = "Closed" },
  ]

  actions = [
    {
      name                        = "SendMessage"
      value                       = alertops_response_play.newrelic.response_play_name
      is_escalation_policy        = false
      send_to_original_recipients = true
      subject                     = "Alert status changed"
      message_text                = "The alert has been acknowledged or closed."
    },
  ]
}

resource "alertops_escalation_policy" "default" {
  escalation_policy_name = "Default"
  workflows              = [alertops_workflow.notify_on_ack_close.id]
}
```

## Argument Reference

### Required

- `workflow_name` (String) The name of the workflow.
- `workflow_type` (String) `Alert`, `Message` or `Notification`. Changing this forces a new workflow.

### Optional

- `alert_type` (String) The alert type name the workflow applies to, e.g. `Standard Alert`.
- `enabled` (Boolean) Whether the workflow is enabled. Defaults to `true`. Only enabled workflows can be attached to escalation policies and response plays.
- `scheduled` (Boolean) Whether the workflow runs on a schedule. Defaults to `false`.
- `recurrence_interval` (Number) Recurrence interval for scheduled workflows. Defaults to `0`.
- `conditions` (Attributes List) Start conditions. Defaults to none. See [below for nested schema](#nestedatt--conditions).
- `actions` (Attributes List) Actions to run. Defaults to none. See [below for nested schema](#nestedatt--actions).

The provider manages the complete list of conditions and actions: any not in the configuration are removed.

<a id="nestedatt--conditions"></a>
### Nested Schema for `conditions`

- `name` (String, Required) The field to test, e.g. `AlertStatus`, `Topic`, `Owner`, `CompleteDelivery`, or a custom alert field.
- `operator` (String, Required) The operator, e.g. `is`, `is not`.
- `value` (String) The value to compare with. `AlertStatus` values: `Open`, `Closed`, `Cancelled`, `Assigned`, `On Hold`, `Merged`, `Snoozed`.
- `match` (String) `all` (every such condition must match) or `any` (one must match). Defaults to `all`.
- `type` (String) Defaults to `start`.
- `list_id` (String) List identifier for list-based conditions.

<a id="nestedatt--actions"></a>
### Nested Schema for `actions`

- `name` (String, Required) The action, e.g. `SendMessage`, `Send Webhook`, `Update Status`, `Update Priority`, `Update Alert Field`, `CancelDeliveryThread`, `Outbound Service Notification`.
- `value` (String) The action value. For `SendMessage`, the **name** of the escalation policy or response play used to deliver the message (it must be enabled). For `Send Webhook`, the HTTP method: `GET`, `PUT`, `POST` or `PATCH`.
- `is_escalation_policy` (Boolean) For `SendMessage`: `true` if `value` names an escalation policy, `false` for a response play. Determined by AlertOps when omitted.
- `send_to_original_recipients`, `send_to_sender`, `send_to_owner`, `launch_new_thread` (Boolean) Defaults to `false`.
- `subject` (String) Message subject.
- `message_text` (String) Message body; HTML is allowed.
- `users` (List of String) Usernames to notify.
- `groups` (List of String) Group names to notify.
- `webhook_url` (String) URL for webhook actions.

## Attributes Reference

- `id` (String) The unique numeric identifier of the workflow.

## Import

```shell
terraform import alertops_workflow.example 12345
```
