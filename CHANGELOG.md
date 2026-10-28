# Changelog

## 2.1.0 (unreleased)

### New resources

- `alertops_response_play` — response plays with an ordered escalation path of users and groups, per-tier wait times, group contact methods, repeat settings, communication options and attached workflows. ([#5](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/5))
- `alertops_workflow` — workflows with start conditions and actions such as `SendMessage` (e.g. notify responders when an alert is acknowledged or closed). ([#6](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/6))
- `alertops_topic` — topics and their group subscriptions. ([#1](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/1))

### Enhancements

- `alertops_inbound_integration`: new `response_play` argument (conflicts with `escalation_policy`) and `response_play_name` attribute. ([#5](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/5))
- `alertops_inbound_integration`: new sensitive `api_url` attribute exposing the generated endpoint URL. ([#4](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/4))
- `alertops_escalation_policy`: new `workflows` argument to attach workflows. ([#6](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/6))
- Provider: client-side rate limiting matching the AlertOps API limits (5/second, 15/minute) with automatic retry and backoff on HTTP 429. New optional `base_url`, `requests_per_second` and `requests_per_minute` settings.
- Provider: transient HTTP 5xx responses are retried (up to 3 times, with backoff) for GET, PUT and DELETE requests. POST requests are never retried, so nothing is created twice.
- Project licensed under the Mozilla Public License 2.0. ([#2](https://github.com/AlertOpsOfficial/terraform-provider-alertops/issues/2))

### Bug fixes

- All resources: a resource deleted outside Terraform is now removed from state (and re-created on the next apply) instead of failing the refresh with a 404 error. Deleting an already-deleted resource succeeds.
- `alertops_inbound_integration`: updates of `API` integrations no longer fail with `URLMapping > Source cannot be null or empty`; basic fields are updated through the `/basic` endpoint and existing `api_settings` are preserved. State is refreshed from the API after updates.
- `alertops_inbound_integration`: creating an `API` integration without `api_settings` no longer fails; the default "Custom" URL mapping is sent.
- `alertops_inbound_integration`: destroying several integrations at once no longer fails intermittently with HTTP 500 (the API's transient errors are now retried).
- `alertops_user`: creating or updating a user no longer fails with "Provider returned invalid result object after apply" for contact method fields not set in configuration (e.g. `country_code`); the user is read back after every write. Reading a user no longer panics if the API omits a field.
- `alertops_user`: `time_zone` and `locale` are now applied on create. The AlertOps API ignores them when creating a user (the account defaults are used), so the provider sets them with a follow-up update; previously any user outside the account's default time zone or locale failed with "Provider produced inconsistent result after apply".
- `alertops_schedule`: the schedule is read back after an update, so values AlertOps derives (weekdays outside the `start_weekday`–`end_weekday` range are ignored) are reported at apply time instead of as drift on the next plan.
- `alertops_schedule`: importing a schedule now sets `group` (looked up from the group ID), so `terraform plan -generate-config-out` produces valid configuration.
- `alertops_maintenance_window`: updates no longer fail with "Provider produced inconsistent result after apply" (every attribute was blanked); the window is read back after the update.
- `alertops_outbound_integration`: the integration is read back after an update, and importing one no longer plans a change to `interval_in_sec`.

### Documentation

- Examples corrected to match what the API accepts: default contact method names (`Phone-Official`, `SMS-Official`), `to_bcc_or_cc` for email contact methods and `notify_using_centralized_settings = true` for per-role retries in escalation policies, `start_weekday`/`end_weekday` for schedules, rotation settings for rotating schedules, dates and enabled integrations for maintenance windows, `mail_box` for Email integrations, and the HTTP method for `Send Webhook` workflow actions.
- New example suites under `examples/` (`regression`, `full-stack`, `tf-test`) and a "Testing" section in the README. Removed the reference to non-existent acceptance tests.

### Build

- `go.mod` is now tidy (the release workflow runs `go mod tidy`); the minimum Go version is 1.25.8, required by the documentation tooling.
- New `Test` GitHub Actions workflow: tidy check, formatting, build, vet and unit tests on every push and pull request.
