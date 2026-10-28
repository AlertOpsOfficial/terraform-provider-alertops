# Regression suite covering every resource type and the more complex arguments.
# All objects are prefixed "TF Test". Integrations are disabled and contact details use
# reserved example.com addresses and fictional 555-01xx numbers, so nothing is notified.
#
#   terraform apply                    # revision 1: create
#   terraform plan                     # expect "No changes"
#   terraform apply -var revision=2    # adds, removes and changes nested items
#   terraform plan  -var revision=2    # expect "No changes"
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
  type    = number
  default = 1
}

# Business Services require a plan that includes them (otherwise the API returns 403).
variable "test_services" {
  type    = bool
  default = false
}

# Creating outbound integrations through the v2 API returned HTTP 500 on the test account
# for every payload, including the web app's own; enable once that is resolved.
variable "test_outbound" {
  type    = bool
  default = false
}

locals {
  v2 = var.revision >= 2
}

# --- User with email, phone and SMS ---------------------------------------------------

resource "alertops_user" "oncall" {
  user_name  = "tftest.oncall@example.com"
  first_name = "TF Test"
  last_name  = "Oncall"
  locale     = "en-US"
  time_zone  = "(UTC-06:00) Central Time (US & Canada)"
  type       = "Standard"
  roles      = ["Basic"]

  contact_methods = concat(
    [
      {
        contact_method_name = "Email-Official"
        email_address       = "tftest.oncall@example.com"
        sequence            = 1
      },
      {
        contact_method_name = "Phone-Official"
        country_code        = "1"
        phone_number        = "5555550100"
        wait_time_in_mins   = local.v2 ? 7 : 5
        repeat              = true
        repeat_times        = 2
        repeat_minutes      = 5
        sequence            = 2
      },
    ],
    local.v2 ? [] : [
      {
        contact_method_name = "SMS-Official"
        country_code        = "1"
        phone_number        = "5555550101"
        wait_time_in_mins   = 10
        sequence            = 3
      },
    ],
  )
}

# --- Topic, group (with contact method and topic), rotating schedule ---------------------

resource "alertops_topic" "adv" {
  topic_name = "TF Test Topic Adv"
  priority   = local.v2 ? "High" : "Medium"
}

resource "alertops_group" "adv" {
  group_name  = "TF Test Group Adv"
  description = ["TF Test - regression group"]

  members = [
    {
      member_type = "User"
      member      = alertops_user.oncall.user_name
      sequence    = 1
      # roles is omitted: the rotating schedule below assigns roles per shift, so AlertOps
      # sets a default role on create and clears group-level roles once the schedule exists.
    },
  ]

  contact_methods = [
    {
      contact_method_name = "Group Email"
      email_address       = local.v2 ? "tftest.group2@example.com" : "tftest.group@example.com"
      enabled             = true
      sequence            = 1
    },
  ]

  topics = [alertops_topic.adv.topic_name]
}

resource "alertops_schedule" "rotating" {
  group                      = alertops_group.adv.group_name
  schedule_name              = "TF Test Rotating 24x7"
  schedule_type              = "Rotating"
  time_zone                  = "(UTC) Coordinated Universal Time"
  continuous                 = true
  include_all_users_in_group = false
  color                      = local.v2 ? "#33AA55" : "#8844CC"
  start_weekday              = "Sun"
  end_weekday                = "Sat"

  weekday_sun = true
  weekday_mon = true
  weekday_tue = true
  weekday_wed = true
  weekday_thu = true
  weekday_fri = true
  weekday_sat = true

  # Rotation settings are not first-class arguments; pass them through extra_attributes.
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
      { user = alertops_user.oncall.user_name, role = "Primary" },
    ])
  }
}

# --- Workflows of every type ---------------------------------------------------------------

resource "alertops_workflow" "alert_webhook" {
  workflow_name = "TF Test WF Alert Webhook"
  workflow_type = "Alert"
  alert_type    = "Standard Alert"

  conditions = concat(
    [{ name = "AlertStatus", operator = "is", value = "Closed" }],
    local.v2 ? [{ name = "Topic", operator = "is", value = alertops_topic.adv.topic_name }] : [],
  )

  actions = [
    {
      name        = "Send Webhook"
      value       = "POST" # HTTP method: GET, PUT, POST or PATCH
      webhook_url = local.v2 ? "https://example.com/tf-test-webhook-v2" : "https://example.com/tf-test-webhook"
    },
  ]
}

resource "alertops_workflow" "notification" {
  workflow_name = "TF Test WF Cancel On Ack"
  workflow_type = "Notification"

  conditions = [
    { name = "DeliveryStatusType", operator = "is", value = "Acknowledged" },
  ]

  actions = [
    { name = "CancelUserDeliveries" },
  ]
}

resource "alertops_workflow" "message" {
  workflow_name = local.v2 ? "TF Test WF Message v2" : "TF Test WF Message"
  workflow_type = "Message"
  alert_type    = "Standard Message"
  enabled       = false

  actions = [
    { name = "Send Webhook", value = "POST", webhook_url = "https://example.com/tf-test-message" },
  ]
}

# --- Escalation policy with member roles, options and a workflow ---------------------------

resource "alertops_escalation_policy" "adv" {
  escalation_policy_name = "TF Test EP Adv"
  description            = "TF Test - regression escalation policy"
  priority               = "High"

  # Per-role retries and contact methods require centralized (policy-level) notification settings.
  notify_using_centralized_settings = true

  options = {
    sla_in_hours                = local.v2 ? 48 : 24
    include_alert_id_in_subject = true
  }

  member_roles = concat(
    [
      {
        member_role_type                  = "Primary"
        wait_time_between_members_in_mins = 5
        no_of_retries                     = 1
        retry_interval                    = 5
        contact_methods = [
          { contact_method_name = "Email-Official", wait_time_in_mins = 0, sequence = 1, to_bcc_or_cc = "To" },
          { contact_method_name = "Phone-Official", wait_time_in_mins = 5, sequence = 2 },
        ]
      },
    ],
    local.v2 ? [] : [
      {
        member_role_type                  = "Secondary"
        wait_time_between_members_in_mins = 10
        no_of_retries                     = 0
        retry_interval                    = 0
        role_wait_time_in_mins            = 15
        contact_methods = [
          { contact_method_name = "Email-Official", wait_time_in_mins = 0, sequence = 1, to_bcc_or_cc = "To" },
        ]
      },
    ],
  )

  workflows = [alertops_workflow.alert_webhook.id]
}

# --- Response play with three tiers, contact methods, repeat, options, workflow -------------

resource "alertops_response_play" "adv" {
  response_play_name = "TF Test RP Adv"
  description        = "TF Test - regression response play"
  priority           = "Critical"

  members = concat(
    [{ type = "User", id = alertops_user.oncall.id }],
    local.v2 ? [] : [{ type = "Group", id = alertops_group.adv.id, wait_time_in_mins = 10 }],
    [{ type = "User", id = alertops_user.oncall.id, wait_time_in_mins = 30 }],
  )

  group_contact_methods = concat(
    [{ contact_method_name = "Group Email", to_bcc_or_cc = local.v2 ? "BCC" : "CC" }],
    local.v2 ? [] : [{ contact_method_name = "Group SMS" }],
  )

  repeat         = !local.v2
  repeat_times   = local.v2 ? 0 : 1
  repeat_minutes = local.v2 ? 0 : 15

  options = {
    sla_in_hours = 8
    message_text = local.v2 ? "TF Test v2" : "TF Test"
    close = {
      email = !local.v2
      sms   = false
    }
  }

  workflows = [alertops_workflow.notification.id]
}

# --- Inbound integrations of each type -------------------------------------------------------

resource "alertops_inbound_integration" "email" {
  inbound_integration_name = local.v2 ? "TF Test Inbound Email v2" : "TF Test Inbound Email"
  type                     = "Email"
  enabled                  = false
  escalation_policy        = alertops_escalation_policy.adv.escalation_policy_name

  # Email integrations require a mailbox name.
  extra_attributes = {
    mail_box = "tftestinbound"
  }
}

resource "alertops_inbound_integration" "heartbeat" {
  inbound_integration_name = "TF Test Inbound Heartbeat"
  type                     = "Heartbeat"
  enabled                  = false
  # Switches from the response play to the escalation policy in revision 2.
  response_play     = local.v2 ? null : alertops_response_play.adv.id
  escalation_policy = local.v2 ? alertops_escalation_policy.adv.escalation_policy_name : null
}

resource "alertops_inbound_integration" "api_custom" {
  inbound_integration_name = "TF Test Inbound API Custom"
  type                     = "API"
  # Maintenance windows can only target enabled integrations. It routes to example.com
  # and 555-01xx contacts only, and nothing posts to it.
  enabled           = true
  escalation_policy = alertops_escalation_policy.adv.escalation_policy_name
  recipient_groups  = [alertops_group.adv.group_name]

  extra_attributes = {
    api_settings = jsonencode({
      url_mapping = {
        method       = "POST"
        content      = "JSON"
        source       = "TFTestSource"
        source_name  = local.v2 ? "TF Test Source v2" : "TF Test Source"
        static       = true
        source_value = "TFTestSource"
        open_alert_when = {
          field_name = "Body"
          type       = "Contains Any "
          values     = [""]
        }
      }
    })
  }
}

# --- Service, maintenance window, outbound integration --------------------------------------

resource "alertops_service" "adv" {
  count = var.test_services ? 1 : 0

  service_name = "TF Test Service"
  show_uptime  = !local.v2

  components = concat(
    [{ component_name = "TF Test API", external_id = "tf-test-api" }],
    local.v2 ? [{ component_name = "TF Test DB", external_id = "tf-test-db" }] : [],
  )

  templates = [
    {
      template_name = "TF Test Template"
      subject       = local.v2 ? "TF Test degraded (v2)" : "TF Test degraded"
      message       = "TF Test - service degraded"
    },
  ]
}

resource "alertops_maintenance_window" "weekly" {
  description = local.v2 ? "TF Test MW v2" : "TF Test MW"
  recurrence  = "Weekly"
  # The API requires dates for every recurrence (it returns HTTP 500 without them).
  start_date  = "2026-11-02"
  end_date    = "2027-12-31"
  start_time  = local.v2 ? "03:00" : "02:00"
  end_time    = local.v2 ? "04:00" : "03:00"
  integration = [alertops_inbound_integration.api_custom.id]

  sunday    = false
  monday    = false
  tuesday   = false
  wednesday = false
  thursday  = false
  friday    = false
  saturday  = true
}

resource "alertops_outbound_integration" "webhook" {
  count = var.test_outbound ? 1 : 0

  outbound_integration_name = local.v2 ? "TF Test Outbound v2" : "TF Test Outbound"
  security_type             = "None"

  extra_attributes = {
    methods = jsonencode([
      {
        method_name         = "TF Test POST"
        alert_type          = "Standard Alert"
        update_alert_fields = false
        method_type = {
          type               = "REST"
          web_method         = "POST"
          uri                = "https://example.com/tf-test-outbound"
          request_type       = "JSON"
          response_data_type = "JSON"
          request_data       = jsonencode({ text = "TF Test" })
          response_data      = ""
          headers            = jsonencode({})
        }
      },
    ])
  }
}

output "ids" {
  value = {
    user          = alertops_user.oncall.id
    topic         = alertops_topic.adv.id
    group         = alertops_group.adv.id
    schedule      = alertops_schedule.rotating.id
    wf_alert      = alertops_workflow.alert_webhook.id
    wf_notify     = alertops_workflow.notification.id
    wf_message    = alertops_workflow.message.id
    ep            = alertops_escalation_policy.adv.id
    rp            = alertops_response_play.adv.id
    inbound_email = alertops_inbound_integration.email.id
    inbound_hb    = alertops_inbound_integration.heartbeat.id
    inbound_api   = alertops_inbound_integration.api_custom.id
    service       = one(alertops_service.adv[*].id)
    maintenance   = alertops_maintenance_window.weekly.id
    outbound      = one(alertops_outbound_integration.webhook[*].id)
  }
}
