package provider

import (
	"context"
	"encoding/json"
	"fmt"

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
	_ resource.Resource                = &InboundIntegrationResource{}
	_ resource.ResourceWithConfigure   = &InboundIntegrationResource{}
	_ resource.ResourceWithImportState = &InboundIntegrationResource{}
)

func NewInboundIntegrationResource() resource.Resource {
	return &InboundIntegrationResource{}
}

type InboundIntegrationResource struct {
	client *Client
}

type inboundIntegrationModel struct {
	ID                     types.String `tfsdk:"id"`
	InboundIntegrationName types.String `tfsdk:"inbound_integration_name"`
	Type                   types.String `tfsdk:"type"`
	Sequence               types.Int64  `tfsdk:"sequence"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	EscalationPolicy       types.String `tfsdk:"escalation_policy"`
	RecipientGroups        types.List   `tfsdk:"recipient_groups"`
	RecipientUsers         types.List   `tfsdk:"recipient_users"`
	InboundTemplateID      types.Int64  `tfsdk:"inbound_template_id"`
	ExtraAttributes        types.Map    `tfsdk:"extra_attributes"`
}

func (r *InboundIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inbound_integration"
}

func (r *InboundIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps inbound integration. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`. Complex nested fields (e.g., `api_settings`, `email_settings`, `chat_settings`) can be passed as JSON strings in `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the inbound integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inbound_integration_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the inbound integration.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of inbound integration (e.g., `API`, `Email`, `Heartbeat`).",
			},
			"sequence": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				MarkdownDescription: "The sequence order of the integration.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the integration is enabled.",
			},
			"escalation_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the escalation policy to use for this integration.",
			},
			"recipient_groups": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of group names to receive alerts from this integration.",
			},
			"recipient_users": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of usernames to receive alerts from this integration.",
			},
			"inbound_template_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "The ID of the inbound template to use for this integration.",
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned. Use this for complex nested fields like `api_settings`, `email_settings`, etc.",
			},
		},
	}
}

func (r *InboundIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildInboundPayload(ctx context.Context, plan *inboundIntegrationModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"inbound_integration_name": plan.InboundIntegrationName.ValueString(),
		"type":                     plan.Type.ValueString(),
		"sequence":                 plan.Sequence.ValueInt64(),
		"enabled":                  plan.Enabled.ValueBool(),
		"inbound_template_id":      plan.InboundTemplateID.ValueInt64(),
	}
	if v := plan.EscalationPolicy.ValueString(); v != "" {
		payload["escalation_policy"] = v
	}
	if !plan.RecipientGroups.IsNull() && !plan.RecipientGroups.IsUnknown() {
		var groups []string
		if diags := plan.RecipientGroups.ElementsAs(ctx, &groups, false); diags.HasError() {
			return nil, fmt.Errorf("error reading recipient_groups")
		}
		payload["recipient_groups"] = groups
	}
	if !plan.RecipientUsers.IsNull() && !plan.RecipientUsers.IsUnknown() {
		var users []string
		if diags := plan.RecipientUsers.ElementsAs(ctx, &users, false); diags.HasError() {
			return nil, fmt.Errorf("error reading recipient_users")
		}
		payload["recipient_users"] = users
	}
	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func parseInboundFromAPI(ctx context.Context, result map[string]interface{}, state *inboundIntegrationModel) error {
	if v, ok := result["inbound_integration_name"].(string); ok {
		state.InboundIntegrationName = types.StringValue(v)
	}
	if v, ok := result["type"].(string); ok {
		state.Type = types.StringValue(v)
	}
	if v, ok := result["sequence"].(float64); ok {
		state.Sequence = types.Int64Value(int64(v))
	}
	if v, ok := result["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(v)
	}
	if v, ok := result["escalation_policy_name"].(string); ok {
		state.EscalationPolicy = types.StringValue(v)
	} else if v, ok := result["escalation_policy"].(string); ok {
		state.EscalationPolicy = types.StringValue(v)
	} else {
		state.EscalationPolicy = types.StringValue("")
	}
	if v, ok := result["inbound_template_id"].(float64); ok {
		state.InboundTemplateID = types.Int64Value(int64(v))
	}

	if rawGroups, ok := result["recipient_groups"].([]interface{}); ok {
		var groups []string
		for _, g := range rawGroups {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, groups)
		if diags.HasError() {
			return fmt.Errorf("error building recipient_groups list")
		}
		state.RecipientGroups = list
	} else {
		state.RecipientGroups = types.ListValueMust(types.StringType, nil)
	}

	if rawUsers, ok := result["recipient_users"].([]interface{}); ok {
		var users []string
		for _, u := range rawUsers {
			if s, ok := u.(string); ok {
				users = append(users, s)
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, users)
		if diags.HasError() {
			return fmt.Errorf("error building recipient_users list")
		}
		state.RecipientUsers = list
	} else {
		state.RecipientUsers = types.ListValueMust(types.StringType, nil)
	}

	return nil
}

func (r *InboundIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan inboundIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildInboundPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building inbound integration payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/integrations/inbound", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating inbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
		return
	}

	if id, ok := result["inbound_integration_id"].(float64); ok {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(id)))
	} else if idStr, ok := result["inbound_integration_id"].(string); ok {
		plan.ID = types.StringValue(idStr)
	} else {
		resp.Diagnostics.AddError("Error creating inbound integration",
			fmt.Sprintf("Could not get inbound_integration_id from response: %s", string(respBody)))
		return
	}

	if err := parseInboundFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing inbound integration response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InboundIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state inboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/integrations/inbound/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading inbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
		return
	}

	if err := parseInboundFromAPI(ctx, result, &state); err != nil {
		resp.Diagnostics.AddError("Error processing inbound integration response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InboundIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan inboundIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state inboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildInboundPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building inbound integration payload", err.Error())
		return
	}

	respBody, err := r.client.Put(ctx, "/integrations/inbound/"+state.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating inbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
		return
	}

	plan.ID = state.ID
	if err := parseInboundFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing inbound integration response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InboundIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state inboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/integrations/inbound/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting inbound integration", err.Error())
		return
	}
}

func (r *InboundIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
