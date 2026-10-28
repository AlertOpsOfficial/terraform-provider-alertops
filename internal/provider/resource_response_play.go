package provider

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ResponsePlayResource{}
	_ resource.ResourceWithConfigure   = &ResponsePlayResource{}
	_ resource.ResourceWithImportState = &ResponsePlayResource{}
)

func NewResponsePlayResource() resource.Resource {
	return &ResponsePlayResource{}
}

type ResponsePlayResource struct {
	client *Client
}

var rpMemberAttrTypes = map[string]attr.Type{
	"type":              types.StringType,
	"id":                types.StringType,
	"wait_time_in_mins": types.Int64Type,
}

var rpContactMethodAttrTypes = map[string]attr.Type{
	"contact_method_name": types.StringType,
	"to_bcc_or_cc":        types.StringType,
	"wait_time_in_mins":   types.Int64Type,
}

var rpChannelsAttrTypes = map[string]attr.Type{
	"phone":      types.BoolType,
	"sms":        types.BoolType,
	"email":      types.BoolType,
	"group_chat": types.BoolType,
}

var rpChannelEvents = []string{"acknowledgement", "assignment", "escalate", "close"}

var rpOptionsAttrTypes = map[string]attr.Type{
	"alert_type_id":                    types.Int64Type,
	"alert_type_name":                  types.StringType,
	"sla_in_hours":                     types.Int64Type,
	"message_text":                     types.StringType,
	"include_alert_id_in_subject":      types.BoolType,
	"one_email_per_message":            types.BoolType,
	"one_message_per_recipient":        types.BoolType,
	"sequence_group_first":             types.BoolType,
	"email_notification_setting":       types.StringType,
	"phone_notification_setting":       types.StringType,
	"sms_notification_setting":         types.StringType,
	"escalation_policy_name_for_reply": types.StringType,
	"acknowledgement":                  types.ObjectType{AttrTypes: rpChannelsAttrTypes},
	"assignment":                       types.ObjectType{AttrTypes: rpChannelsAttrTypes},
	"escalate":                         types.ObjectType{AttrTypes: rpChannelsAttrTypes},
	"close":                            types.ObjectType{AttrTypes: rpChannelsAttrTypes},
}

type responsePlayModel struct {
	ID                  types.String `tfsdk:"id"`
	ResponsePlayName    types.String `tfsdk:"response_play_name"`
	Description         types.String `tfsdk:"description"`
	Priority            types.String `tfsdk:"priority"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Repeat              types.Bool   `tfsdk:"repeat"`
	RepeatTimes         types.Int64  `tfsdk:"repeat_times"`
	RepeatMinutes       types.Int64  `tfsdk:"repeat_minutes"`
	Members             types.List   `tfsdk:"members"`
	GroupContactMethods types.List   `tfsdk:"group_contact_methods"`
	Workflows           types.Set    `tfsdk:"workflows"`
	Options             types.Object `tfsdk:"options"`
}

func (r *ResponsePlayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_response_play"
}

func channelsSchema(description string) schema.SingleNestedAttribute {
	boolAttr := func(d string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: d,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	return schema.SingleNestedAttribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: description,
		PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Attributes: map[string]schema.Attribute{
			"phone":      boolAttr("Notify by phone (VOIP)."),
			"sms":        boolAttr("Notify by SMS."),
			"email":      boolAttr("Notify by email."),
			"group_chat": boolAttr("Notify group chat channels."),
		},
	}
}

func (r *ResponsePlayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps response play: an escalation path of users and groups with custom wait times, tied to schedules rather than user roles. Reference it from `alertops_inbound_integration.response_play`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the response play.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"response_play_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the response play.",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A description of the response play.",
			},
			"priority": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The priority (e.g., `No Priority`, `Low`, `Medium`, `High`, `Critical`).",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the response play is enabled. Workflows can only send messages through enabled response plays.",
			},
			"repeat": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to repeat the whole escalation path when nobody responds.",
			},
			"repeat_times": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "How many times to repeat the escalation path.",
			},
			"repeat_minutes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "Minutes to wait before repeating the escalation path.",
			},
			"members": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The escalation path, in order. Each entry is one tier: the first member is notified first, then each following member after its `wait_time_in_mins`. When omitted, members are left unmanaged.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "`User` or `Group`.",
							Validators:          []validator.String{oneOfStringValidator{values: []string{"User", "Group"}}},
						},
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The ID of the user or group (e.g. `alertops_user.example.id`).",
						},
						"wait_time_in_mins": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							MarkdownDescription: "Minutes to wait after the previous tier before notifying this member.",
						},
					},
				},
			},
			"group_contact_methods": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Group contact methods (e.g. `Group Email`, `Slack Group`, `Microsoft Teams`) notified for groups in the escalation path. When omitted, they are left unmanaged.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"contact_method_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The group contact method name.",
						},
						"to_bcc_or_cc": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "For email methods: `To`, `CC` or `BCC`.",
						},
						"wait_time_in_mins": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							MarkdownDescription: "Minutes to wait before using this contact method.",
						},
					},
				},
			},
			"workflows": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				ElementType:         types.Int64Type,
				MarkdownDescription: "IDs of the workflows attached to this response play. When set, the provider attaches and detaches workflows so that exactly these are attached, including the default workflow AlertOps adds to new response plays. Only enabled workflows can be attached. When omitted, attached workflows are left unmanaged.",
			},
			"options": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Communication options. Any option not set keeps the value AlertOps assigns.",
				Attributes: map[string]schema.Attribute{
					"alert_type_id": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						MarkdownDescription: "The ID of the alert type used by this response play.",
					},
					"alert_type_name": schema.StringAttribute{
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "The name of the alert type used by this response play.",
					},
					"sla_in_hours": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						MarkdownDescription: "SLA in hours.",
					},
					"message_text": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Additional message text included in notifications.",
					},
					"include_alert_id_in_subject": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Whether to include the alert ID in the subject.",
					},
					"one_email_per_message": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Whether to send one email per message.",
					},
					"one_message_per_recipient": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Whether to send one message per recipient.",
					},
					"sequence_group_first": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Whether to sequence the group first.",
					},
					"email_notification_setting": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Email notification setting (e.g. `Email Using Long Text`).",
					},
					"phone_notification_setting": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Phone notification setting (e.g. `VOIP`).",
					},
					"sms_notification_setting": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "SMS notification setting (e.g. `SMS`).",
					},
					"escalation_policy_name_for_reply": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						MarkdownDescription: "Escalation policy used for replies. Empty means the system default.",
					},
					"acknowledgement": channelsSchema("Channels notified when the alert is acknowledged."),
					"assignment":      channelsSchema("Channels notified when the alert is assigned."),
					"escalate":        channelsSchema("Channels notified when the alert escalates."),
					"close":           channelsSchema("Channels notified when the alert is closed."),
				},
			},
		},
	}
}

func (r *ResponsePlayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

// ---------------------------------------------------------------------------
// API <-> model conversion
// ---------------------------------------------------------------------------

type rpMember struct {
	Type     string
	ID       string
	WaitTime int64
}

type rpContactMethod struct {
	Name      string
	ToBccOrCc string
	WaitTime  int64
	toKnown   bool
}

func rpMembersFromAPI(result map[string]interface{}) []rpMember {
	raw := mapArray(result, "members")
	type seqMember struct {
		seq int64
		m   rpMember
	}
	var items []seqMember
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		member := rpMember{Type: mapString(m, "member_type"), WaitTime: mapInt64(m, "wait_time_in_mins")}
		if member.Type == "Group" {
			member.ID, _ = idFromResult(m, "group_id")
		} else {
			member.ID, _ = idFromResult(m, "user_id")
		}
		items = append(items, seqMember{seq: mapInt64(m, "sequence"), m: member})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].seq < items[j].seq })
	out := make([]rpMember, 0, len(items))
	for _, it := range items {
		out = append(out, it.m)
	}
	return out
}

func rpContactMethodsFromAPI(result map[string]interface{}) []rpContactMethod {
	var out []rpContactMethod
	for _, item := range mapArray(mapObject(result, "group_contact_notifications"), "contact_methods") {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, rpContactMethod{
				Name:      mapString(m, "contact_method_name"),
				ToBccOrCc: mapString(m, "to_bcc_or_cc"),
				WaitTime:  mapInt64(m, "wait_time_in_mins"),
				toKnown:   true,
			})
		}
	}
	return out
}

func rpMembersFromPlan(list types.List) ([]rpMember, error) {
	var out []rpMember
	for _, elem := range list.Elements() {
		attrs := elem.(types.Object).Attributes()
		m := rpMember{
			Type:     attrs["type"].(types.String).ValueString(),
			ID:       attrs["id"].(types.String).ValueString(),
			WaitTime: attrs["wait_time_in_mins"].(types.Int64).ValueInt64(),
		}
		if _, err := parseID(m.ID); err != nil {
			return nil, fmt.Errorf("member %d: %w", len(out)+1, err)
		}
		out = append(out, m)
	}
	return out, nil
}

func rpContactMethodsFromPlan(list types.List) []rpContactMethod {
	var out []rpContactMethod
	for _, elem := range list.Elements() {
		attrs := elem.(types.Object).Attributes()
		to := attrs["to_bcc_or_cc"].(types.String)
		out = append(out, rpContactMethod{
			Name:      attrs["contact_method_name"].(types.String).ValueString(),
			ToBccOrCc: to.ValueString(),
			WaitTime:  attrs["wait_time_in_mins"].(types.Int64).ValueInt64(),
			toKnown:   isKnown(to),
		})
	}
	return out
}

func membersEqual(a, b []rpMember) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func channelsFromAPI(raw map[string]interface{}) types.Object {
	// The API returns null for an event with every channel disabled.
	return types.ObjectValueMust(rpChannelsAttrTypes, map[string]attr.Value{
		"phone":      types.BoolValue(mapBool(raw, "phone")),
		"sms":        types.BoolValue(mapBool(raw, "sms")),
		"email":      types.BoolValue(mapBool(raw, "email")),
		"group_chat": types.BoolValue(mapBool(raw, "group_chat")),
	})
}

func rpOptionsFromAPI(raw map[string]interface{}) types.Object {
	settings := mapObject(raw, "notification_settings")
	slaHours := int64(0)
	if v, ok := raw["sla_in_hours"].(float64); ok {
		slaHours = int64(v)
	}
	values := map[string]attr.Value{
		"alert_type_id":                    types.Int64Value(mapInt64(raw, "alert_type_id")),
		"alert_type_name":                  types.StringValue(mapString(raw, "alert_type_name")),
		"sla_in_hours":                     types.Int64Value(slaHours),
		"message_text":                     types.StringValue(mapString(raw, "message_text")),
		"include_alert_id_in_subject":      types.BoolValue(mapBool(raw, "include_alert_id_in_subject")),
		"one_email_per_message":            types.BoolValue(mapBool(raw, "one_email_per_message")),
		"one_message_per_recipient":        types.BoolValue(mapBool(raw, "one_message_per_recipient")),
		"sequence_group_first":             types.BoolValue(mapBool(raw, "sequence_group_first")),
		"email_notification_setting":       types.StringValue(mapString(settings, "email")),
		"phone_notification_setting":       types.StringValue(mapString(settings, "phone")),
		"sms_notification_setting":         types.StringValue(mapString(settings, "sms")),
		"escalation_policy_name_for_reply": types.StringValue(mapString(raw, "escalation_policy_name_for_reply")),
	}
	for _, event := range rpChannelEvents {
		values[event] = channelsFromAPI(mapObject(raw, event))
	}
	return types.ObjectValueMust(rpOptionsAttrTypes, values)
}

func parseResponsePlayFromAPI(result map[string]interface{}, state *responsePlayModel) {
	state.ResponsePlayName = types.StringValue(mapString(result, "response_play_name"))
	state.Description = types.StringValue(mapString(result, "description"))
	state.Priority = types.StringValue(mapString(result, "priority"))
	state.Enabled = types.BoolValue(mapBool(result, "enabled"))
	state.Repeat = types.BoolValue(mapBool(result, "repeat"))
	state.RepeatTimes = types.Int64Value(mapInt64(result, "repeat_times"))
	state.RepeatMinutes = types.Int64Value(mapInt64(result, "repeat_minutes"))

	members := []attr.Value{}
	for _, m := range rpMembersFromAPI(result) {
		members = append(members, types.ObjectValueMust(rpMemberAttrTypes, map[string]attr.Value{
			"type":              types.StringValue(m.Type),
			"id":                types.StringValue(m.ID),
			"wait_time_in_mins": types.Int64Value(m.WaitTime),
		}))
	}
	state.Members = types.ListValueMust(types.ObjectType{AttrTypes: rpMemberAttrTypes}, members)

	cms := []attr.Value{}
	for _, cm := range rpContactMethodsFromAPI(result) {
		cms = append(cms, types.ObjectValueMust(rpContactMethodAttrTypes, map[string]attr.Value{
			"contact_method_name": types.StringValue(cm.Name),
			"to_bcc_or_cc":        types.StringValue(cm.ToBccOrCc),
			"wait_time_in_mins":   types.Int64Value(cm.WaitTime),
		}))
	}
	state.GroupContactMethods = types.ListValueMust(types.ObjectType{AttrTypes: rpContactMethodAttrTypes}, cms)

	state.Workflows = int64Set(workflowIDsFromAPI(mapArray(result, "workflows")))
	state.Options = rpOptionsFromAPI(mapObject(result, "options"))
}

// buildOptionsPayload starts from the current API options and overlays every option known in the plan.
func buildOptionsPayload(current map[string]interface{}, plan types.Object) map[string]interface{} {
	settings := mapObject(current, "notification_settings")
	payload := map[string]interface{}{
		"escalation_policy_name_for_reply": mapString(current, "escalation_policy_name_for_reply"),
		"sla_in_hours":                     mapInt64(current, "sla_in_hours"),
		"message_text":                     mapString(current, "message_text"),
		"include_alert_id_in_subject":      mapBool(current, "include_alert_id_in_subject"),
		"one_email_per_message":            mapBool(current, "one_email_per_message"),
		"one_message_per_recipient":        mapBool(current, "one_message_per_recipient"),
		"sequence_group_first":             mapBool(current, "sequence_group_first"),
		"alert_type":                       strconv.FormatInt(mapInt64(current, "alert_type_id"), 10),
	}
	if v, ok := current["sla_in_hours"].(float64); ok {
		payload["sla_in_hours"] = int64(v)
	}
	notif := map[string]interface{}{
		"email": mapString(settings, "email"),
		"phone": mapString(settings, "phone"),
		"sms":   mapString(settings, "sms"),
	}
	payload["notification_settings"] = notif
	for _, event := range rpChannelEvents {
		raw := mapObject(current, event)
		payload[event] = map[string]interface{}{
			"phone":      mapBool(raw, "phone"),
			"sms":        mapBool(raw, "sms"),
			"email":      mapBool(raw, "email"),
			"group_chat": mapBool(raw, "group_chat"),
		}
	}

	if !isKnown(plan) {
		return payload
	}
	attrs := plan.Attributes()
	setBool := func(key string) {
		if v, ok := attrs[key].(types.Bool); ok && isKnown(v) {
			payload[key] = v.ValueBool()
		}
	}
	setString := func(key string) {
		if v, ok := attrs[key].(types.String); ok && isKnown(v) {
			payload[key] = v.ValueString()
		}
	}
	if v, ok := attrs["sla_in_hours"].(types.Int64); ok && isKnown(v) {
		payload["sla_in_hours"] = v.ValueInt64()
	}
	if v, ok := attrs["alert_type_id"].(types.Int64); ok && isKnown(v) {
		payload["alert_type"] = strconv.FormatInt(v.ValueInt64(), 10)
	}
	setString("message_text")
	setString("escalation_policy_name_for_reply")
	setBool("include_alert_id_in_subject")
	setBool("one_email_per_message")
	setBool("one_message_per_recipient")
	setBool("sequence_group_first")
	for key, apiKey := range map[string]string{
		"email_notification_setting": "email",
		"phone_notification_setting": "phone",
		"sms_notification_setting":   "sms",
	} {
		if v, ok := attrs[key].(types.String); ok && isKnown(v) {
			notif[apiKey] = v.ValueString()
		}
	}
	for _, event := range rpChannelEvents {
		obj, ok := attrs[event].(types.Object)
		if !ok || !isKnown(obj) {
			continue
		}
		channels := payload[event].(map[string]interface{})
		for name, v := range obj.Attributes() {
			if b, ok := v.(types.Bool); ok && isKnown(b) {
				channels[name] = b.ValueBool()
			}
		}
	}
	return payload
}

// ---------------------------------------------------------------------------
// Sync helpers
// ---------------------------------------------------------------------------

func (r *ResponsePlayResource) get(ctx context.Context, id string) (map[string]interface{}, error) {
	body, err := r.client.Get(ctx, "/response_plays/"+id)
	if err != nil {
		return nil, err
	}
	return decodeObject(body)
}

func (r *ResponsePlayResource) syncMembers(ctx context.Context, id string, plan types.List, current map[string]interface{}) error {
	if !isKnown(plan) {
		return nil
	}
	desired, err := rpMembersFromPlan(plan)
	if err != nil {
		return err
	}
	existing := rpMembersFromAPI(current)
	if membersEqual(existing, desired) {
		return nil
	}

	// Rebuild the escalation path: tiers are ordered by sequence, so replacing them is simplest.
	for _, item := range mapArray(current, "members") {
		if m, ok := item.(map[string]interface{}); ok {
			memberID, _ := idFromResult(m, "member_id")
			if _, err := r.client.Delete(ctx, fmt.Sprintf("/response_plays/%s/members/%s", id, memberID)); err != nil && !IsNotFound(err) {
				return fmt.Errorf("error removing member %s: %w", memberID, err)
			}
		}
	}
	if len(desired) == 0 {
		return nil
	}
	payload := make([]map[string]interface{}, 0, len(desired))
	for i, m := range desired {
		numericID, _ := parseID(m.ID)
		item := map[string]interface{}{
			"member_type":       m.Type,
			"sequence":          i + 1,
			"wait_time_in_mins": m.WaitTime,
		}
		if m.Type == "Group" {
			item["group_id"] = numericID
		} else {
			item["user_id"] = numericID
		}
		payload = append(payload, item)
	}
	if _, err := r.client.Post(ctx, "/response_plays/"+id+"/members", payload); err != nil {
		return fmt.Errorf("error adding members: %w", err)
	}
	return nil
}

func (r *ResponsePlayResource) syncContactMethods(ctx context.Context, id string, plan types.List, current map[string]interface{}) error {
	if !isKnown(plan) {
		return nil
	}
	base := "/response_plays/" + id + "/group_contact_notifications/contact_methods"
	existing := map[string]rpContactMethod{}
	for _, cm := range rpContactMethodsFromAPI(current) {
		existing[cm.Name] = cm
	}
	desired := rpContactMethodsFromPlan(plan)
	want := map[string]bool{}
	for _, cm := range desired {
		want[cm.Name] = true
	}
	for name := range existing {
		if !want[name] {
			if _, err := r.client.Delete(ctx, base+"/"+url.PathEscape(name)); err != nil && !IsNotFound(err) {
				return fmt.Errorf("error removing group contact method %q: %w", name, err)
			}
		}
	}
	for _, cm := range desired {
		body := map[string]interface{}{
			"contact_method_name": cm.Name,
			"wait_time_in_mins":   cm.WaitTime,
		}
		if cm.toKnown {
			body["to_bcc_or_cc"] = cm.ToBccOrCc
		}
		old, ok := existing[cm.Name]
		switch {
		case !ok:
			if _, err := r.client.Post(ctx, base, body); err != nil {
				return fmt.Errorf("error adding group contact method %q: %w", cm.Name, err)
			}
		case old.WaitTime != cm.WaitTime || (cm.toKnown && old.ToBccOrCc != cm.ToBccOrCc):
			if !cm.toKnown {
				body["to_bcc_or_cc"] = old.ToBccOrCc
			}
			if _, err := r.client.Put(ctx, base+"/"+url.PathEscape(cm.Name), body); err != nil {
				return fmt.Errorf("error updating group contact method %q: %w", cm.Name, err)
			}
		}
	}
	return nil
}

// applySettings brings everything except the basic fields in line with the plan.
func (r *ResponsePlayResource) applySettings(ctx context.Context, id string, plan *responsePlayModel, current map[string]interface{}) error {
	if err := r.syncMembers(ctx, id, plan.Members, current); err != nil {
		return err
	}
	if err := r.syncContactMethods(ctx, id, plan.GroupContactMethods, current); err != nil {
		return err
	}

	if plan.Repeat.ValueBool() != mapBool(current, "repeat") ||
		plan.RepeatTimes.ValueInt64() != mapInt64(current, "repeat_times") ||
		plan.RepeatMinutes.ValueInt64() != mapInt64(current, "repeat_minutes") {
		body := map[string]interface{}{
			"repeat":         plan.Repeat.ValueBool(),
			"repeat_times":   plan.RepeatTimes.ValueInt64(),
			"repeat_minutes": plan.RepeatMinutes.ValueInt64(),
		}
		if _, err := r.client.Put(ctx, "/response_plays/"+id+"/repeat", body); err != nil {
			return fmt.Errorf("error updating repeat settings: %w", err)
		}
	}

	if isKnown(plan.Options) {
		currentOptions := mapObject(current, "options")
		if !rpOptionsFromAPI(currentOptions).Equal(plan.Options) {
			if _, err := r.client.Put(ctx, "/response_plays/"+id+"/options", buildOptionsPayload(currentOptions, plan.Options)); err != nil {
				return fmt.Errorf("error updating options: %w", err)
			}
		}
	}

	if isKnown(plan.Workflows) {
		existing := workflowIDsFromAPI(mapArray(current, "workflows"))
		if err := reconcileWorkflowAttachments(ctx, r.client, "/response_plays/"+id, existing, int64sFromSet(plan.Workflows)); err != nil {
			return err
		}
	}
	return nil
}

func basicResponsePlayPayload(plan *responsePlayModel) map[string]interface{} {
	return map[string]interface{}{
		"response_play_name": plan.ResponsePlayName.ValueString(),
		"description":        plan.Description.ValueString(),
		"priority":           plan.Priority.ValueString(),
		"enabled":            plan.Enabled.ValueBool(),
	}
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *ResponsePlayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan responsePlayModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Post(ctx, "/response_plays/basic", basicResponsePlayPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating response play", err.Error())
		return
	}
	current, err := decodeObject(respBody)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing response play response", err.Error())
		return
	}
	id, ok := idFromResult(current, "response_play_id")
	if !ok {
		resp.Diagnostics.AddError("Error creating response play",
			fmt.Sprintf("Could not get response_play_id from response: %s", string(respBody)))
		return
	}
	plan.ID = types.StringValue(id)
	// Save the ID immediately so a failure below does not orphan the response play.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	if err := r.applySettings(ctx, id, &plan, current); err != nil {
		resp.Diagnostics.AddError("Error configuring response play", err.Error())
		return
	}

	result, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading response play after create", err.Error())
		return
	}
	parseResponsePlayFromAPI(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResponsePlayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state responsePlayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.get(ctx, state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading response play", err.Error())
		return
	}
	parseResponsePlayFromAPI(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResponsePlayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state responsePlayModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	id := state.ID.ValueString()

	current, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading response play before update", err.Error())
		return
	}

	if plan.ResponsePlayName.ValueString() != mapString(current, "response_play_name") ||
		plan.Description.ValueString() != mapString(current, "description") ||
		plan.Priority.ValueString() != mapString(current, "priority") ||
		plan.Enabled.ValueBool() != mapBool(current, "enabled") {
		if _, err := r.client.Put(ctx, "/response_plays/"+id+"/basic", basicResponsePlayPayload(&plan)); err != nil {
			resp.Diagnostics.AddError("Error updating response play", err.Error())
			return
		}
	}

	if err := r.applySettings(ctx, id, &plan, current); err != nil {
		resp.Diagnostics.AddError("Error updating response play", err.Error())
		return
	}

	result, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading response play after update", err.Error())
		return
	}
	parseResponsePlayFromAPI(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResponsePlayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state responsePlayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Delete(ctx, "/response_plays/"+state.ID.ValueString()); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting response play", err.Error())
	}
}

func (r *ResponsePlayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
