---
page_title: "alertops_user Resource - alertops"
subcategory: ""
description: |-
  Manages an AlertOps user, including contact methods for email, phone, and SMS notification delivery.
---

# alertops_user

Manages an AlertOps user. At least one `Email-Official` contact method must be provided.

Any additional fields supported by the AlertOps API can be passed via `extra_attributes`; they are merged into the request payload. If the API rejects a field, the raw error is returned.

## Example Usage

### Basic user with email contact method

```terraform
resource "alertops_user" "example" {
  user_name  = "jsmith"
  first_name = "Jane"
  last_name  = "Smith"
  locale     = "en-US"
  time_zone  = "(UTC-05:00) Eastern Time (US & Canada)"
  type       = "Standard"
  roles      = ["App Admin"]

  contact_methods = [
    {
      contact_method_name   = "Email-Official"
      email_address         = "jsmith@example.com"
      notification_time24x7 = true
      enabled               = true
      sequence              = 1
    }
  ]
}
```

### User with email and phone contact methods

```terraform
resource "alertops_user" "example" {
  user_name  = "jsmith"
  first_name = "Jane"
  last_name  = "Smith"
  locale     = "en-US"
  time_zone  = "(UTC-05:00) Eastern Time (US & Canada)"
  type       = "Standard"
  roles      = ["App Admin"]

  contact_methods = [
    {
      contact_method_name   = "Email-Official"
      email_address         = "jsmith@example.com"
      notification_time24x7 = true
      enabled               = true
      sequence              = 1
    },
    {
      contact_method_name   = "Phone-Official"
      country_code          = "1"
      phone_number          = "5555550100"
      phone_extension       = ""
      wait_time_in_mins     = 5
      repeat                = true
      repeat_times          = 3
      repeat_minutes        = 5
      notification_time24x7 = true
      enabled               = true
      sequence              = 2
    },
    {
      contact_method_name   = "SMS-Official"
      country_code          = "1"
      phone_number          = "5555550100"
      wait_time_in_mins     = 10
      repeat                = false
      notification_time24x7 = true
      enabled               = true
      sequence              = 3
    }
  ]
}
```

## Argument Reference

### Required

- `user_name` (String) The login username for the user. Must be unique within your AlertOps account.
- `first_name` (String) The user's first name.
- `last_name` (String) The user's last name.
- `locale` (String) The user's locale (e.g., `en-US`).
- `time_zone` (String) The user's time zone in AlertOps format (e.g., `(UTC-05:00) Eastern Time (US & Canada)`).
- `type` (String) The user type. Valid values: `Standard`, `Stakeholder`.
- `roles` (List of String) Roles assigned to the user (e.g., `["App Admin"]`).
- `contact_methods` (List of Object) One or more contact methods. At least one `Email-Official` entry is required. See [contact_methods](#nestedblock--contact_methods) below.

### Optional

- `external_id` (String) An optional external identifier for the user, for correlation with external systems.
- `extra_attributes` (Map of String) Additional fields to merge into the API request payload. JSON objects and arrays are automatically parsed. If the API rejects a field, the raw error is returned.

### `contact_methods`

Each entry supports the following attributes:

- `contact_method_name` (String, Required) The contact method type. Names are defined per AlertOps account; the defaults are `Email-Official`, `Email-Personal`, `Phone-Official`, `Phone-Official-Mobile`, `Phone-Personal`, `Phone-Personal-Mobile`, `SMS-Official`, `SMS-Personal` and `Push Notification`. Any other name is rejected with `Invalid Contact Method Type`.
- `email_address` (String, Optional) Email address. Required when the contact method name indicates email.
- `country_code` (String, Optional) Country dialing code for phone/SMS contact methods (e.g., `1` for US/Canada).
- `phone_number` (String, Optional) Phone number digits (no dashes or spaces) for phone/SMS contact methods.
- `phone_extension` (String, Optional) Phone extension for voice contact methods.
- `wait_time_in_mins` (Number, Optional) Minutes to wait before attempting this contact method. Defaults to `0`.
- `repeat` (Boolean, Optional) Whether to retry notifications on this contact method. Defaults to `false`.
- `repeat_times` (Number, Optional) Number of retry attempts. Defaults to `0`.
- `repeat_minutes` (Number, Optional) Minutes between retry attempts. Defaults to `0`.
- `notification_time24x7` (Boolean, Optional) If `true`, notifications are sent at any time of day. Defaults to `true`.
- `enabled` (Boolean, Optional) Whether this contact method is active. Defaults to `true`.
- `sequence` (Number, Optional) The order in which contact methods are attempted (lower = first). Defaults to `1`.

## Attributes Reference

In addition to the arguments above, the following computed attributes are exported:

- `id` (String) The unique numeric identifier of the user, assigned by AlertOps.

## Import

Users can be imported using their numeric AlertOps user ID:

```shell
terraform import alertops_user.example 12345
```
