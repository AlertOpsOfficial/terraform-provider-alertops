terraform {
  required_providers {
    alertops = {
      source  = "alertopsofficial/alertops"
      version = "~> 2.1"
    }
  }
}

provider "alertops" {
  # Set via ALERTOPS_API_KEY environment variable or uncomment below:
  # api_key = var.alertops_api_key
}

# Password the outbound integration example uses for Basic auth.
variable "outbound_basic_auth_password" {
  type      = string
  sensitive = true
}

# ============================================================
# 1. USER RESOURCE
# ============================================================
resource "alertops_user" "test" {
  user_name  = "tf-crud-test-user"
  first_name = "TF"
  last_name  = "CRUDTest"
  locale     = "en-US"
  time_zone  = "(UTC-06:00) Central Time (US & Canada)"
  type       = "Standard"
  roles      = ["App Admin"]

  contact_methods = [
    {
      contact_method_name   = "Email-Official"
      email_address         = "tf-crud-test@example.com"
      wait_time_in_mins     = 0
      repeat                = false
      repeat_times          = 0
      repeat_minutes        = 0
      notification_time24x7 = true
      enabled               = true
      sequence              = 1
      country_code          = ""
      phone_number          = ""
      phone_extension       = ""
    }
  ]
}

# ============================================================
# 2. GROUP RESOURCE
# ============================================================
resource "alertops_group" "test" {
  group_name = "tf-crud-test-group"
  dynamic    = false

  members = [
    {
      member_type = "User"
      member      = alertops_user.test.user_name
      sequence    = 1
      roles       = ["Primary"]
    }
  ]

  depends_on = [alertops_user.test]
}

# ============================================================
# 3. ESCALATION POLICY RESOURCE
# ============================================================
resource "alertops_escalation_policy" "test" {
  escalation_policy_name = "tf-crud-test-ep"
  description            = "Updated by Terraform CRUD test - v3"
  priority               = "Low"
  enabled                = true
  member_roles           = []
}

# ============================================================
# 4. SCHEDULE RESOURCE
# Requires: group with at least one member, color field
# ============================================================
resource "alertops_schedule" "test" {
  group                      = alertops_group.test.group_name
  schedule_name              = "tf-crud-test-schedule"
  schedule_type              = "Fixed"
  time_zone                  = "(UTC-06:00) Central Time (US & Canada)"
  continuous                 = true
  include_all_users_in_group = true
  enabled                    = true
  color                      = "#FF5733"
  start_weekday              = "Mon"
  end_weekday                = "Fri"
  weekday_sun                = false
  weekday_mon                = true
  weekday_tue                = true
  weekday_wed                = true
  weekday_thu                = true
  weekday_fri                = true
  weekday_sat                = false

  depends_on = [alertops_group.test]
}

# ============================================================
# 5. INBOUND INTEGRATION RESOURCE
# Requires: api_settings with url_mapping (source must be non-empty)
# field_name must be one of: Body, Subject, SourceStatus
# ============================================================
resource "alertops_inbound_integration" "test" {
  inbound_integration_name = "tf-crud-test-inbound"
  type                     = "API"
  sequence                 = 1
  enabled                  = true
  escalation_policy        = alertops_escalation_policy.test.escalation_policy_name
  inbound_template_id      = 0

  extra_attributes = {
    "response_play_id" = "0"
    "api_settings" = jsonencode({
      is_bidirection = false
      is_secure      = false
      url_mapping = {
        method      = "POST"
        content     = "JSON"
        source      = "dev"
        source_name = "tf_test"
        static      = false
        open_alert_when = {
          field_name = "SourceStatus"
          type       = "Contains Any"
          values     = ["open"]
        }
        close_alert_when = {
          type   = "Contains Any"
          values = ["closed"]
        }
      }
    })
  }

  depends_on = [alertops_escalation_policy.test]
}

# ============================================================
# 6. MAINTENANCE WINDOW RESOURCE
# Requires: recurrence = "Once" (not "None"), integration as array of IDs
# ============================================================
resource "alertops_maintenance_window" "test" {
  description = "TF CRUD Test Maintenance Window"
  recurrence  = "Once"
  start_date  = "2027-04-01"
  end_date    = "2027-04-01"
  start_time  = "02:00"
  end_time    = "04:00"
  integration = [tostring(alertops_inbound_integration.test.id)]
  # For recurrence = "Once", the API ignores weekday fields and returns all false
  sunday    = false
  monday    = false
  tuesday   = false
  wednesday = false
  thursday  = false
  friday    = false
  saturday  = false

  depends_on = [alertops_inbound_integration.test]
}

# ============================================================
# 7. SERVICE RESOURCE
# ============================================================
resource "alertops_service" "test" {
  service_name = "tf-crud-test-service-v2"
  show_uptime  = true
}

# ============================================================
# 8. OUTBOUND INTEGRATION RESOURCE
# Requires: security_type = "Basic" when methods are included
# ============================================================
resource "alertops_outbound_integration" "test" {
  outbound_integration_name = "tf-crud-test-outbound"
  security_type             = "Basic"

  extra_attributes = {
    "security" = jsonencode({
      user_name = "tf-test@example.com"
      password  = var.outbound_basic_auth_password
    })
    "methods" = jsonencode([
      {
        method_name = "POST Alert"
        method_type = {
          request_type       = "JSON"
          web_method         = "POST"
          response_data_type = "JSON"
          type               = "REST"
          uri                = "https://webhook.site/test"
          request_data       = "{\"alert\": \"test\"}"
          response_data      = ""
          headers            = "{}"
        }
        alert_type          = "Standard Alert"
        update_alert_fields = false
      }
    ])
  }
}

# ============================================================
# OUTPUTS
# ============================================================
output "user_id" {
  value = alertops_user.test.id
}

output "group_id" {
  value = alertops_group.test.id
}

output "escalation_policy_id" {
  value = alertops_escalation_policy.test.id
}

output "schedule_id" {
  value = alertops_schedule.test.id
}

output "inbound_integration_id" {
  value = alertops_inbound_integration.test.id
}

output "maintenance_window_id" {
  value = alertops_maintenance_window.test.id
}

output "service_id" {
  value = alertops_service.test.id
}

output "outbound_integration_id" {
  value = alertops_outbound_integration.test.id
}
