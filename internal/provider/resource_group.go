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
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithConfigure   = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
)

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

type GroupResource struct {
	client *Client
}

var groupMemberAttrTypes = map[string]attr.Type{
	"member_type": types.StringType,
	"member":      types.StringType,
	"sequence":    types.Int64Type,
	"roles":       types.ListType{ElemType: types.StringType},
}

var groupContactMethodAttrTypes = map[string]attr.Type{
	"contact_method_name": types.StringType,
	"email_address":       types.StringType,
	"country_code":        types.StringType,
	"phone_number":        types.StringType,
	"extension":           types.StringType,
	"url":                 types.StringType,
	"get_alert_update":    types.BoolType,
	"enabled":             types.BoolType,
	"sequence":            types.Int64Type,
}

type groupModel struct {
	ID              types.String `tfsdk:"id"`
	GroupName       types.String `tfsdk:"group_name"`
	Dynamic         types.Bool   `tfsdk:"dynamic"`
	Description     types.List   `tfsdk:"description"`
	Members         types.List   `tfsdk:"members"`
	ContactMethods  types.List   `tfsdk:"contact_methods"`
	Topics          types.List   `tfsdk:"topics"`
	ExtraAttributes types.Map    `tfsdk:"extra_attributes"`
}

func (r *GroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps group. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the group.",
			},
			"dynamic": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the group is dynamic.",
			},
			"description": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of description strings for the group.",
			},
			"members": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of members in the group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"member_type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The type of member (`User` or `Group`).",
						},
						"member": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The username or group name of the member.",
						},
						"sequence": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1),
							MarkdownDescription: "The sequence order of the member.",
						},
						"roles": schema.ListAttribute{
							Optional:            true,
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Roles for this member (e.g., `[\"Primary\", \"Manager\"]`).",
						},
					},
				},
			},
			"contact_methods": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of contact methods for the group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"contact_method_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the contact method.",
						},
						"email_address": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Email address for email contact methods.",
						},
						"country_code": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Country code for phone/SMS contact methods.",
						},
						"phone_number": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Phone number for phone/SMS contact methods.",
						},
						"extension": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Phone extension.",
						},
						"url": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "URL for webhook-based contact methods (e.g., Slack).",
						},
						"get_alert_update": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Whether to receive alert updates.",
						},
						"enabled": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
							MarkdownDescription: "Whether this contact method is enabled.",
						},
						"sequence": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1),
							MarkdownDescription: "The sequence order of this contact method.",
						},
					},
				},
			},
			"topics": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of topic names associated with the group.",
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned.",
			},
		},
	}
}

func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildGroupPayload(ctx context.Context, plan *groupModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"group_name": plan.GroupName.ValueString(),
		"dynamic":    plan.Dynamic.ValueBool(),
	}

	// Description
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		var desc []string
		if diags := plan.Description.ElementsAs(ctx, &desc, false); diags.HasError() {
			return nil, fmt.Errorf("error reading description")
		}
		payload["description"] = desc
	}

	// Topics
	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		if diags := plan.Topics.ElementsAs(ctx, &topics, false); diags.HasError() {
			return nil, fmt.Errorf("error reading topics")
		}
		payload["topics"] = topics
	}

	// Members
	if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
		var members []map[string]interface{}
		for _, elem := range plan.Members.Elements() {
			obj := elem.(types.Object)
			attrs := obj.Attributes()
			m := map[string]interface{}{
				"member_type": attrs["member_type"].(types.String).ValueString(),
				"member":      attrs["member"].(types.String).ValueString(),
				"sequence":    attrs["sequence"].(types.Int64).ValueInt64(),
			}
			rolesList := attrs["roles"].(types.List)
			if !rolesList.IsNull() && !rolesList.IsUnknown() {
				var roles []string
				rolesList.ElementsAs(ctx, &roles, false)
				m["roles"] = roles
			}
			members = append(members, m)
		}
		payload["members"] = members
	}

	// Contact Methods
	if !plan.ContactMethods.IsNull() && !plan.ContactMethods.IsUnknown() {
		var cms []map[string]interface{}
		for _, elem := range plan.ContactMethods.Elements() {
			obj := elem.(types.Object)
			attrs := obj.Attributes()
			cm := map[string]interface{}{
				"contact_method_name": attrs["contact_method_name"].(types.String).ValueString(),
				"enabled":             attrs["enabled"].(types.Bool).ValueBool(),
				"sequence":            attrs["sequence"].(types.Int64).ValueInt64(),
				"get_alert_update":    attrs["get_alert_update"].(types.Bool).ValueBool(),
			}
			if v := attrs["email_address"].(types.String).ValueString(); v != "" {
				cm["email_address"] = v
			}
			if v := attrs["country_code"].(types.String).ValueString(); v != "" {
				cm["country_code"] = v
			}
			if v := attrs["phone_number"].(types.String).ValueString(); v != "" {
				cm["phone_number"] = v
			}
			if v := attrs["extension"].(types.String).ValueString(); v != "" {
				cm["extension"] = v
			}
			if v := attrs["url"].(types.String).ValueString(); v != "" {
				cm["url"] = v
			}
			cms = append(cms, cm)
		}
		payload["contact_methods"] = cms
	}

	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func parseGroupFromAPI(ctx context.Context, result map[string]interface{}, state *groupModel) error {
	if v, ok := result["group_name"].(string); ok {
		state.GroupName = types.StringValue(v)
	}
	if v, ok := result["dynamic"].(bool); ok {
		state.Dynamic = types.BoolValue(v)
	}

	// Description
	if rawDesc, ok := result["description"].([]interface{}); ok {
		var desc []string
		for _, d := range rawDesc {
			if s, ok := d.(string); ok {
				desc = append(desc, s)
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, desc)
		if diags.HasError() {
			return fmt.Errorf("error building description list")
		}
		state.Description = list
	} else {
		state.Description = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Topics
	if rawTopics, ok := result["topics"].([]interface{}); ok {
		var topics []string
		for _, t := range rawTopics {
			if s, ok := t.(string); ok {
				topics = append(topics, s)
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, topics)
		if diags.HasError() {
			return fmt.Errorf("error building topics list")
		}
		state.Topics = list
	} else {
		state.Topics = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Members
	if rawMembers, ok := result["members"].([]interface{}); ok {
		var memberObjs []attr.Value
		for _, item := range rawMembers {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			memberType := ""
			if v, ok := m["member_type"].(string); ok {
				memberType = v
			}
			memberName := ""
			if v, ok := m["member"].(string); ok {
				memberName = v
			}
			seq := int64(1)
			if v, ok := m["sequence"].(float64); ok {
				seq = int64(v)
			}
			var roleVals []attr.Value
			if rawRoles, ok := m["roles"].([]interface{}); ok {
				for _, r := range rawRoles {
					if s, ok := r.(string); ok {
						roleVals = append(roleVals, types.StringValue(s))
					}
				}
			}
			rolesList, diags := types.ListValue(types.StringType, roleVals)
			if diags.HasError() {
				rolesList = types.ListValueMust(types.StringType, []attr.Value{})
			}
			obj, diags := types.ObjectValue(groupMemberAttrTypes, map[string]attr.Value{
				"member_type": types.StringValue(memberType),
				"member":      types.StringValue(memberName),
				"sequence":    types.Int64Value(seq),
				"roles":       rolesList,
			})
			if diags.HasError() {
				return fmt.Errorf("error building member object")
			}
			memberObjs = append(memberObjs, obj)
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: groupMemberAttrTypes}, memberObjs)
		if diags.HasError() {
			return fmt.Errorf("error building members list")
		}
		state.Members = list
	} else {
		state.Members = types.ListValueMust(types.ObjectType{AttrTypes: groupMemberAttrTypes}, []attr.Value{})
	}

	// Contact Methods
	if rawCMs, ok := result["contact_methods"].([]interface{}); ok {
		var cmObjs []attr.Value
		for _, item := range rawCMs {
			cm, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			getString := func(key string) string {
				if v, ok := cm[key].(string); ok {
					return v
				}
				return ""
			}
			getBool := func(key string) bool {
				if v, ok := cm[key].(bool); ok {
					return v
				}
				return false
			}
			getInt64 := func(key string) int64 {
				if v, ok := cm[key].(float64); ok {
					return int64(v)
				}
				return 0
			}
			obj, diags := types.ObjectValue(groupContactMethodAttrTypes, map[string]attr.Value{
				"contact_method_name": types.StringValue(getString("contact_method_name")),
				"email_address":       types.StringValue(getString("email_address")),
				"country_code":        types.StringValue(getString("country_code")),
				"phone_number":        types.StringValue(getString("phone_number")),
				"extension":           types.StringValue(getString("extension")),
				"url":                 types.StringValue(getString("url")),
				"get_alert_update":    types.BoolValue(getBool("get_alert_update")),
				"enabled":             types.BoolValue(getBool("enabled")),
				"sequence":            types.Int64Value(getInt64("sequence")),
			})
			if diags.HasError() {
				return fmt.Errorf("error building group contact method object")
			}
			cmObjs = append(cmObjs, obj)
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: groupContactMethodAttrTypes}, cmObjs)
		if diags.HasError() {
			return fmt.Errorf("error building group contact methods list")
		}
		state.ContactMethods = list
	} else {
		state.ContactMethods = types.ListValueMust(types.ObjectType{AttrTypes: groupContactMethodAttrTypes}, []attr.Value{})
	}

	return nil
}

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildGroupPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building group payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/groups", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing group response", err.Error())
		return
	}

	groupID, ok := result["group_id"].(float64)
	if !ok {
		// Try string ID
		if idStr, ok := result["group_id"].(string); ok {
			plan.ID = types.StringValue(idStr)
		} else {
			resp.Diagnostics.AddError("Error creating group", fmt.Sprintf("Could not get group_id from response: %s", string(respBody)))
			return
		}
	} else {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(groupID)))
	}

	if err := parseGroupFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing group response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/groups/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing group response", err.Error())
		return
	}

	if err := parseGroupFromAPI(ctx, result, &state); err != nil {
		resp.Diagnostics.AddError("Error processing group response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state groupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildGroupPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building group payload", err.Error())
		return
	}

	// PUT returns 204 No Content, so we must GET after update
	_, err = r.client.Put(ctx, "/groups/"+state.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group", err.Error())
		return
	}

	// Refresh state via GET
	respBody, err := r.client.Get(ctx, "/groups/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading group after update", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing group response", err.Error())
		return
	}

	plan.ID = state.ID
	if err := parseGroupFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing group response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/groups/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting group", err.Error())
		return
	}
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
