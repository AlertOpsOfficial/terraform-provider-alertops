---
page_title: "alertops_topic Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps topic and the groups subscribed to it.
---

# alertops_topic

Manages an AlertOps topic. Topics are used in alert routing: alerts tagged with a topic are delivered to the groups subscribed to it.

## Example Usage

```terraform
resource "alertops_topic" "database" {
  topic_name = "Database"
  priority   = "High"
  groups     = [alertops_group.database.group_name]
}
```

## Argument Reference

### Required

- `topic_name` (String) The name of the topic.
- `priority` (String) `No Priority`, `Low`, `Medium`, `High` or `Critical`.

### Optional

- `template` (String) The topic template. Defaults to `""`.
- `groups` (Set of String) Names of the groups subscribed to the topic. When set, exactly these groups are subscribed. When omitted, subscriptions are not managed.

~> **Note:** Manage a topic's subscriptions either here or through `alertops_group.topics`, not both, or the two resources will keep undoing each other's changes.

## Attributes Reference

- `id` (String) The unique numeric identifier of the topic.

## Import

```shell
terraform import alertops_topic.example 12345
```
