# Full-stack smoke test: user -> group -> schedule -> escalation policy ->
# response play -> inbound integrations. Every object is prefixed "TF Test".
#
#   terraform apply                      # create
#   terraform plan                       # expect "No changes"
#   terraform apply -var revision=2      # update every resource
#   terraform plan  -var revision=2      # expect "No changes"
#   terraform destroy -var revision=2

terraform {
  required_providers {
    alertops = {
      source = "alertops/alertops"
    }
  }
}

provider "alertops" {}

variable "revision" {
  description = "Bump to 2 to exercise in-place updates on every resource."
  type        = number
  default     = 1
}

locals {
  v2 = var.revision >= 2
}

resource "alertops_user" "pqr" {
  user_name  = "pqr@abc.com"
  first_name = "TF Test"
  last_name  = local.v2 ? "PQR Updated" : "PQR"
  locale     = "en-US"
  time_zone  = "(UTC-05:00) Eastern Time (US & Canada)"
  type       = "Standard"
  roles      = ["Basic"]

  contact_methods = [
    {
      contact_method_name = "Email-Official"
      # example.com is reserved (RFC 2606) so no real mailbox receives notifications.
      email_address = "pqr@example.com"
      sequence      = 1
    },
  ]
}

resource "alertops_group" "team" {
  group_name  = "TF Test Group"
  description = [local.v2 ? "TF Test - updated by Terraform" : "TF Test - managed by Terraform"]

  members = [
    {
      member_type = "User"
      member      = alertops_user.pqr.user_name
      sequence    = 1
      roles       = ["Primary"]
    },
  ]
}

resource "alertops_schedule" "business_hours" {
  group         = alertops_group.team.group_name
  schedule_name = "TF Test Business Hours"
  schedule_type = "Fixed"
  time_zone     = "(UTC-05:00) Eastern Time (US & Canada)"
  color         = local.v2 ? "#FF5733" : "#0052CC"
  start_weekday = "Mon"
  end_weekday   = local.v2 ? "Sat" : "Fri"

  weekday_mon = true
  weekday_tue = true
  weekday_wed = true
  weekday_thu = true
  weekday_fri = true
  weekday_sat = local.v2
  weekday_sun = false
}

resource "alertops_escalation_policy" "team" {
  escalation_policy_name = "TF Test Escalation Policy"
  description            = local.v2 ? "TF Test - updated by Terraform" : "TF Test - managed by Terraform"
  priority               = local.v2 ? "High" : "Medium"
}

resource "alertops_response_play" "team" {
  response_play_name = "TF Test Response Play"
  description        = local.v2 ? "TF Test - updated by Terraform" : "TF Test - managed by Terraform"
  priority           = "High"

  members = [
    { type = "User", id = alertops_user.pqr.id },
    { type = "Group", id = alertops_group.team.id, wait_time_in_mins = local.v2 ? 20 : 10 },
  ]
}

resource "alertops_inbound_integration" "via_escalation_policy" {
  inbound_integration_name = "TF Test Integration EP"
  type                     = "API"
  enabled                  = false
  escalation_policy        = alertops_escalation_policy.team.escalation_policy_name
  recipient_groups         = [alertops_group.team.group_name]
}

resource "alertops_inbound_integration" "via_response_play" {
  inbound_integration_name = local.v2 ? "TF Test Integration RP v2" : "TF Test Integration RP"
  type                     = "API"
  enabled                  = false
  response_play            = alertops_response_play.team.id
}

output "ids" {
  value = {
    user              = alertops_user.pqr.id
    group             = alertops_group.team.id
    schedule          = alertops_schedule.business_hours.id
    escalation_policy = alertops_escalation_policy.team.id
    response_play     = alertops_response_play.team.id
    integration_ep    = alertops_inbound_integration.via_escalation_policy.id
    integration_rp    = alertops_inbound_integration.via_response_play.id
  }
}

output "integration_rp_api_url" {
  value     = alertops_inbound_integration.via_response_play.api_url
  sensitive = true
}
