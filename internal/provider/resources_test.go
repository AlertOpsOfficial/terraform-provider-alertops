package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Shapes below were captured from the AlertOps v2 API.
const responsePlayJSON = `{
  "response_play_id": 42811, "response_play_name": "TF Test RP API 1", "description": "TF Test",
  "priority": "High", "enabled": true, "repeat": true, "repeat_times": 2, "repeat_minutes": 5,
  "members": [
    {"member_id": 175861, "group_id": 28756, "group_name": "Adi_Test", "member_type": "Group", "sequence": 2, "wait_time_in_mins": 25},
    {"member_id": 175860, "user_id": 72302, "first_name": "A", "last_name": "V", "member_type": "User", "sequence": 1, "wait_time_in_mins": 0}
  ],
  "group_contact_notifications": {"contact_methods": [{"contact_method_name": "Group Email", "wait_time_in_mins": 0, "to_bcc_or_cc": "CC"}]},
  "workflows": [{"workflow_id": 32875, "workflow_name": "Stop Delivery on Assignment or Closed"}],
  "outbound_integrations": [], "outbound_actions": [],
  "options": {"alert_type_id": 22188, "alert_type_name": "Standard Alert", "acknowledgement": null,
    "assignment": {"phone": true, "sms": true, "email": true, "group_chat": true},
    "escalate": {"phone": true, "sms": true, "email": true, "group_chat": true},
    "close": {"phone": true, "sms": false, "email": true, "group_chat": true},
    "notification_settings": {"email": "Email Using Long Text", "phone": "VOIP", "sms": "SMS"},
    "escalation_policy_name_for_reply": null, "sla_in_hours": 6.0, "message_text": "",
    "include_alert_id_in_subject": true, "one_email_per_message": false, "one_message_per_recipient": false, "sequence_group_first": true},
  "is_used": false
}`

const workflowJSON = `{
  "workflow_id": 41482, "workflow_name": "TF Test WF", "workflow_type_id": 2, "workflow_type_name": "Alert",
  "enabled": false, "alert_type": "Standard Alert", "alert_type_id": 22188, "scheduled": false, "recurrence_interval": 0,
  "conditions": [{"condition_id": 84890, "type": "start", "match": "all", "name": "AlertStatus", "operator": "is", "value": "Closed", "list_id": null}],
  "actions": [{"action_id": 42258, "name": "SendMessage", "value": "TF Test RP 1", "webhook_url": null,
    "send_to_original_recipients": false, "send_to_sender": false, "send_to_owner": false, "launch_new_thread": false,
    "subject": "TF Test - alert closed", "message_text": "<p>closed</p>", "users": [], "groups": ["Ops"], "is_escalation_policy": false}]
}`

func mustDecode(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestParseResponsePlay(t *testing.T) {
	var state responsePlayModel
	parseResponsePlayFromAPI(mustDecode(t, responsePlayJSON), &state)

	if state.ResponsePlayName.ValueString() != "TF Test RP API 1" || !state.Repeat.ValueBool() || state.RepeatTimes.ValueInt64() != 2 {
		t.Fatalf("basic fields not parsed: %+v", state)
	}

	members := state.Members.Elements()
	if len(members) != 2 {
		t.Fatalf("got %d members", len(members))
	}
	first := members[0].(types.Object).Attributes()
	if first["type"].(types.String).ValueString() != "User" || first["id"].(types.String).ValueString() != "72302" {
		t.Errorf("members not ordered by sequence: %v", first)
	}
	second := members[1].(types.Object).Attributes()
	if second["id"].(types.String).ValueString() != "28756" || second["wait_time_in_mins"].(types.Int64).ValueInt64() != 25 {
		t.Errorf("group member wrong: %v", second)
	}

	if got := int64sFromSet(state.Workflows); len(got) != 1 || got[0] != 32875 {
		t.Errorf("workflows = %v", got)
	}

	opts := state.Options.Attributes()
	if opts["sla_in_hours"].(types.Int64).ValueInt64() != 6 {
		t.Errorf("sla_in_hours = %v", opts["sla_in_hours"])
	}
	ack := opts["acknowledgement"].(types.Object).Attributes()
	if ack["email"].(types.Bool).ValueBool() {
		t.Error("null acknowledgement should parse as all channels disabled")
	}
	if opts["escalation_policy_name_for_reply"].(types.String).ValueString() != "" {
		t.Error("null escalation_policy_name_for_reply should parse as empty string")
	}
}

func TestBuildOptionsPayloadOverlaysKnownPlanValues(t *testing.T) {
	current := mapObject(mustDecode(t, responsePlayJSON), "options")

	plan := rpOptionsFromAPI(current)
	attrs := plan.Attributes()
	values := map[string]attr.Value{}
	for k, v := range attrs {
		values[k] = v
	}
	values["sla_in_hours"] = types.Int64Value(8)
	values["message_text"] = types.StringUnknown()
	values["close"] = types.ObjectValueMust(rpChannelsAttrTypes, map[string]attr.Value{
		"phone": types.BoolValue(false), "sms": types.BoolUnknown(), "email": types.BoolValue(true), "group_chat": types.BoolValue(true),
	})
	planObj := types.ObjectValueMust(rpOptionsAttrTypes, values)

	payload := buildOptionsPayload(current, planObj)
	if payload["sla_in_hours"] != int64(8) {
		t.Errorf("sla_in_hours = %v", payload["sla_in_hours"])
	}
	if payload["alert_type"] != "22188" {
		t.Errorf("alert_type = %v", payload["alert_type"])
	}
	closeChannels := payload["close"].(map[string]interface{})
	if closeChannels["phone"] != false || closeChannels["sms"] != false {
		t.Errorf("close channels = %v", closeChannels)
	}
	ack := payload["acknowledgement"].(map[string]interface{})
	if ack["email"] != false {
		t.Errorf("acknowledgement should be sent as booleans: %v", ack)
	}
	if payload["notification_settings"].(map[string]interface{})["phone"] != "VOIP" {
		t.Errorf("notification_settings = %v", payload["notification_settings"])
	}
}

func TestParseWorkflowAndCompareItems(t *testing.T) {
	var state workflowModel
	result := mustDecode(t, workflowJSON)
	parseWorkflowFromAPI(result, &state)

	if state.WorkflowType.ValueString() != "Alert" || state.AlertType.ValueString() != "Standard Alert" {
		t.Fatalf("type fields: %v %v", state.WorkflowType, state.AlertType)
	}
	cond := state.Conditions.Elements()[0].(types.Object).Attributes()
	if cond["list_id"].(types.String).ValueString() != "" || cond["value"].(types.String).ValueString() != "Closed" {
		t.Errorf("condition = %v", cond)
	}
	action := state.Actions.Elements()[0].(types.Object)
	if action.Attributes()["webhook_url"].(types.String).ValueString() != "" {
		t.Error("null webhook_url should parse as empty string")
	}

	// A planned action with is_escalation_policy still unknown matches the API's action.
	values := map[string]attr.Value{}
	for k, v := range action.Attributes() {
		values[k] = v
	}
	values["is_escalation_policy"] = types.BoolUnknown()
	planned := []attr.Value{types.ObjectValueMust(wfActionAttrTypes, values)}
	current := []types.Object{actionFromAPI(mapArray(result, "actions")[0].(map[string]interface{}))}
	if !itemsEqual(planned, current) {
		t.Error("items with only unknown differences should be equal")
	}
	values["subject"] = types.StringValue("changed")
	if itemsEqual([]attr.Value{types.ObjectValueMust(wfActionAttrTypes, values)}, current) {
		t.Error("changed subject should not be equal")
	}

	payload := actionPayload(action)
	if _, ok := payload["webhook_url"]; ok {
		t.Error("empty webhook_url should be omitted")
	}
	if payload["is_escalation_policy"] != false {
		t.Errorf("is_escalation_policy = %v", payload["is_escalation_policy"])
	}
}

func TestInboundPayloadDefaultsAPISettings(t *testing.T) {
	ctx := context.Background()
	plan := inboundIntegrationModel{
		InboundIntegrationName: types.StringValue("TF Test"),
		Type:                   types.StringValue("API"),
		Sequence:               types.Int64Value(0),
		Enabled:                types.BoolValue(true),
		EscalationPolicy:       types.StringUnknown(),
		ResponsePlay:           types.StringValue("42810"),
		RecipientGroups:        types.ListNull(types.StringType),
		RecipientUsers:         types.ListNull(types.StringType),
		InboundTemplateID:      types.Int64Value(0),
		ExtraAttributes:        types.MapNull(types.StringType),
	}
	payload, err := buildInboundPayload(ctx, &plan)
	if err != nil {
		t.Fatal(err)
	}
	if payload["response_play"] != "42810" {
		t.Errorf("response_play = %v", payload["response_play"])
	}
	if _, ok := payload["escalation_policy"]; ok {
		t.Error("escalation_policy must not be sent with a response play")
	}
	if _, ok := payload["api_settings"]; !ok {
		t.Error("API integrations need default api_settings")
	}

	plan.ExtraAttributes = types.MapValueMust(types.StringType, map[string]attr.Value{
		"api_settings": types.StringValue(`{"url_mapping":{"source":"NewRelic"}}`),
	})
	payload, _ = buildInboundPayload(ctx, &plan)
	if !strings.Contains(mustJSON(t, payload["api_settings"]), "NewRelic") {
		t.Errorf("extra_attributes api_settings should win: %v", payload["api_settings"])
	}
}

func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestReconcileWorkflowAttachments(t *testing.T) {
	var calls []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls = append(calls, r.Method+" "+string(b))
		w.WriteHeader(http.StatusNoContent)
	})

	if err := reconcileWorkflowAttachments(context.Background(), c, "/response_plays/1", []int64{32875, 7}, []int64{7, 9}); err != nil {
		t.Fatal(err)
	}
	want := []string{`DELETE [{"workflow_id":32875}]`, `POST [{"workflow_id":9}]`}
	if strings.Join(calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestResourceSchemasAreValid(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "alertops"}, &meta)

		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: %v", meta.TypeName, resp.Diagnostics)
		}
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: %v", meta.TypeName, diags)
		}
	}
}
