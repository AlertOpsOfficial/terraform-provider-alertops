package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &OutboundIntegrationResource{}
	_ resource.ResourceWithConfigure   = &OutboundIntegrationResource{}
	_ resource.ResourceWithImportState = &OutboundIntegrationResource{}
)

func NewOutboundIntegrationResource() resource.Resource {
	return &OutboundIntegrationResource{}
}

type OutboundIntegrationResource struct {
	client *Client
}

type outboundIntegrationModel struct {
	ID                      types.String `tfsdk:"id"`
	OutboundIntegrationName types.String `tfsdk:"outbound_integration_name"`
	SecurityType            types.String `tfsdk:"security_type"`
	IntervalInSec           types.Int64  `tfsdk:"interval_in_sec"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	ExtraAttributes         types.Map    `tfsdk:"extra_attributes"`
}

func (r *OutboundIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_outbound_integration"
}

func (r *OutboundIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps outbound integration. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`. Complex nested fields (e.g., `security`, `methods`) can be passed as JSON strings in `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the outbound integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"outbound_integration_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the outbound integration.",
			},
			"security_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The security type for the outbound integration (e.g., `None`, `Basic`, `ApiKey`).",
			},
			"interval_in_sec": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "The polling interval in seconds for the outbound integration.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the outbound integration is enabled.",
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned. Use this for complex nested fields like `security`, `methods`, etc.",
			},
		},
	}
}

func (r *OutboundIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildOutboundPayload(ctx context.Context, plan *outboundIntegrationModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"outbound_integration_name": plan.OutboundIntegrationName.ValueString(),
	}
	if v := plan.SecurityType.ValueString(); v != "" {
		payload["security_type"] = v
	}
	if !plan.IntervalInSec.IsNull() && !plan.IntervalInSec.IsUnknown() {
		payload["interval_in_sec"] = plan.IntervalInSec.ValueInt64()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		payload["enabled"] = plan.Enabled.ValueBool()
	}
	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func parseOutboundFromAPI(result map[string]interface{}, state *outboundIntegrationModel) {
	if v, ok := result["outbound_integration_name"].(string); ok {
		state.OutboundIntegrationName = types.StringValue(v)
	}
	if v, ok := result["security_type"].(string); ok {
		state.SecurityType = types.StringValue(v)
	} else {
		state.SecurityType = types.StringValue("")
	}
	if v, ok := result["interval_in_sec"].(float64); ok {
		state.IntervalInSec = types.Int64Value(int64(v))
	}
	if v, ok := result["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(v)
	} else {
		// API may not return enabled field; default to true
		state.Enabled = types.BoolValue(true)
	}
}

func (r *OutboundIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan outboundIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildOutboundPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building outbound integration payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/integrations/outbound", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating outbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing outbound integration response", err.Error())
		return
	}

	if id, ok := result["outbound_integration_id"].(float64); ok {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(id)))
	} else if idStr, ok := result["outbound_integration_id"].(string); ok {
		plan.ID = types.StringValue(idStr)
	} else {
		resp.Diagnostics.AddError("Error creating outbound integration",
			fmt.Sprintf("Could not get outbound_integration_id from response: %s", string(respBody)))
		return
	}

	parseOutboundFromAPI(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OutboundIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state outboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/integrations/outbound/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading outbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing outbound integration response", err.Error())
		return
	}

	parseOutboundFromAPI(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OutboundIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan outboundIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state outboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildOutboundPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building outbound integration payload", err.Error())
		return
	}

	respBody, err := r.client.Put(ctx, "/integrations/outbound/"+state.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating outbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing outbound integration response", err.Error())
		return
	}

	plan.ID = state.ID
	parseOutboundFromAPI(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OutboundIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state outboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/integrations/outbound/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting outbound integration", err.Error())
		return
	}
}

func (r *OutboundIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
