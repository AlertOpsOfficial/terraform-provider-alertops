package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &EscalationPolicyResource{}
	_ resource.ResourceWithConfigure   = &EscalationPolicyResource{}
	_ resource.ResourceWithImportState = &EscalationPolicyResource{}
)

func NewEscalationPolicyResource() resource.Resource {
	return &EscalationPolicyResource{}
}

type EscalationPolicyResource struct {
	client *Client
}

var epContactMethodAttrTypes = map[string]attr.Type{
	"contact_method_name": types.StringType,
	"wait_time_in_mins":   types.Int64Type,
	"repeat":              types.BoolType,
	"repeat_times":        types.Int64Type,
	"repeat_minutes":      types.Int64Type,
	"to_bcc_or_cc":        types.StringType,
	"sequence":            types.Int64Type,
}

var memberRoleAttrTypes = map[string]attr.Type{
	"member_role_type":                  types.StringType,
	"wait_time_between_members_in_mins": types.Int64Type,
	"role_wait_time_in_mins":            types.Int64Type,
	"no_of_retries":                     types.Int64Type,
	"retry_interval":                    types.Int64Type,
	"contact_methods":                   types.ListType{ElemType: types.ObjectType{AttrTypes: epContactMethodAttrTypes}},
}

var epOptionsAttrTypes = map[string]attr.Type{
	"sla_in_hours":                types.Int64Type,
	"include_alert_id_in_subject": types.BoolType,
	"one_email_per_message":       types.BoolType,
	"one_message_per_recipient":   types.BoolType,
	"sequence_group_first":        types.BoolType,
}

type escalationPolicyModel struct {
	ID                                    types.String `tfsdk:"id"`
	EscalationPolicyName                  types.String `tfsdk:"escalation_policy_name"`
	Description                           types.String `tfsdk:"description"`
	Priority                              types.String `tfsdk:"priority"`
	Enabled                               types.Bool   `tfsdk:"enabled"`
	QuickLaunch                           types.Bool   `tfsdk:"quick_launch"`
	NotifyUsingCentralizedSettings        types.Bool   `tfsdk:"notify_using_centralized_settings"`
	WaitTimeBeforeNotifyingNextGroupInMin types.Int64  `tfsdk:"wait_time_before_notifying_next_group_in_min"`
	Options                               types.Object `tfsdk:"options"`
	MemberRoles                           types.List   `tfsdk:"member_roles"`
	Workflows                             types.Set    `tfsdk:"workflows"`
	ExtraAttributes                       types.Map    `tfsdk:"extra_attributes"`
}

func (r *EscalationPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation_policy"
}

func (r *EscalationPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps escalation policy. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the escalation policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"escalation_policy_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the escalation policy.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A description of the escalation policy.",
			},
			"priority": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The priority level (e.g., `Low`, `Medium`, `High`, `Critical`).",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the escalation policy is enabled.",
			},
			"quick_launch": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether quick launch is enabled for this policy.",
			},
			"notify_using_centralized_settings": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to use centralized notification settings.",
			},
			"wait_time_before_notifying_next_group_in_min": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "Wait time in minutes before notifying the next group.",
			},
			"options": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Options for the escalation policy.",
				Attributes: map[string]schema.Attribute{
					"sla_in_hours": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "SLA time in hours. The API enforces a minimum of 24 hours; setting 0 will result in 24 being stored.",
					},
					"include_alert_id_in_subject": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether to include the alert ID in the subject.",
					},
					"one_email_per_message": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether to send one email per message.",
					},
					"one_message_per_recipient": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether to send one message per recipient.",
					},
					"sequence_group_first": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether to sequence the group first.",
					},
				},
			},
			"member_roles": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of member roles in the escalation policy.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"member_role_type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The type of member role.",
						},
						"wait_time_between_members_in_mins": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Wait time in minutes between members.",
						},
						"role_wait_time_in_mins": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							MarkdownDescription: "Wait time in minutes for the role.",
						},
						"no_of_retries": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Number of retries.",
						},
						"retry_interval": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Retry interval in minutes.",
						},
						"contact_methods": schema.ListNestedAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Contact methods for this member role.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"contact_method_name": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The name of the contact method.",
									},
									"wait_time_in_mins": schema.Int64Attribute{
										Required:            true,
										MarkdownDescription: "Wait time in minutes.",
									},
									"repeat": schema.BoolAttribute{
										Optional:            true,
										Computed:            true,
										Default:             booldefault.StaticBool(false),
										MarkdownDescription: "Whether to repeat.",
									},
									"repeat_times": schema.Int64Attribute{
										Optional:            true,
										Computed:            true,
										Default:             int64default.StaticInt64(0),
										MarkdownDescription: "Number of times to repeat.",
									},
									"repeat_minutes": schema.Int64Attribute{
										Optional:            true,
										Computed:            true,
										Default:             int64default.StaticInt64(0),
										MarkdownDescription: "Minutes between repeats.",
									},
									"to_bcc_or_cc": schema.StringAttribute{
										Optional:            true,
										Computed:            true,
										MarkdownDescription: "BCC or CC setting.",
									},
									"sequence": schema.Int64Attribute{
										Required:            true,
										MarkdownDescription: "The sequence order.",
									},
								},
							},
						},
					},
				},
			},
			"workflows": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "IDs of the workflows attached to this escalation policy (e.g. `alertops_workflow.example.id`). When set, the provider attaches and detaches workflows so that exactly these are attached, including any default workflow AlertOps adds to new policies. Only enabled workflows can be attached. When omitted, attached workflows are left unmanaged.",
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned.",
			},
		},
	}
}

func (r *EscalationPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildEPPayload(ctx context.Context, plan *escalationPolicyModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"escalation_policy_name":            plan.EscalationPolicyName.ValueString(),
		"enabled":                           plan.Enabled.ValueBool(),
		"quick_launch":                      plan.QuickLaunch.ValueBool(),
		"notify_using_centralized_settings": plan.NotifyUsingCentralizedSettings.ValueBool(),
		"wait_time_before_notifying_next_group_in_min": plan.WaitTimeBeforeNotifyingNextGroupInMin.ValueInt64(),
	}
	if plan.Description.ValueString() != "" {
		payload["description"] = plan.Description.ValueString()
	}
	if plan.Priority.ValueString() != "" {
		payload["priority"] = plan.Priority.ValueString()
	}

	// Build options object
	if !plan.Options.IsNull() && !plan.Options.IsUnknown() {
		optAttrs := plan.Options.Attributes()
		options := map[string]interface{}{
			"sla_in_hours":                optAttrs["sla_in_hours"].(types.Int64).ValueInt64(),
			"include_alert_id_in_subject": optAttrs["include_alert_id_in_subject"].(types.Bool).ValueBool(),
			"one_email_per_message":       optAttrs["one_email_per_message"].(types.Bool).ValueBool(),
			"one_message_per_recipient":   optAttrs["one_message_per_recipient"].(types.Bool).ValueBool(),
			"sequence_group_first":        optAttrs["sequence_group_first"].(types.Bool).ValueBool(),
		}
		payload["options"] = options
	} else {
		// Always include default options to satisfy the API requirement
		payload["options"] = map[string]interface{}{
			"sla_in_hours":                0,
			"include_alert_id_in_subject": false,
			"one_email_per_message":       false,
			"one_message_per_recipient":   false,
			"sequence_group_first":        false,
		}
	}

	if !plan.MemberRoles.IsNull() && !plan.MemberRoles.IsUnknown() {
		var mrList []map[string]interface{}
		for _, elem := range plan.MemberRoles.Elements() {
			obj := elem.(types.Object)
			attrs := obj.Attributes()
			mr := map[string]interface{}{
				"wait_time_between_members_in_mins": attrs["wait_time_between_members_in_mins"].(types.Int64).ValueInt64(),
				"role_wait_time_in_mins":            attrs["role_wait_time_in_mins"].(types.Int64).ValueInt64(),
				"no_of_retries":                     attrs["no_of_retries"].(types.Int64).ValueInt64(),
				"retry_interval":                    attrs["retry_interval"].(types.Int64).ValueInt64(),
			}
			if v := attrs["member_role_type"].(types.String).ValueString(); v != "" {
				mr["member_role_type"] = v
			}

			cmList := attrs["contact_methods"].(types.List)
			if !cmList.IsNull() && !cmList.IsUnknown() {
				var cms []map[string]interface{}
				for _, cmElem := range cmList.Elements() {
					cmObj := cmElem.(types.Object)
					cmAttrs := cmObj.Attributes()
					cm := map[string]interface{}{
						"contact_method_name": cmAttrs["contact_method_name"].(types.String).ValueString(),
						"wait_time_in_mins":   cmAttrs["wait_time_in_mins"].(types.Int64).ValueInt64(),
						"repeat":              cmAttrs["repeat"].(types.Bool).ValueBool(),
						"repeat_times":        cmAttrs["repeat_times"].(types.Int64).ValueInt64(),
						"repeat_minutes":      cmAttrs["repeat_minutes"].(types.Int64).ValueInt64(),
						"sequence":            cmAttrs["sequence"].(types.Int64).ValueInt64(),
					}
					if v := cmAttrs["to_bcc_or_cc"].(types.String).ValueString(); v != "" {
						cm["to_bcc_or_cc"] = v
					}
					cms = append(cms, cm)
				}
				mr["contact_methods"] = cms
			}
			mrList = append(mrList, mr)
		}
		payload["member_roles"] = mrList
	}

	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func parseEPFromAPI(ctx context.Context, result map[string]interface{}, state *escalationPolicyModel) error {
	if v, ok := result["escalation_policy_name"].(string); ok {
		state.EscalationPolicyName = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		state.Description = types.StringValue(v)
	} else {
		state.Description = types.StringValue("")
	}
	if v, ok := result["priority"].(string); ok {
		state.Priority = types.StringValue(v)
	} else {
		state.Priority = types.StringValue("")
	}
	if v, ok := result["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(v)
	}
	if v, ok := result["quick_launch"].(bool); ok {
		state.QuickLaunch = types.BoolValue(v)
	}
	if v, ok := result["notify_using_centralized_settings"].(bool); ok {
		state.NotifyUsingCentralizedSettings = types.BoolValue(v)
	}
	if v, ok := result["wait_time_before_notifying_next_group_in_min"].(float64); ok {
		state.WaitTimeBeforeNotifyingNextGroupInMin = types.Int64Value(int64(v))
	}

	// Parse options
	if rawOpts, ok := result["options"].(map[string]interface{}); ok {
		getOptInt64 := func(key string) int64 {
			if v, ok := rawOpts[key].(float64); ok {
				return int64(v)
			}
			return 0
		}
		getOptBool := func(key string) bool {
			if v, ok := rawOpts[key].(bool); ok {
				return v
			}
			return false
		}
		optObj, diags := types.ObjectValue(epOptionsAttrTypes, map[string]attr.Value{
			"sla_in_hours":                types.Int64Value(getOptInt64("sla_in_hours")),
			"include_alert_id_in_subject": types.BoolValue(getOptBool("include_alert_id_in_subject")),
			"one_email_per_message":       types.BoolValue(getOptBool("one_email_per_message")),
			"one_message_per_recipient":   types.BoolValue(getOptBool("one_message_per_recipient")),
			"sequence_group_first":        types.BoolValue(getOptBool("sequence_group_first")),
		})
		if diags.HasError() {
			return fmt.Errorf("error building options object")
		}
		state.Options = optObj
	} else {
		optObj, diags := types.ObjectValue(epOptionsAttrTypes, map[string]attr.Value{
			"sla_in_hours":                types.Int64Value(0),
			"include_alert_id_in_subject": types.BoolValue(false),
			"one_email_per_message":       types.BoolValue(false),
			"one_message_per_recipient":   types.BoolValue(false),
			"sequence_group_first":        types.BoolValue(false),
		})
		if diags.HasError() {
			return fmt.Errorf("error building default options object")
		}
		state.Options = optObj
	}

	if rawMRs, ok := result["member_roles"].([]interface{}); ok {
		var mrObjs []attr.Value
		for _, item := range rawMRs {
			mr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			getInt64 := func(key string) int64 {
				if v, ok := mr[key].(float64); ok {
					return int64(v)
				}
				return 0
			}
			getString := func(key string) string {
				if v, ok := mr[key].(string); ok {
					return v
				}
				return ""
			}

			var cmObjs []attr.Value
			if rawCMs, ok := mr["contact_methods"].([]interface{}); ok {
				for _, cmItem := range rawCMs {
					cm, ok := cmItem.(map[string]interface{})
					if !ok {
						continue
					}
					getCMInt64 := func(key string) int64 {
						if v, ok := cm[key].(float64); ok {
							return int64(v)
						}
						return 0
					}
					getCMBool := func(key string) bool {
						if v, ok := cm[key].(bool); ok {
							return v
						}
						return false
					}
					getCMString := func(key string) string {
						if v, ok := cm[key].(string); ok {
							return v
						}
						return ""
					}
					cmObj, diags := types.ObjectValue(epContactMethodAttrTypes, map[string]attr.Value{
						"contact_method_name": types.StringValue(getCMString("contact_method_name")),
						"wait_time_in_mins":   types.Int64Value(getCMInt64("wait_time_in_mins")),
						"repeat":              types.BoolValue(getCMBool("repeat")),
						"repeat_times":        types.Int64Value(getCMInt64("repeat_times")),
						"repeat_minutes":      types.Int64Value(getCMInt64("repeat_minutes")),
						"to_bcc_or_cc":        types.StringValue(getCMString("to_bcc_or_cc")),
						"sequence":            types.Int64Value(getCMInt64("sequence")),
					})
					if diags.HasError() {
						return fmt.Errorf("error building EP contact method object")
					}
					cmObjs = append(cmObjs, cmObj)
				}
			}
			cmList, diags := types.ListValue(types.ObjectType{AttrTypes: epContactMethodAttrTypes}, cmObjs)
			if diags.HasError() {
				cmList = types.ListValueMust(types.ObjectType{AttrTypes: epContactMethodAttrTypes}, []attr.Value{})
			}

			mrObj, diags := types.ObjectValue(memberRoleAttrTypes, map[string]attr.Value{
				"member_role_type":                  types.StringValue(getString("member_role_type")),
				"wait_time_between_members_in_mins": types.Int64Value(getInt64("wait_time_between_members_in_mins")),
				"role_wait_time_in_mins":            types.Int64Value(getInt64("role_wait_time_in_mins")),
				"no_of_retries":                     types.Int64Value(getInt64("no_of_retries")),
				"retry_interval":                    types.Int64Value(getInt64("retry_interval")),
				"contact_methods":                   cmList,
			})
			if diags.HasError() {
				return fmt.Errorf("error building member role object")
			}
			mrObjs = append(mrObjs, mrObj)
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: memberRoleAttrTypes}, mrObjs)
		if diags.HasError() {
			return fmt.Errorf("error building member roles list")
		}
		state.MemberRoles = list
	} else {
		state.MemberRoles = types.ListValueMust(types.ObjectType{AttrTypes: memberRoleAttrTypes}, []attr.Value{})
	}

	state.Workflows = int64Set(workflowIDsFromAPI(mapArray(result, "workflows")))

	return nil
}

// syncWorkflows reconciles attached workflows when they are configured and returns the
// refreshed policy if anything changed.
func (r *EscalationPolicyResource) syncWorkflows(ctx context.Context, id string, desired types.Set, result map[string]interface{}) (map[string]interface{}, error) {
	if !isKnown(desired) {
		return result, nil
	}
	basePath := "/escalation_policies/" + id
	if _, ok := result["workflows"]; !ok {
		body, err := r.client.Get(ctx, basePath)
		if err != nil {
			return nil, err
		}
		if result, err = decodeObject(body); err != nil {
			return nil, err
		}
	}
	current := workflowIDsFromAPI(mapArray(result, "workflows"))
	if int64Set(current).Equal(int64Set(int64sFromSet(desired))) {
		return result, nil
	}
	if err := reconcileWorkflowAttachments(ctx, r.client, basePath, current, int64sFromSet(desired)); err != nil {
		return nil, err
	}
	body, err := r.client.Get(ctx, basePath)
	if err != nil {
		return nil, err
	}
	return decodeObject(body)
}

func (r *EscalationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan escalationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildEPPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building escalation policy payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/escalation_policies", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating escalation policy", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing escalation policy response", err.Error())
		return
	}

	if id, ok := result["escalation_policy_id"].(float64); ok {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(id)))
	} else if idStr, ok := result["escalation_policy_id"].(string); ok {
		plan.ID = types.StringValue(idStr)
	} else {
		resp.Diagnostics.AddError("Error creating escalation policy",
			fmt.Sprintf("Could not get escalation_policy_id from response: %s", string(respBody)))
		return
	}

	result, err = r.syncWorkflows(ctx, plan.ID.ValueString(), plan.Workflows, result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating escalation policy workflows", err.Error())
		return
	}

	if err := parseEPFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing escalation policy response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EscalationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/escalation_policies/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading escalation policy", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing escalation policy response", err.Error())
		return
	}

	if err := parseEPFromAPI(ctx, result, &state); err != nil {
		resp.Diagnostics.AddError("Error processing escalation policy response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EscalationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan escalationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	// Fetch current state from API to get notification_settings (required for update)
	currentBody, err := r.client.Get(ctx, "/escalation_policies/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading escalation policy before update", err.Error())
		return
	}
	var currentResult map[string]interface{}
	if err := json.Unmarshal(currentBody, &currentResult); err != nil {
		resp.Diagnostics.AddError("Error parsing escalation policy current state", err.Error())
		return
	}

	payload, err := buildEPPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building escalation policy payload", err.Error())
		return
	}

	// Merge notification_settings from current API state into the options payload
	// The API requires notification_settings to be present in the options object
	if currentOpts, ok := currentResult["options"].(map[string]interface{}); ok {
		if notifSettings, ok := currentOpts["notification_settings"]; ok {
			if opts, ok := payload["options"].(map[string]interface{}); ok {
				opts["notification_settings"] = notifSettings
			}
		}
	}

	respBody, err := r.client.Put(ctx, "/escalation_policies/"+state.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating escalation policy", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing escalation policy response", err.Error())
		return
	}

	// If the response is empty (204), re-fetch the current state
	if len(result) == 0 {
		respBody, err = r.client.Get(ctx, "/escalation_policies/"+state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading escalation policy after update", err.Error())
			return
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			resp.Diagnostics.AddError("Error parsing escalation policy response after update", err.Error())
			return
		}
	}

	result, err = r.syncWorkflows(ctx, state.ID.ValueString(), plan.Workflows, result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating escalation policy workflows", err.Error())
		return
	}

	if err := parseEPFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing escalation policy response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EscalationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/escalation_policies/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting escalation policy", err.Error())
		return
	}
}

func (r *EscalationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
