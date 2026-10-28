# End-to-end smoke test for the resources added in v2.1.
# Every object is prefixed "TF Test" so it is easy to find and clean up.
#
#   terraform apply   -var user_id=<id> -var group_id=<id> -var group_name=<name>
#   terraform plan    (same vars)  -> expect "No changes"
#   terraform destroy (same vars)

terraform {
  required_providers {
    alertops = {
      source = "alertopsofficial/alertops"
    }
  }
}

provider "alertops" {}

variable "user_id" {
  description = "ID of an existing user to place first in the escalation path."
  type        = string
}

variable "group_id" {
  description = "ID of an existing group to place second in the escalation path."
  type        = string
}

variable "group_name" {
  description = "Name of the same group, used for the topic subscription."
  type        = string
}

variable "wait_minutes" {
  description = "Wait before notifying the group tier. Change it to exercise updates."
  type        = number
  default     = 15
}

resource "alertops_topic" "test" {
  topic_name = "TF Test Topic TF"
  priority   = "Low"
  groups     = [var.group_name]
}

resource "alertops_response_play" "test" {
  response_play_name = "TF Test RP TF"
  description        = "TF Test - managed by Terraform"
  priority           = "High"

  members = [
    { type = "User", id = var.user_id },
    { type = "Group", id = var.group_id, wait_time_in_mins = var.wait_minutes },
  ]

  group_contact_methods = [
    { contact_method_name = "Group Email", to_bcc_or_cc = "BCC" },
  ]

  options = {
    sla_in_hours = 6
  }
}

resource "alertops_workflow" "notify_on_ack_close" {
  workflow_name = "TF Test WF TF Notify on Ack-Close"
  workflow_type = "Alert"
  alert_type    = "Standard Alert"

  conditions = [
    { match = "any", name = "AlertStatus", operator = "is", value = "Assigned" },
    { match = "any", name = "AlertStatus", operator = "is", value = "Closed" },
  ]

  actions = [
    {
      name                        = "SendMessage"
      value                       = alertops_response_play.test.response_play_name
      is_escalation_policy        = false
      send_to_original_recipients = true
      subject                     = "TF Test - alert status changed"
      message_text                = "TF Test - the alert was acknowledged or closed."
    },
  ]
}

resource "alertops_escalation_policy" "test" {
  escalation_policy_name = "TF Test EP TF"
  description            = "TF Test - managed by Terraform"
  priority               = "Low"
  workflows              = [alertops_workflow.notify_on_ack_close.id]
}

resource "alertops_inbound_integration" "test" {
  inbound_integration_name = "TF Test Inbound TF"
  type                     = "API"
  enabled                  = false
  response_play            = alertops_response_play.test.id
}

output "response_play_id" {
  value = alertops_response_play.test.id
}

output "workflow_id" {
  value = alertops_workflow.notify_on_ack_close.id
}

output "inbound_api_url" {
  value     = alertops_inbound_integration.test.api_url
  sensitive = true
}
